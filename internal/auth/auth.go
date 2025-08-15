package auth

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

func HashPassword(password string) (string, error) {
	hashedPwd, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		fmt.Printf("Error hashing password: %s", err)
		return "", err
	}
	return string(hashedPwd), nil
}

func CheckPasswordHash(password, hash string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err != nil {
		fmt.Printf("Error comparing hash and password: %s", err)
		return err
	}
	return nil
}
