package auth

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIN time.Duration) (string, error) {
	expirationTime := time.Now().Add(expiresIN)
	currentTimeUTC := time.Now().UTC()

	jwtClaims := jwt.RegisteredClaims{
		Issuer:    "chirpy-access",
		IssuedAt:  jwt.NewNumericDate(currentTimeUTC),
		ExpiresAt: jwt.NewNumericDate(expirationTime),
		Subject:   userID.String(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwtClaims)

	signedToken, err := token.SignedString([]byte(tokenSecret))

	if err != nil {
		return "", err
	}

	return signedToken, nil
}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
	regClaims := jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(tokenString, &regClaims, func(t *jwt.Token) (any, error) {
		return []byte(tokenSecret), nil
	})

	if err != nil {
		return uuid.UUID{}, err
	}

	if claims, ok := token.Claims.(*jwt.RegisteredClaims); ok {
		uuidValue, err := uuid.Parse(claims.Subject)
		if err != nil {
			return uuid.UUID{}, err
		}
		return uuidValue, nil
	} else {
		return uuid.UUID{}, fmt.Errorf("unknown claims type")
	}
}

func GetBearerToken(headers http.Header) (string, error) {
	bearerToken := headers.Get("Authorization")

	if bearerToken == "" {
		return "", fmt.Errorf("Bearer token not found")
	}

	if token, ok := strings.CutPrefix(bearerToken, "Bearer "); ok {
		return token, nil
	} else {
		return "", fmt.Errorf("Couldn't extract bearer token")
	}
}
