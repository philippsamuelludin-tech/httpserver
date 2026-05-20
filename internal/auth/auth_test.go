package auth

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestCreateToken(t *testing.T) {
	originalID := uuid.New()
	tokenString, err := MakeJWT(originalID, "1234", time.Hour)
	if err != nil {
		t.Errorf("%s", err)
	}

	validatedID, err := ValidateJWT(tokenString, "1234")
	if err != nil {
		t.Errorf("%s", err)
	}

	if originalID.String() != validatedID.String() {
		t.Errorf("got %v\n, want %v\n", tokenString, validatedID)
	}
}

func TestGetBearerToken_ValidToken(t *testing.T) {
	headers := http.Header{}
	expectedToken := "test-token-123"
	headers.Set("Authorization", "Bearer "+expectedToken)

	token, err := GetBearerToken(headers)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if token != expectedToken {
		t.Errorf("got %q, want %q", token, expectedToken)
	}
}

func TestGetBearerToken_MissingHeader(t *testing.T) {
	headers := http.Header{}

	token, err := GetBearerToken(headers)
	if err == nil {
		t.Errorf("expected error for missing header, got nil")
	}

	if token != "" {
		t.Errorf("got %q, want empty string", token)
	}

	expectedError := "Header has no Authorization Header"
	if err.Error() != expectedError {
		t.Errorf("got error %q, want %q", err.Error(), expectedError)
	}
}

func TestGetBearerToken_EmptyHeader(t *testing.T) {
	headers := http.Header{}
	headers.Set("Authorization", "")

	token, err := GetBearerToken(headers)
	if err == nil {
		t.Errorf("expected error for empty header, got nil")
	}

	if token != "" {
		t.Errorf("got %q, want empty string", token)
	}
}

func TestGetBearerToken_OnlyBearerPrefix(t *testing.T) {
	headers := http.Header{}
	headers.Set("Authorization", "Bearer ")

	token, err := GetBearerToken(headers)
	if err == nil {
		t.Errorf("expected error for bearer-only header, got nil")
	}

	if token != "" {
		t.Errorf("got %q, want empty string", token)
	}
}

func TestGetBearerToken_NoBearerPrefix(t *testing.T) {
	headers := http.Header{}
	headers.Set("Authorization", "token-without-bearer")

	token, err := GetBearerToken(headers)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// When there's no "Bearer " prefix, strings.Trim doesn't remove anything
	if token != "token-without-bearer" {
		t.Errorf("got %q, want %q", token, "token-without-bearer")
	}
}
