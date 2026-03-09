package auth

import (
	"testing"
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
