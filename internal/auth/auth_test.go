package auth

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestHashPassword(t *testing.T) {
	password := "Password123@"
	hashedPassword, err := HashPassword(password)

	if err != nil {
		t.Errorf("HashPassword = %s, Got error: %v", hashedPassword, err)
	}
}

func TestCheckPasswordHash(t *testing.T) {
	password := "Password123@"
	hashedPassword, err := HashPassword(password)

	if err != nil {
		t.Errorf("HashPassword = %s, Got error hashing password: %v", hashedPassword, err)
	}

	passwordMatch, err := CheckPasswordHash(password, hashedPassword)

	if err != nil || !passwordMatch {
		t.Errorf("CheckHashedPassword = %v, Got error matching %v", passwordMatch, err)
	}
}

func TestCheckPasswordHashPasswordMismatch(t *testing.T) {
	password := "Password123@"
	mismatchedPassword := "password123@"
	hashedPassword, err := HashPassword(password)

	if err != nil {
		t.Errorf("HashPassword = %s, Got error hashing password: %v", hashedPassword, err)
	}

	passwordMatch, err := CheckPasswordHash(mismatchedPassword, hashedPassword)

	if err != nil || passwordMatch {
		t.Errorf("CheckHashedPassword = %v, Got error matching %v :::: Password Hashed: %s, Password Checked: %s", passwordMatch, err, password, mismatchedPassword)
	}
}

func TestMakeJWT(t *testing.T) {
	userID := uuid.New()
	tokenSecret := "shhhhh"
	expiresIN := time.Hour * 1

	_, err := MakeJWT(userID, tokenSecret, expiresIN)

	if err != nil {
		t.Errorf("Error making JWT: %v \n Using userID: %v, token secret: %v, expiration during: %v", err, userID, tokenSecret, expiresIN)
	}
}

func TestValidateJWT(t *testing.T) {
	userID := uuid.New()
	tokenSecret := "shhhhh"
	expiresIN := time.Hour * 1

	token, err := MakeJWT(userID, tokenSecret, expiresIN)

	if err != nil {
		t.Errorf("Error making JWT: %v \n Using userID: %v, token secret: %v, expiration during: %v",
			err, userID, tokenSecret, expiresIN)
		return
	}

	jwtUUID, err := ValidateJWT(token, tokenSecret)

	if err != nil {
		t.Errorf("Error validating JWT: %v \n Using userID: %v, token secret: %v, expiration during: %v",
			err, userID, tokenSecret, expiresIN)
		return
	}

	if jwtUUID.String() != userID.String() {
		t.Errorf("Error validating JWT: %v \n Using userID: %v, token secret: %v, expiration during: %v \n Expected: %v Actual: %v",
			err, userID, tokenSecret, expiresIN, userID.String(), jwtUUID.String())
	}
}

func TestValidateJWTExpired(t *testing.T) {
	userID := uuid.New()
	tokenSecret := "shhhhh"
	expiresIN := time.Second * 2

	token, err := MakeJWT(userID, tokenSecret, expiresIN)

	if err != nil {
		t.Errorf("Error making JWT: %v \n Using userID: %v, token secret: %v, expiration during: %v",
			err, userID, tokenSecret, expiresIN)
		return
	}

	time.Sleep(5 * time.Second)

	_, err = ValidateJWT(token, tokenSecret)

	errorMsg := "token has invalid claims: token is expired"

	if !strings.Contains(err.Error(), errorMsg) {
		t.Errorf("Expected error validating expired JWT: %v \n Using userID: %v, token secret: %v, expiration during: %v",
			err, userID, tokenSecret, expiresIN)
		return
	}
}

func TestValidateJWTInvalidTokenSecret(t *testing.T) {
	userID := uuid.New()
	tokenSecret := "shhhhh"
	expiresIN := time.Hour * 1

	token, err := MakeJWT(userID, tokenSecret, expiresIN)

	if err != nil {
		t.Errorf("Error making JWT: %v \n Using userID: %v, token secret: %v, expiration during: %v",
			err, userID, tokenSecret, expiresIN)
		return
	}

	_, err = ValidateJWT(token, tokenSecret+"zyx")

	errorMsg := "token signature is invalid: signature is invalid"

	if !strings.Contains(err.Error(), errorMsg) {
		t.Errorf("Expected error validating expired JWT: %v \n Using userID: %v, token secret: %v, expiration during: %v",
			err, userID, tokenSecret, expiresIN)
		return
	}
}

func TestValidateJWTInvalidTokenForADifferentUser(t *testing.T) {
	userID_1 := uuid.New()
	userID_2 := uuid.New()
	tokenSecret := "shhhhh"
	expiresIN := time.Hour * 1

	token_1, err := MakeJWT(userID_1, tokenSecret, expiresIN)

	if err != nil {
		t.Errorf("Error making JWT: %v \n Using userID: %v, token secret: %v, expiration during: %v",
			err, userID_1, tokenSecret, expiresIN)
		return
	}

	token_2, err := MakeJWT(userID_2, tokenSecret, expiresIN)

	if err != nil {
		t.Errorf("Error making JWT: %v \n Using userID: %v, token secret: %v, expiration during: %v",
			err, userID_1, tokenSecret, expiresIN)
		return
	}

	jwtUUID_1, err := ValidateJWT(token_1, tokenSecret)

	if err != nil {
		t.Errorf("Error validating JWT: %v \n Using userID: %v, token secret: %v, expiration during: %v",
			err, userID_1, tokenSecret, expiresIN)
		return
	}

	jwtUUID_2, err := ValidateJWT(token_2, tokenSecret)

	if err != nil {
		t.Errorf("Error validating JWT: %v \n Using userID: %v, token secret: %v, expiration during: %v",
			err, userID_2, tokenSecret, expiresIN)
		return
	}

	if jwtUUID_1.String() == jwtUUID_2.String() {
		t.Errorf("UUIDs returned by 2 different JWTs should not be the same")
	}
}
