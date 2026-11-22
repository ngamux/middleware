package auth

import (
	"errors"
	"testing"

	"github.com/golang-jwt/jwt/v4"
)

func TestAuthorizerStatic(t *testing.T) {
	authorizer := AuthorizerStatic("alpha", "beta", "")

	cases := []struct {
		token     string
		wantValue any
		wantError error
	}{
		{"alpha", "alpha", nil},
		{"beta", "beta", nil},
		{"gamma", nil, ErrorForbidden},
		{"", nil, ErrorForbidden},
	}

	for _, c := range cases {
		value, err := authorizer(c.token)
		if !errors.Is(err, c.wantError) {
			t.Errorf("AuthorizerStatic(%q) error = %v, want %v", c.token, err, c.wantError)
		}
		if value != c.wantValue {
			t.Errorf("AuthorizerStatic(%q) value = %v, want %v", c.token, value, c.wantValue)
		}
	}
}

func TestAuthorizerTokens(t *testing.T) {
	authorizer := AuthorizerTokens(map[string]any{
		"alpha": "user-a",
		"beta":  "user-b",
		"":      "ignored",
	})

	cases := []struct {
		token     string
		wantValue any
		wantError error
	}{
		{"alpha", "user-a", nil},
		{"beta", "user-b", nil},
		{"gamma", nil, ErrorForbidden},
		{"", nil, ErrorForbidden},
	}

	for _, c := range cases {
		value, err := authorizer(c.token)
		if !errors.Is(err, c.wantError) {
			t.Errorf("AuthorizerTokens(%q) error = %v, want %v", c.token, err, c.wantError)
		}
		if value != c.wantValue {
			t.Errorf("AuthorizerTokens(%q) value = %v, want %v", c.token, value, c.wantValue)
		}
	}
}

func TestAuthorizerJWT(t *testing.T) {
	signingKey := []byte("secret-key")
	tokenString, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "user-1",
	}).SignedString(signingKey)
	if err != nil {
		t.Fatalf("signing token: %v", err)
	}

	value, err := AuthorizerJWT(signingKey)(tokenString)
	if err != nil {
		t.Fatalf("authorize valid token: %v", err)
	}
	if _, ok := value.(*jwt.Token); !ok {
		t.Errorf("value type = %T, want *jwt.Token", value)
	}

	if _, err := AuthorizerJWT([]byte("other-key"))(tokenString); err == nil {
		t.Error("authorize token with wrong key: expected error")
	}

	if _, err := AuthorizerJWT(nil)(tokenString); !errors.Is(err, ErrorForbidden) {
		t.Errorf("authorize with empty key error = %v, want %v", err, ErrorForbidden)
	}
}

func TestAuthorizerJWTRejectsNonHMAC(t *testing.T) {
	tokenString, err := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{
		"sub": "user-1",
	}).SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("signing token: %v", err)
	}

	if _, err := AuthorizerJWT([]byte("secret-key"))(tokenString); err == nil {
		t.Error("authorize unsigned token: expected error")
	}
}

func TestAuthorizerJWTWithKeyFunc(t *testing.T) {
	signingKey := []byte("secret-key")
	tokenString, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "user-1",
	}).SignedString(signingKey)
	if err != nil {
		t.Fatalf("signing token: %v", err)
	}

	called := false
	authorizer := AuthorizerJWTWithKeyFunc(func(*jwt.Token) (any, error) {
		called = true
		return signingKey, nil
	})

	if _, err := authorizer(tokenString); err != nil {
		t.Fatalf("authorize valid token: %v", err)
	}
	if !called {
		t.Error("key function was not called")
	}

	if _, err := AuthorizerJWTWithKeyFunc(nil)(tokenString); !errors.Is(err, ErrorForbidden) {
		t.Errorf("nil key function error = %v, want %v", err, ErrorForbidden)
	}
}
