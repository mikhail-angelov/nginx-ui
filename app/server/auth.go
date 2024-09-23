package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const cookieName = "jwt"
var secretKey = []byte("secret-key-from-env-file")

func SetAuthCookie(w http.ResponseWriter, email string) {
	token, _ := createToken(email)
	// fmt.Println("token: ", token, "err: ", err)
	expiration := time.Now().Add(365 * 24 * time.Hour)
	cookie := http.Cookie{Name: cookieName, Value: token, Expires: expiration, MaxAge: 86400, HttpOnly: true}
	http.SetCookie(w, &cookie)
}

func CleanAuthCookie(w http.ResponseWriter) {
	cookie := http.Cookie{Name: cookieName, Value: "", Expires: time.Now().Add(-1 * time.Second), MaxAge: -1, HttpOnly: true}
	http.SetCookie(w, &cookie)
}

func GetAuthCookie(r *http.Request) (jwt.Claims, error) {
	cookie, err := r.Cookie(cookieName)
	if err != nil {
		return nil, err
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
		return "", err
	}

	return tokenString, nil
}

func verifyToken(tokenString string) (jwt.Claims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return secretKey, nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return token.Claims, nil
}
