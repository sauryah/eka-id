package middleware

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sauryah/eka-id/services/api/internal/service"
)

type contextKey string

const (
	RequestIDKey  contextKey = "request_id"
	UserClaimsKey contextKey = "user_claims"
)

// RequestID middleware assigns a unique UUID to every request
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := r.Header.Get("X-Request-ID")
		if reqID == "" {
			reqID = uuid.New().String()
		}
		w.Header().Set("X-Request-ID", reqID)
		ctx := context.WithValue(r.Context(), RequestIDKey, reqID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetRequestID retrieves request ID from context
func GetRequestID(ctx context.Context) string {
	if val, ok := ctx.Value(RequestIDKey).(string); ok {
		return val
	}
	return ""
}

// CORS middleware handles cross-origin requests
func CORS(allowedOrigins string) func(http.Handler) http.Handler {
	var originsList []string
	allowAll := false
	if allowedOrigins == "*" {
		allowAll = true
	} else {
		for _, o := range strings.Split(allowedOrigins, ",") {
			trimmed := strings.TrimSpace(o)
			if trimmed != "" {
				originsList = append(originsList, trimmed)
			}
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			originAllowed := false

			if allowAll {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			} else if origin != "" {
				for _, o := range originsList {
					if o == origin {
						originAllowed = true
						break
					}
				}
				if originAllowed {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Set("Access-Control-Allow-Credentials", "true")
				}
			}

			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, X-Request-ID")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// --- HYBRID DISTRIBUTED & IN-MEMORY RATE LIMITER ---

// RateLimiter defines the interface for sliding-window rate limiters
type RateLimiter interface {
	Allow(ctx context.Context, key string, limit int, window time.Duration) (allowed bool, remaining int, resetAfter time.Duration)
	Limit(limit int, window time.Duration) func(http.Handler) http.Handler
	LimitWithKey(keyFunc func(*http.Request) string, limit int, window time.Duration) func(http.Handler) http.Handler
	Ping(ctx context.Context) error
}

// MemoryRateLimiter provides thread-safe in-memory sliding window rate limiting
type MemoryRateLimiter struct {
	mu      sync.RWMutex
	clients map[string][]time.Time
}

func NewMemoryRateLimiter() *MemoryRateLimiter {
	return &MemoryRateLimiter{
		clients: make(map[string][]time.Time),
	}
}

func (m *MemoryRateLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, int, time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-window)

	var valid []time.Time
	for _, t := range m.clients[key] {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}

	if len(valid) >= limit {
		oldest := valid[0]
		resetAfter := window - now.Sub(oldest)
		if resetAfter < 0 {
			resetAfter = time.Second
		}
		m.clients[key] = valid
		return false, 0, resetAfter
	}

	valid = append(valid, now)
	m.clients[key] = valid
	remaining := limit - len(valid)
	if remaining < 0 {
		remaining = 0
	}
	return true, remaining, window
}

// HybridRateLimiter implements Redis-backed distributed rate limiting with seamless in-memory fallback
type HybridRateLimiter struct {
	redisHost     string
	redisPort     string
	redisPassword string
	memLimiter    *MemoryRateLimiter
	redisEnabled  bool
	redisMu       sync.Mutex
	redisConn     net.Conn
	redisReader   *bufio.Reader
}

func NewHybridRateLimiter(host, port, password string) *HybridRateLimiter {
	limiter := &HybridRateLimiter{
		redisHost:     host,
		redisPort:     port,
		redisPassword: password,
		memLimiter:    NewMemoryRateLimiter(),
		redisEnabled:  host != "" && port != "",
	}
	return limiter
}

// NewRateLimiter backward compatibility constructor
func NewRateLimiter(limit int, window time.Duration) *HybridRateLimiter {
	return NewHybridRateLimiter("", "", "")
}

func (h *HybridRateLimiter) Ping(ctx context.Context) error {
	if !h.redisEnabled {
		return fmt.Errorf("redis is disabled")
	}
	h.redisMu.Lock()
	defer h.redisMu.Unlock()

	conn, _, err := h.getRedisConn()
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(conn, "*1\r\n$4\r\nPING\r\n")
	if err != nil {
		h.closeRedisConn()
		return err
	}
	line, err := h.redisReader.ReadString('\n')
	if err != nil || !strings.Contains(line, "PONG") {
		h.closeRedisConn()
		return fmt.Errorf("invalid ping response: %s (%v)", line, err)
	}
	return nil
}

func (h *HybridRateLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, int, time.Duration) {
	if !h.redisEnabled {
		return h.memLimiter.Allow(ctx, key, limit, window)
	}

	// Attempt Redis sliding window log
	allowed, remaining, resetAfter, err := h.allowRedis(key, limit, window)
	if err != nil {
		// Gracefully fall back to memory limiter without blocking or failing
		return h.memLimiter.Allow(ctx, key, limit, window)
	}

	return allowed, remaining, resetAfter
}

func (h *HybridRateLimiter) allowRedis(key string, limit int, window time.Duration) (bool, int, time.Duration, error) {
	h.redisMu.Lock()
	defer h.redisMu.Unlock()

	conn, reader, err := h.getRedisConn()
	if err != nil {
		return false, 0, 0, err
	}
	_ = conn.SetDeadline(time.Now().Add(250 * time.Millisecond))

	nowMs := time.Now().UnixNano() / int64(time.Millisecond)
	windowMs := window.Milliseconds()
	cutoffMs := nowMs - windowMs
	member := fmt.Sprintf("%d-%s", nowMs, uuid.New().String()[:8])
	redisKey := fmt.Sprintf("ratelimit:%s", key)

	// Step 1: Remove timestamps older than cutoff
	remCmd := fmt.Sprintf("*4\r\n$17\r\nZREMRANGEBYSCORE\r\n$%d\r\n%s\r\n$4\r\n-inf\r\n$%d\r\n%d\r\n", len(redisKey), redisKey, len(strconv.FormatInt(cutoffMs, 10)), cutoffMs)
	// Step 2: Get current count
	cardCmd := fmt.Sprintf("*2\r\n$5\r\nZCARD\r\n$%d\r\n%s\r\n", len(redisKey), redisKey)

	_, err = conn.Write([]byte(remCmd + cardCmd))
	if err != nil {
		h.closeRedisConn()
		return false, 0, 0, err
	}

	// Read ZREMRANGEBYSCORE response
	_, err = reader.ReadString('\n')
	if err != nil {
		h.closeRedisConn()
		return false, 0, 0, err
	}

	// Read ZCARD response: :<count>\r\n
	cardLine, err := reader.ReadString('\n')
	if err != nil || !strings.HasPrefix(cardLine, ":") {
		h.closeRedisConn()
		return false, 0, 0, fmt.Errorf("unexpected zcard resp: %s", cardLine)
	}

	currentCount, _ := strconv.Atoi(strings.TrimSpace(cardLine[1:]))

	if currentCount >= limit {
		return false, 0, window, nil
	}

	// Step 3: Add new entry and set expire
	windowSec := int(window.Seconds()) + 5
	zaddCmd := fmt.Sprintf("*4\r\n$4\r\nZADD\r\n$%d\r\n%s\r\n$%d\r\n%d\r\n$%d\r\n%s\r\n", len(redisKey), redisKey, len(strconv.FormatInt(nowMs, 10)), nowMs, len(member), member)
	expCmd := fmt.Sprintf("*3\r\n$6\r\nEXPIRE\r\n$%d\r\n%s\r\n$%d\r\n%d\r\n", len(redisKey), redisKey, len(strconv.Itoa(windowSec)), windowSec)

	_, err = conn.Write([]byte(zaddCmd + expCmd))
	if err != nil {
		h.closeRedisConn()
		return false, 0, 0, err
	}

	// Read responses
	_, err1 := reader.ReadString('\n')
	_, err2 := reader.ReadString('\n')
	if err1 != nil || err2 != nil {
		h.closeRedisConn()
		return false, 0, 0, fmt.Errorf("failed to read redis responses")
	}

	remaining := limit - (currentCount + 1)
	if remaining < 0 {
		remaining = 0
	}
	return true, remaining, window, nil
}

func (h *HybridRateLimiter) getRedisConn() (net.Conn, *bufio.Reader, error) {
	if h.redisConn != nil {
		return h.redisConn, h.redisReader, nil
	}

	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%s", h.redisHost, h.redisPort), 200*time.Millisecond)
	if err != nil {
		return nil, nil, err
	}

	reader := bufio.NewReader(conn)
	if h.redisPassword != "" {
		_, err = fmt.Fprintf(conn, "*2\r\n$4\r\nAUTH\r\n$%d\r\n%s\r\n", len(h.redisPassword), h.redisPassword)
		if err != nil {
			_ = conn.Close()
			return nil, nil, err
		}
		_, err = reader.ReadString('\n')
		if err != nil {
			_ = conn.Close()
			return nil, nil, err
		}
	}

	h.redisConn = conn
	h.redisReader = reader
	return h.redisConn, h.redisReader, nil
}

func (h *HybridRateLimiter) closeRedisConn() {
	if h.redisConn != nil {
		_ = h.redisConn.Close()
		h.redisConn = nil
		h.redisReader = nil
	}
}

func isTrustedProxy(ip string) bool {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return false
	}
	if parsed.IsLoopback() {
		return true
	}
	if ip4 := parsed.To4(); ip4 != nil {
		if ip4[0] == 10 ||
			(ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31) ||
			(ip4[0] == 192 && ip4[1] == 168) {
			return true
		}
	}
	return false
}

func extractClientIP(r *http.Request) string {
	remoteIP := r.RemoteAddr
	if host, _, err := net.SplitHostPort(remoteIP); err == nil {
		remoteIP = host
	}

	if isTrustedProxy(remoteIP) {
		if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
			parts := strings.Split(forwarded, ",")
			clientIP := strings.TrimSpace(parts[0])
			if net.ParseIP(clientIP) != nil {
				return clientIP
			}
		}
		if realIP := strings.TrimSpace(r.Header.Get("X-Real-IP")); realIP != "" {
			if net.ParseIP(realIP) != nil {
				return realIP
			}
		}
	}

	return remoteIP
}

func (h *HybridRateLimiter) Limit(limit int, window time.Duration) func(http.Handler) http.Handler {
	return h.LimitWithKey(extractClientIP, limit, window)
}

func (h *HybridRateLimiter) LimitWithKey(keyFunc func(*http.Request) string, limit int, window time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := keyFunc(r)
			allowed, remaining, resetAfter := h.Allow(r.Context(), key, limit, window)

			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(limit))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
			w.Header().Set("X-RateLimit-Reset", strconv.Itoa(int(resetAfter.Seconds())))

			if !allowed {
				w.Header().Set("Retry-After", strconv.Itoa(int(resetAfter.Seconds())))
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"error": map[string]string{
						"code":    "RATE_LIMIT_EXCEEDED",
						"message": fmt.Sprintf("Too many requests. Rate limit is %d requests per %v. Please try again in %d seconds.", limit, window, int(resetAfter.Seconds())),
					},
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// AuthMiddleware validates JWT Bearer tokens
func AuthMiddleware(authSvc *service.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			var tokenString string
			if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
				tokenString = strings.TrimPrefix(authHeader, "Bearer ")
			} else if strings.HasSuffix(r.URL.Path, "/events/stream") {
				// Query param token is ONLY permitted on the real-time SSE stream endpoint
				// where browser EventSource API does not support custom Authorization headers.
				tokenString = r.URL.Query().Get("token")
			}

			if tokenString == "" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"error": map[string]string{
						"code":    "UNAUTHORIZED",
						"message": "Authorization header missing or malformed.",
					},
				})
				return
			}

			claims, err := authSvc.ValidateToken(tokenString)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"error": map[string]string{
						"code":    "INVALID_TOKEN",
						"message": "Token is invalid or expired.",
					},
				})
				return
			}

			ctx := context.WithValue(r.Context(), UserClaimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserClaims retrieves claims from context
func GetUserClaims(ctx context.Context) *service.JWTClaims {
	if val, ok := ctx.Value(UserClaimsKey).(*service.JWTClaims); ok {
		return val
	}
	return nil
}

// RequireRole checks if authenticated principal has one of the allowed roles
func RequireRole(allowedRoles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := GetUserClaims(r.Context())
			if claims == nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			authorized := false
			for _, role := range allowedRoles {
				if claims.Role == role {
					authorized = true
					break
				}
			}

			if !authorized {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"error": map[string]string{
						"code":    "FORBIDDEN",
						"message": "Insufficient permissions for this resource.",
					},
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}