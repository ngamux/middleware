package auth

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang-jwt/jwt/v4"
	"github.com/ngamux/ngamux"
)

func jwtAuthorizer(signingKey []byte) func(token string) (any, error) {
	return func(tokenString string) (any, error) {
		token, err := jwt.Parse(tokenString, func(*jwt.Token) (any, error) {
			return signingKey, nil
		})
		if err != nil {
			return nil, err
		}
		if !token.Valid {
			return nil, ErrorForbidden
		}

		return token, nil
	}
}

func TestBearerAcceptsAnyCaseForBearerScheme(t *testing.T) {
	middleware := Bearer(BearerConfig{
		Authorizer: func(token string) (any, error) {
			return token, nil
		},
	})

	for _, scheme := range []string{"Bearer", "bearer", "BEARER", "bEaReR"} {
		t.Run(scheme, func(t *testing.T) {
			nextCalled := false
			handler := middleware(func(rw http.ResponseWriter, r *http.Request) {
				nextCalled = true
				rw.WriteHeader(http.StatusNoContent)
			})

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Authorization", scheme+" random-key")
			rec := httptest.NewRecorder()

			handler(rec, req)

			if !nextCalled {
				t.Fatalf("next not called for %q scheme, got status %d, body %s", scheme, rec.Code, rec.Body.String())
			}
			if rec.Code != http.StatusNoContent {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
			}
		})
	}
}

func TestBearerRejectsMissingToken(t *testing.T) {
	nextCalled := false
	handler := Bearer(BearerConfig{
		Authorizer: func(token string) (any, error) {
			return token, nil
		},
	})(func(rw http.ResponseWriter, r *http.Request) {
		nextCalled = true
	})

	rec := httptest.NewRecorder()
	handler(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if nextCalled {
		t.Fatal("next was called without a token")
	}
	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestBearerRejectsWithoutAuthorizer(t *testing.T) {
	nextCalled := false
	handler := Bearer()(func(rw http.ResponseWriter, r *http.Request) {
		nextCalled = true
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer random-key")
	rec := httptest.NewRecorder()
	handler(rec, req)

	if nextCalled {
		t.Fatal("next was called without an authorizer")
	}
	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestBearerUsesCustomAuthorizer(t *testing.T) {
	var received string
	var stored any

	handler := Bearer(BearerConfig{
		Authorizer: func(token string) (any, error) {
			received = token
			if token != "random-key" {
				return nil, errors.New("invalid token")
			}
			return "principal", nil
		},
	})(func(rw http.ResponseWriter, r *http.Request) {
		stored = ngamux.Req(r).Locals("token")
		rw.WriteHeader(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer random-key")
	rec := httptest.NewRecorder()
	handler(rec, req)

	if received != "random-key" {
		t.Errorf("authorizer received %q, want %q", received, "random-key")
	}
	if stored != "principal" {
		t.Errorf("stored value = %v, want %q", stored, "principal")
	}
	if rec.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
}

func TestBearerRejectsWithCustomAuthorizer(t *testing.T) {
	nextCalled := false

	handler := Bearer(BearerConfig{
		Authorizer: func(string) (any, error) {
			return nil, errors.New("invalid token")
		},
	})(func(http.ResponseWriter, *http.Request) {
		nextCalled = true
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer random-key")
	rec := httptest.NewRecorder()
	handler(rec, req)

	if nextCalled {
		t.Fatal("next was called for an invalid token")
	}
	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestBearerSupportsJWTAuthorizer(t *testing.T) {
	signingKey := []byte("secret-key")
	tokenString, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "user-1",
	}).SignedString(signingKey)
	if err != nil {
		t.Fatalf("signing token: %v", err)
	}

	nextCalled := false
	handler := Bearer(BearerConfig{
		Authorizer: jwtAuthorizer(signingKey),
	})(func(rw http.ResponseWriter, r *http.Request) {
		nextCalled = true
		rw.WriteHeader(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if !nextCalled {
		t.Fatalf("next was not called for a valid JWT, got status %d, body %s", rec.Code, rec.Body.String())
	}
	if rec.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
}

func TestBearerRejectsInvalidJWTSignature(t *testing.T) {
	signingKey := []byte("secret-key")
	tokenString, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "user-1",
	}).SignedString(signingKey)
	if err != nil {
		t.Fatalf("signing token: %v", err)
	}

	nextCalled := false
	handler := Bearer(BearerConfig{
		Authorizer: jwtAuthorizer([]byte("other-key")),
	})(func(http.ResponseWriter, *http.Request) {
		nextCalled = true
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if nextCalled {
		t.Fatal("next was called for invalid signature")
	}
	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestBearerStoresTokenInLocals(t *testing.T) {
	var gotToken any
	handler := Bearer(BearerConfig{
		ContextKey: "bearer",
		Authorizer: func(token string) (any, error) {
			return "principal", nil
		},
	})(func(rw http.ResponseWriter, r *http.Request) {
		gotToken = ngamux.Req(r).Locals("bearer")
		rw.WriteHeader(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer random-key")
	handler(httptest.NewRecorder(), req)

	if gotToken != "principal" {
		t.Errorf("stored value = %v, want %q", gotToken, "principal")
	}
}
