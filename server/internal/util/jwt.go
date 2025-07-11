package util

import (
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var jwtSecret = []byte(os.Getenv("JWT_SECRET"))
var expireTime time.Duration = 30

func GenerateJTWTokens(userID int32, email string) (accessToken string, refreshToken string, err error) {
	accessClaims := jwt.MapClaims{
		"user_id": userID,
		"email":   email,
		"exp":     time.Now().Add(15 * time.Minute).Unix(),
	}

	refreshClaims := jwt.MapClaims{
		"user_id": userID,
		"email":   email,
		"exp":     time.Now().Add(expireTime * 24 * time.Hour).Unix(),
	}

	accessJWT := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	refreshJWT := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)

	accessToken, err = accessJWT.SignedString(jwtSecret)
	if err != nil {
		return "", "", err
	}

	refreshToken, err = refreshJWT.SignedString(jwtSecret)
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

func VerifyAccessToken(tokenStr string) (claims jwt.MapClaims, err error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("Unexpected signing method")
		}
		return jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("Invalid access token")
}

func VerifyRefreshToken(tokenStr string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return jwtSecret, nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}

	if exp, ok := claims["exp"].(float64); ok {
		if time.Now().Unix() > int64(exp) {
			return nil, errors.New("token expired")
		}
	}

	return claims, nil
}
