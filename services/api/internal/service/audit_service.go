package service

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sauryah/eka-id/services/api/internal/domain"
	"github.com/sauryah/eka-id/services/api/internal/repository"
)

type AuditService struct {
	repo repository.AuditRepository
}

func NewAuditService(repo repository.AuditRepository) *AuditService {
	return &AuditService{repo: repo}
}

// Record emits an immutable audit record, stripping any sensitive keys
func (s *AuditService) Record(
	ctx context.Context,
	actorID *uuid.UUID,
	actorType string,
	action string,
	resourceType string,
	resourceID string,
	result string,
	ipAddress string,
	userAgent string,
	requestID string,
	metadata map[string]interface{},
) error {
	sanitizedMeta := s.sanitizeMetadata(metadata)

	event := &domain.AuditEvent{
		EventID:      uuid.New(),
		ActorID:      actorID,
		ActorType:    actorType,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Result:       result,
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		RequestID:    requestID,
		Metadata:     sanitizedMeta,
		CreatedAt:    time.Now().UTC(),
	}

	return s.repo.Record(ctx, event)
}

func (s *AuditService) ListEvents(ctx context.Context, limit, offset int, actorID *uuid.UUID, resourceID string) ([]*domain.AuditEvent, int, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	return s.repo.List(ctx, limit, offset, actorID, resourceID)
}

// sanitizeMetadata ensures passwords, tokens, OTPs, or private keys never leak into audit logs
func (s *AuditService) sanitizeMetadata(meta map[string]interface{}) map[string]interface{} {
	if meta == nil {
		return map[string]interface{}{}
	}
	sanitized := make(map[string]interface{}, len(meta))
	sensitiveKeys := []string{"password", "secret", "token", "otp", "code", "private_key", "pin"}

	for k, v := range meta {
		lowerKey := strings.ToLower(k)
		isSensitive := false
		for _, sens := range sensitiveKeys {
			if strings.Contains(lowerKey, sens) {
				isSensitive = true
				break
			}
		}
		if isSensitive {
			sanitized[k] = "[REDACTED]"
		} else {
			sanitized[k] = v
		}
	}
	return sanitized
}