package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/grigory222/go-chat-server/internal/domain/models"
)

// Claims - структура для кастомных полей в JWT
type Claims struct {
	jwt.RegisteredClaims
	UserID int64  `json:"uid"`
	Name   string `json:"name"`
}

// NewTokens создает новую пару access и refresh токенов для пользователя.
func NewTokens(user *models.User, accessTokenTTL, refreshTokenTTL time.Duration, signingKey []byte) (string, string, error) {
	// Создание Access токена
	accessToken, err := newAccessToken(user, accessTokenTTL, signingKey)
	if err != nil {
		return "", "", err
	}

	// Создание Refresh токена
	refreshToken, err := newRefreshToken(user.ID, refreshTokenTTL, signingKey)
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

// newAccessToken создает только access токен.
func newAccessToken(user *models.User, ttl time.Duration, signingKey []byte) (string, error) {
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
		},
		UserID: user.ID,
		Name:   user.Name,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString(signingKey)
}

// newRefreshToken создает только refresh токен.
func newRefreshToken(userID int64, ttl time.Duration, signingKey []byte) (string, error) {
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
		},
		UserID: userID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString(signingKey)
}

// GetUserID извлекает ID пользователя из токена.
func GetUserID(tokenString string, signingKey []byte) (int64, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Строго проверяем, что используется именно HS256
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return signingKey, nil
	})

	if err != nil {
		return 0, fmt.Errorf("invalid token: %w", err)
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims.UserID, nil
	}

	return 0, fmt.Errorf("invalid token claims")
}
