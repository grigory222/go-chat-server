package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/grigory222/go-chat-server/internal/domain/models"
)

func TestNewTokensAndGetUserID(t *testing.T) {
	user := &models.User{ID: 42, Name: "Alice"}

	access, refresh, err := NewTokens(user, time.Minute, time.Minute*2, []byte("secret"))
	if err != nil {
		t.Fatalf("NewTokens returned error: %v", err)
	}
	if access == "" || refresh == "" {
		t.Fatalf("expected non-empty tokens")
	}

	uid, err := GetUserID(refresh, []byte("secret"))
	if err != nil {
		t.Fatalf("GetUserID failed: %v", err)
	}
	if uid != user.ID {
		t.Fatalf("expected user id %d, got %d", user.ID, uid)
	}

	// Access token should also decode
	uid2, err := GetUserID(access, []byte("secret"))
	if err != nil || uid2 != user.ID {
		t.Fatalf("access token user id mismatch: %v %d", err, uid2)
	}

	// Decode access token to ensure Name claim present, and refresh token missing Name.
	tokAccess, _ := jwt.ParseWithClaims(access, &Claims{}, func(token *jwt.Token) (interface{}, error) { return []byte("secret"), nil })
	aClaims := tokAccess.Claims.(*Claims)
	if aClaims.Name != user.Name {
		t.Fatalf("expected name claim %s, got %s", user.Name, aClaims.Name)
	}
	tokRefresh, _ := jwt.ParseWithClaims(refresh, &Claims{}, func(token *jwt.Token) (interface{}, error) { return []byte("secret"), nil })
	rClaims := tokRefresh.Claims.(*Claims)
	if rClaims.Name != "" { // refresh shouldn't set name in current implementation
		t.Fatalf("expected empty name in refresh token, got %s", rClaims.Name)
	}
}

func TestGetUserID_InvalidToken(t *testing.T) {
	_, err := GetUserID("not-a-token", []byte("secret"))
	if err == nil {
		t.Fatalf("expected error for invalid token string")
	}
}

func TestGetUserID_ExpiredToken(t *testing.T) {
	claims := Claims{RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Minute))}, UserID: 100}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := token.SignedString([]byte("secret"))
	if err != nil {
		t.Fatalf("sign error: %v", err)
	}
	_, err = GetUserID(s, []byte("secret"))
	if err == nil {
		t.Fatalf("expected error for expired token")
	}
}

func TestGetUserID_WrongAlg(t *testing.T) {
	// Use HS512 to trigger signing method mismatch
	claims := Claims{RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute))}, UserID: 7}
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
	s, err := token.SignedString([]byte("secret"))
	if err != nil {
		t.Fatalf("sign error: %v", err)
	}
	_, err = GetUserID(s, []byte("secret"))
	if err == nil {
		t.Fatalf("expected error for wrong signing method")
	}
}
