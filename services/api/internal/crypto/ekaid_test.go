package crypto

import (
	"testing"
)

func TestGenerateEkaID_FormatAndValidity(t *testing.T) {
	for i := 0; i < 100; i++ {
		id, err := GenerateEkaID()
		if err != nil {
			t.Fatalf("GenerateEkaID returned error: %v", err)
		}
		if !ValidateEkaID(id) {
			t.Fatalf("Generated ID %s failed validation", id)
		}
		if len(id) != 13 {
			t.Fatalf("Expected length 13, got %d for %s", len(id), id)
		}
	}
}

func TestGenerateEkaID_CollisionResistance(t *testing.T) {
	// Test generation of 10,000 IDs to guarantee 0 collisions
	generated := make(map[string]struct{}, 10000)
	for i := 0; i < 10000; i++ {
		id, err := GenerateEkaID()
		if err != nil {
			t.Fatalf("Error generating ID: %v", err)
		}
		if _, exists := generated[id]; exists {
			t.Fatalf("Collision detected for ID: %s at iteration %d", id, i)
		}
		generated[id] = struct{}{}
	}
}

func TestValidateEkaID(t *testing.T) {
	tests := []struct {
		id    string
		valid bool
	}{
		{"EKA-7K4M-92PX", true},
		{"EKA-0000-0000", true},
		{"EKA-ZZZZ-ZZZZ", true},
		{"EKA-7K4M-92PI", false}, // Contains 'I'
		{"EKA-7K4M-92PL", false}, // Contains 'L'
		{"EKA-7K4M-92PO", false}, // Contains 'O'
		{"EKA-7K4M-92PU", false}, // Contains 'U'
		{"EKA-7K4-92PX", false},  // Too short
		{"EKA-7K4MM-92PX", false}, // Too long
		{"7K4M-92PX", false},     // Missing prefix
		{"AADHAAR-1234", false},  // Wrong format
	}

	for _, tt := range tests {
		got := ValidateEkaID(tt.id)
		if got != tt.valid {
			t.Errorf("ValidateEkaID(%q) = %v; want %v", tt.id, got, tt.valid)
		}
	}
}
