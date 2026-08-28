package crypto

import (
	"crypto/rand"
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
