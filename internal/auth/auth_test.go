package auth

import (
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