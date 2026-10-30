package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBasicQuotesRealmInChallenge(t *testing.T) {
	tests := []struct {
		name  string
		realm string
		want  string
	}{
		{
			name:  "realm without spaces",
			realm: "Staging",
			want:  `Basic realm="Staging"`,
		},
		{
			name:  "realm with spaces",
			realm: "My App",
			want:  `Basic realm="My App"`,
		},
		{
			name:  "realm with quotes",
			realm: `He said "hi"`,
			want:  `Basic realm="He said \"hi\""`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			middleware := Basic(BasicConfig{Realm: tt.realm})
			handler := middleware(func(rw http.ResponseWriter, r *http.Request) {})

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/", nil)

			handler(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
			}

			if got := rec.Header().Get("WWW-Authenticate"); got != tt.want {
				t.Errorf("expected WWW-Authenticate %q, got %q", tt.want, got)
			}
		})
	}
}
