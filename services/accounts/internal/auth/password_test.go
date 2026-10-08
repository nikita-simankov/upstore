package auth

import (
	"strings"
	"testing"
)

// TestHashAndVerifyPassword tests that a hash verifies its own password and no other.
func TestHashAndVerifyPassword(t *testing.T) {
	hash, err := HashPassword("correct horse")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if !strings.HasPrefix(hash, "$argon2id$") {
		t.Errorf("hash = %q, want argon2id PHC format", hash)
	}

	ok, err := VerifyPassword("correct horse", hash)
	if err != nil || !ok {
		t.Errorf("VerifyPassword(correct) = %v, %v; want true, nil", ok, err)
	}

	ok, err = VerifyPassword("wrong horse", hash)
	if err != nil || ok {
		t.Errorf("VerifyPassword(wrong) = %v, %v; want false, nil", ok, err)
	}
}

// TestHashUsesRandomSalt tests that hashing the same password twice gives different hashes.
func TestHashUsesRandomSalt(t *testing.T) {
	a, _ := HashPassword("same password")
	b, _ := HashPassword("same password")
	if a == b {
		t.Error("two hashes of the same password are identical, so the salt is not random")
	}
}

// TestVerifyPasswordRejectsMalformedHash tests that a corrupt stored hash is an error, not a match.
func TestVerifyPasswordRejectsMalformedHash(t *testing.T) {
	malformed := []string{
		"",
		"plaintext",
		"$bcrypt$v=1$m=1,t=1,p=1$c2FsdA$a2V5",
		"$argon2id$v=19$m=bad$c2FsdA$a2V5",
		"$argon2id$v=19$m=65536,t=3,p=4$!!!$a2V5",
		"$argon2id$v=19$m=65536,t=3,p=4$c2FsdA$",
	}
	for _, hash := range malformed {
		ok, err := VerifyPassword("anything", hash)
		if err == nil || ok {
			t.Errorf("VerifyPassword(%q) = %v, %v; want false with error", hash, ok, err)
		}
	}
}

// TestValidatePassword tests the length rules.
func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{"minimum length", "12345678", false},
		{"too short", "1234567", true},
		{"maximum length", strings.Repeat("a", 128), false},
		{"too long", strings.Repeat("a", 129), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidatePassword(tt.password); (err != nil) != tt.wantErr {
				t.Errorf("ValidatePassword() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
