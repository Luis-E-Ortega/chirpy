package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestBasicTokenCreationAndValidation(t *testing.T) {
	t.Parallel()

	id := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	secret := "pass"

	token, err := MakeJWT(id, secret, 1*time.Hour)
	if err != nil {
		t.Fatalf("MakeJWT returned an error: %v", err)
	}
	if token == "" {
		t.Fatalf("expected non-empty token")
	}

	gotID, err := ValidateJWT(token, secret)
	if err != nil {
		t.Fatalf("ValidateJWT return error :%v", err)
	}
	if gotID != id {
		t.Errorf("got ID %v, want %v", gotID, id)
	}
}

func TestExpiredToken(t *testing.T) {
	t.Parallel()
	id := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	secret := "pass"
	tok, err := MakeJWT(id, secret, 1*time.Nanosecond)
	if err != nil {
		t.Fatalf("MakeJWT err : %v", err)
	}
	if _, err := ValidateJWT(tok, secret); err == nil {
		t.Fatalf("expected error for expried token")
	}
}

func TestWrongSecret(t *testing.T) {
	t.Parallel()
	id := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	tok, err := MakeJWT(id, "secretA", 1*time.Hour)
	if err != nil {
		t.Fatalf("MakeJWT err : %v", err)
	}
	if _, err := ValidateJWT(tok, "secretB"); err == nil {
		t.Fatalf("expected error with wrong secret")
	}
}
