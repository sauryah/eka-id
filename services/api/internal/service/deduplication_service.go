package service

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sauryah/eka-id/services/api/internal/domain"
	"github.com/sauryah/eka-id/services/api/internal/repository"
)

// DeduplicationService abstracts duplicate detection heuristics
type DeduplicationService struct {
	profRepo  repository.ProfileRepository
	dedupRepo repository.DuplicateRepository
	auditSvc  *AuditService
}

func NewDeduplicationService(
	profRepo repository.ProfileRepository,
	dedupRepo repository.DuplicateRepository,
	auditSvc *AuditService,
) *DeduplicationService {
	return &DeduplicationService{
		profRepo:  profRepo,
		dedupRepo: dedupRepo,
		auditSvc:  auditSvc,
	}
}

// EvaluateDuplicates checks if an incoming profile matches existing identities
// Rather than auto-merging, it calculates confidence and flags for administrative review
func (s *DeduplicationService) EvaluateDuplicates(ctx context.Context, identityID uuid.UUID, profile *domain.Profile) error {
	candidates, err := s.profRepo.FindPotentialDuplicates(ctx, profile.LegalName, profile.DateOfBirth, profile.Phone, profile.Email)
	if err != nil {
		return err
	}

	for _, cand := range candidates {
		if cand.IdentityID == identityID {
			continue // Skip self
		}

		score, reasons := s.calculateSimilarity(profile, cand)
		if score >= 70.0 {
			flag := &domain.DuplicateFlag{
				ID:                   uuid.New(),
				IdentityID:           identityID,
				SuspectedDuplicateID: cand.IdentityID,
				ConfidenceScore:      score,
				MatchReasons:         reasons,
				Status:               "PENDING_REVIEW",
				CreatedAt:            time.Now().UTC(),
			}

			_ = s.dedupRepo.CreateFlag(ctx, flag)

			_ = s.auditSvc.Record(ctx, nil, "SYSTEM", "POSSIBLE_DUPLICATE_DETECTED", "IDENTITY", identityID.String(), "FLAGGED", "", "", "", map[string]interface{}{
				"suspected_duplicate_id": cand.IdentityID.String(),
				"confidence_score":      score,
				"reasons":               reasons,
			})
		}
	}
	return nil
}

func (s *DeduplicationService) calculateSimilarity(p1, p2 *domain.Profile) (float64, []string) {
	var score float64
	var reasons []string

	if p1.Phone != "" && p2.Phone != "" && p1.Phone == p2.Phone {
		score += 50.0
		reasons = append(reasons, "Identical phone number match")
	}

	if p1.Email != "" && p2.Email != "" && strings.EqualFold(p1.Email, p2.Email) {
		score += 45.0
		reasons = append(reasons, "Identical email address match")
	}

	if strings.EqualFold(p1.LegalName, p2.LegalName) {
		score += 30.0
		reasons = append(reasons, "Exact legal name match")
	}

	if p1.DateOfBirth != "" && p2.DateOfBirth != "" && p1.DateOfBirth == p2.DateOfBirth {
		score += 25.0
		reasons = append(reasons, "Matching date of birth")
	}

	if score > 100.0 {
		score = 99.0
	}

	return score, reasons
}

func (s *DeduplicationService) ListPendingFlags(ctx context.Context) ([]*domain.DuplicateFlag, error) {
	return s.dedupRepo.ListPending(ctx)
}

func (s *DeduplicationService) ResolveFlag(ctx context.Context, flagID uuid.UUID, status string, reviewerID uuid.UUID) error {
	return s.dedupRepo.ResolveFlag(ctx, flagID, status, reviewerID)
}