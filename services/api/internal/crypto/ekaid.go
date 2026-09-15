package crypto

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// Crockford Base32 alphabet (32 characters)
// Excludes: 'I', 'L', 'O' (to avoid confusion with 1 and 0), and 'U' (to prevent accidental offensive words).
const CrockfordAlphabet = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

var (
	// EkaIDRegex matches format: EKA-XXXX-XXXX
	EkaIDRegex = regexp.MustCompile(`^EKA-[0-9A-HJKMNP-TV-Z]{4}-[0-9A-HJKMNP-TV-Z]{4}$`)
	
	ErrInvalidEkaIDFormat = errors.New("invalid EKA ID format: must follow EKA-XXXX-XXXX using Crockford Base32")
)

// GenerateEkaID generates a cryptographically secure, collision-resistant EKA ID
// Pattern: EKA-CCCC-CCCC
// 8 Crockford characters provide 32^8 ≈ 1.1 trillion distinct combinations.
func GenerateEkaID() (string, error) {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to read secure random bytes: %w", err)
	}

	alphabetLen := byte(len(CrockfordAlphabet)) // 32
	var sb strings.Builder
	sb.WriteString("EKA-")

	for i := 0; i < 8; i++ {
		if i == 4 {
			sb.WriteByte('-')
		}
		// Mask with 0x1F (31) to get uniform distribution across 32 symbols
		charIdx := bytes[i] & 0x1F
		sb.WriteByte(CrockfordAlphabet[charIdx%alphabetLen])
	}

	return sb.String(), nil
}

// ValidateEkaID verifies whether a string conforms to the public EKA ID specification
func ValidateEkaID(ekaID string) bool {
	clean := strings.TrimSpace(strings.ToUpper(ekaID))
	return EkaIDRegex.MatchString(clean)
}

// NormalizeEkaID cleans and capitalizes an EKA ID
func NormalizeEkaID(ekaID string) (string, error) {
	clean := strings.TrimSpace(strings.ToUpper(ekaID))
	if !ValidateEkaID(clean) {
		return "", ErrInvalidEkaIDFormat
	}
	return clean, nil
}

// --- Ed25519 Asymmetric Cryptography & DID JWK Primitives ---

// DeriveEd25519KeyPair derives a deterministic Ed25519 keypair from a seed/secret
func DeriveEd25519KeyPair(seedBytes []byte) (ed25519.PublicKey, ed25519.PrivateKey, error) {
	if len(seedBytes) == 0 {
		return ed25519.GenerateKey(rand.Reader)
	}
	// Use SHA-256 of seedBytes to ensure exact 32 bytes for Ed25519 seed
	hash := sha256.Sum256(seedBytes)
	priv := ed25519.NewKeyFromSeed(hash[:])
	pub := priv.Public().(ed25519.PublicKey)
	return pub, priv, nil
}

// SignEd25519 signs a message payload with an Ed25519 private key
func SignEd25519(priv ed25519.PrivateKey, message []byte) string {
	sig := ed25519.Sign(priv, message)
	return base64.RawURLEncoding.EncodeToString(sig)
}

// VerifyEd25519 verifies an Ed25519 signature against a public key
func VerifyEd25519(pub ed25519.PublicKey, message []byte, sigBase64 string) bool {
	sig, err := base64.RawURLEncoding.DecodeString(sigBase64)
	if err != nil || len(sig) != ed25519.SignatureSize {
		return false
	}
	return ed25519.Verify(pub, message, sig)
}

// PublicKeyToJWK converts an Ed25519 public key to W3C compliant JWK (RFC 8037)
func PublicKeyToJWK(pub ed25519.PublicKey, keyID string) map[string]interface{} {
	return map[string]interface{}{
		"kty": "OKP",
		"crv": "Ed25519",
		"x":   base64.RawURLEncoding.EncodeToString(pub),
		"kid": keyID,
		"use": "sig",
		"alg": "EdDSA",
	}
}
