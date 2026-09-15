package repository_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sauryah/eka-id/services/api/internal/domain"
	"github.com/sauryah/eka-id/services/api/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

func TestMemoryStore_PersistencePreservesPasswordHash(t *testing.T) {
	tmpDir := t.TempDir()
	dbFile := filepath.Join(tmpDir, "test_database.json")

	// 1. Initialize store and create user with password
	store1 := repository.NewMemoryStore()
	store1.SetPersistenceFile(dbFile)

	rawPassword := "Password123!"
	hashed, err := bcrypt.GenerateFromPassword([]byte(rawPassword), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	userID := uuid.New()
	user := &domain.User{
		ID:           userID,
		Email:        "admin@eka.dev",
		PasswordHash: string(hashed),
		Role:         domain.RoleSystemAdmin,
		Status:       domain.IdentityStatusActive,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	ctx := context.Background()
	if err := store1.Users.Create(ctx, user); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	if err := store1.SaveToFile(); err != nil {
		t.Fatalf("failed to save to file: %v", err)
	}

	// 2. Initialize fresh store and load from file
	store2 := repository.NewMemoryStore()
	loaded, err := store2.LoadFromFile(dbFile)
	if err != nil {
		t.Fatalf("failed to load from file: %v", err)
	}
	if !loaded {
		t.Fatalf("expected file to be loaded")
	}

	loadedUser, err := store2.Users.GetByEmail(ctx, "admin@eka.dev")
	if err != nil {
		t.Fatalf("failed to get user by email: %v", err)
	}

	if loadedUser.PasswordHash == "" {
		t.Fatalf("expected password hash to be preserved, got empty string")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(loadedUser.PasswordHash), []byte(rawPassword)); err != nil {
		t.Fatalf("password hash mismatch: %v", err)
	}
}
