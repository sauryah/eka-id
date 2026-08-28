package service

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sauryah/eka-id/services/api/internal/domain"
	"github.com/sauryah/eka-id/services/api/internal/repository"
)

// DeduplicationService abstracts duplicate detection heuristics including biometric face matching
type DeduplicationService struct {
	profRepo  repository.ProfileRepository
	dedupRepo repository.DuplicateRepository
	identRepo repository.IdentityRepository
	auditSvc  *AuditService
}

func NewDeduplicationService(
	profRepo repository.ProfileRepository,
	dedupRepo repository.DuplicateRepository,
	identRepo repository.IdentityRepository,
	auditSvc *AuditService,
) *DeduplicationService {
	return &DeduplicationService{
		profRepo:  profRepo,
		dedupRepo: dedupRepo,
		identRepo: identRepo,
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

// extractFaceEmbedding extracts []float64 from profile.Metadata["face_embedding"]
func extractFaceEmbedding(meta map[string]interface{}) []float64 {
	if meta == nil {
		return nil
	}
	raw, ok := meta["face_embedding"]
	if !ok || raw == nil {
		return nil
	}
	switch v := raw.(type) {
	case []float64:
		return v
	case []interface{}:
		res := make([]float64, len(v))
		for i, item := range v {
			if f, ok := item.(float64); ok {
				res[i] = f
			}
		}
		return res
	}
	return nil
}

// cosineSimilarity computes cosine similarity between two float vectors
func cosineSimilarity(a, b []float64) float64 {
	if len(a) == 0 || len(b) == 0 || len(a) != len(b) {
		return 0
	}
	var dot, normA, normB float64
	for i := 0; i < len(a); i++ {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

func (s *DeduplicationService) calculateSimilarity(p1, p2 *domain.Profile) (float64, []string) {
	var score float64
	var reasons []string

	// 1. Biometric Facial Recognition Check
	emb1 := extractFaceEmbedding(p1.Metadata)
	emb2 := extractFaceEmbedding(p2.Metadata)
	if len(emb1) > 0 && len(emb2) > 0 {
		sim := cosineSimilarity(emb1, emb2)
		if sim >= 0.80 {
			// Cosine similarity >= 0.80 indicates the exact same person's face
			bioScore := 80.0 + (sim-0.80)*95.0
			if bioScore > 99.0 {
				bioScore = 99.0
			}
			if bioScore > score {
				score = bioScore
			}
			reasons = append(reasons, fmt.Sprintf("🧬 Biometric Face Match: %.1f%% facial similarity with %s", sim*100, p2.LegalName))
		}
	} else if p1.ProfilePhotoURL != "" && p2.ProfilePhotoURL != "" && p1.ProfilePhotoURL == p2.ProfilePhotoURL {
		// Identical portrait image URL match
		if 95.0 > score {
			score = 95.0
		}
		reasons = append(reasons, fmt.Sprintf("🧬 Biometric Match: Identical facial portrait image match with %s", p2.LegalName))
	}

	// 2. Demographic & Contact Checks
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
	flags, err := s.dedupRepo.ListPending(ctx)
	if err != nil {
		return nil, err
	}

	// Enrich flags with details for Admin Console
	for _, f := range flags {
		if s.identRepo != nil {
			if ident, err := s.identRepo.GetByID(ctx, f.IdentityID); err == nil && ident != nil {
				f.PrimaryEkaID = ident.EkaID
			}
			if suspIdent, err := s.identRepo.GetByID(ctx, f.SuspectedDuplicateID); err == nil && suspIdent != nil {
				f.SuspectedEkaID = suspIdent.EkaID
			}
		}
		if s.profRepo != nil {
			if prof, err := s.profRepo.GetByIdentityID(ctx, f.IdentityID); err == nil && prof != nil {
				f.PrimaryName = prof.LegalName
				f.PrimaryPhoto = prof.ProfilePhotoURL
			}
			if suspProf, err := s.profRepo.GetByIdentityID(ctx, f.SuspectedDuplicateID); err == nil && suspProf != nil {
				f.SuspectedName = suspProf.LegalName
				f.SuspectedPhoto = suspProf.ProfilePhotoURL
			}
		}
	}

	return flags, nil
}

func (s *DeduplicationService) ResolveFlag(ctx context.Context, flagID uuid.UUID, status string, reviewerID uuid.UUID) error {
	return s.dedupRepo.ResolveFlag(ctx, flagID, status, reviewerID)
}