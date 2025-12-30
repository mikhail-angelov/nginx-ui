package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const cookieName = "jwt"

var secretKey = []byte("secret-key-from-env-file")

// SetAuthCookie sets an authentication cookie with a JWT token for the given email.
func SetAuthCookie(w http.ResponseWriter, email string) error {
	token, err := createToken(email)
	if err != nil {
		return err
	}

	expiration := time.Now().Add(365 * 24 * time.Hour)
	cookie := http.Cookie{Name: cookieName, Value: token, Expires: expiration, MaxAge: 86400, HttpOnly: true}
	http.SetCookie(w, &cookie)
	return nil
}

// CleanAuthCookie removes the authentication cookie.
func CleanAuthCookie(w http.ResponseWriter) {
	cookie := http.Cookie{Name: cookieName, Value: "", Expires: time.Now().Add(-1 * time.Second), MaxAge: -1, HttpOnly: true}
	http.SetCookie(w, &cookie)
}

// GetAuthCookie retrieves and verifies the authentication cookie from the request.
func GetAuthCookie(r *http.Request) (map[string]string, error) {
	cookie, err := r.Cookie(cookieName)
	if err != nil {
		return nil, fmt.Errorf("failed to get cookie: %w", err)
	}
	return verifyToken(cookie.Value)
}

func createToken(username string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256,
		jwt.MapClaims{
			"username": username,
			"exp":      time.Now().Add(time.Hour * 24).Unix(),
		})

	tokenString, err := token.SignedString(secretKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

func verifyToken(tokenString string) (map[string]string, error) {
	token, err := jwt.Parse(tokenString, func(_ *jwt.Token) (any, error) {
		return secretKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	claim := make(map[string]string)
	claim["username"] = token.Claims.(jwt.MapClaims)["username"].(string)

	return claim, nil
}
