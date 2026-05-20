package auth

import (
	"github.com/alexedwards/argon2id"
	"time"
	"github.com/google/uuid"
	"github.com/golang-jwt/jwt/v5"
	"strings"
	"net/http"
	"errors"
)

func HashPassword(password string) (string, error) {
	hashed_pass, err := argon2id.CreateHash(password, argon2id.DefaultParams)
	return hashed_pass, err
}

func CheckPasswordHash(password, hash string) (bool, error) {
	valid, err := argon2id.ComparePasswordAndHash(password, hash)
	return valid, err
}

func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error) {
	now := jwt.NewNumericDate(time.Now().UTC())
	expire := jwt.NewNumericDate(now.Time.Add(expiresIn))
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer: "chirpy-access",
		Subject: userID.String(),
		ExpiresAt: expire,
		IssuedAt: now,
	})

	tokenString, err := token.SignedString([]byte(tokenSecret))
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
    claims := &jwt.RegisteredClaims{}

    _, err := jwt.ParseWithClaims(
        tokenString,
        claims,
        func(token *jwt.Token) (any, error) {
            return []byte(tokenSecret), nil
        },
    )

    if err != nil {
		return uuid.Nil , err
	}
	
	subjectUUID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return uuid.Nil, err
	}
	return subjectUUID, nil
}

func GetBearerToken(headers http.Header) (string, error) {
	tokenString := headers.Get("Authorization")
	if tokenString == "" {
		return "", errors.New("Header has no Authorization Header")
	}
	tokenString = strings.TrimPrefix(tokenString, "Bearer ")
	return tokenString, nil
}