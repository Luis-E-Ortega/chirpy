package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
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

func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error) {
	if expiresIn <= 0 {
		err := errors.New("token is already expired")
		return "", err
	}
	claims := jwt.RegisteredClaims{
		Issuer:    "chirpy",
		IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
		ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(expiresIn)),
		Subject:   userID.String(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(tokenSecret))
	if err != nil {
		fmt.Printf("Error while attempting to sign token")
		return "", err
	}
	return signedToken, nil

}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
	claims := &jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		// return signing key (making sure to check the signing method type for security!)
		return []byte(tokenSecret), nil
	})
	if err != nil {
		return uuid.UUID{}, err
	}

	// Check to confirm token is valid
	if !token.Valid {
		err = errors.New("invalid token")
		return uuid.UUID{}, err
	}

	uniqueID, err := uuid.Parse(claims.Subject)
	if err != nil {
		fmt.Printf("Error parsing unique ID from claims")
		return uuid.UUID{}, err
	}

	return uniqueID, nil
}

func GetBearerToken(headers http.Header) (string, error) {
	authHeader := headers.Get("Authorization")
	if authHeader == "" {
		err := errors.New("authorization header is empty")
		return "", err
	}
	correctPrefix := strings.HasPrefix(authHeader, "Bearer ")
	if !correctPrefix {
		err := errors.New("authorization header is missing 'Bearer' prefix")
		return "", err
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	token = strings.TrimSpace(token)
	return token, nil
}

func MakeRefreshToken() (string, error) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		return "", err
	}
	hexToken := hex.EncodeToString(key)
	return hexToken, nil
}
