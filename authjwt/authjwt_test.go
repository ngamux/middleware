package authjwt

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang-jwt/jwt/v4"
)

func TestNewAcceptsAnyCaseForBearerScheme(t *testing.T) {
	signingKey := []byte("secret-key")
	tokenString, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "user-1",
	}).SignedString(signingKey)
	if err != nil {
		t.Fatalf("signing token: %v", err)
	}

	middleware := New(Config{SigningKey: signingKey})

	for _, scheme := range []string{"Bearer", "bearer", "BEARER", "bEaReR"} {
		t.Run(scheme, func(t *testing.T) {
			nextCalled := false
			handler := middleware(func(rw http.ResponseWriter, r *http.Request) {
				nextCalled = true
				rw.WriteHeader(http.StatusNoContent)
			})

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Authorization", scheme+" "+tokenString)
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
