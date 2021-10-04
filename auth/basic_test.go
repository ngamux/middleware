package auth

import (
	"github.com/ngamux/ngamux"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBasicAuthSuccess(t *testing.T) {
	mux := ngamux.NewNgamux()
	mux.Use(Basic(BasicConfig{
		Authorizer: func(username string, password string) bool {
			return username == "test-user" && password == "test-password"
		},
		ErrorHandler: func(rw http.ResponseWriter, err error) error {
			rw.WriteHeader(401)
			rw.Write([]byte("Unauthenticated"))
			return nil
		},
		Realm: "",
		Creds: nil,
	}))
	mux.Get("/greet", func(rw http.ResponseWriter, r *http.Request) error {
		rw.WriteHeader(200)
		rw.Write([]byte("hello"))
		return nil
	})

	request := httptest.NewRequest("GET", "/greet", nil)
	request.Header.Add("authorization", "Basic dGVzdC11c2VyOnRlc3QtcGFzc3dvcmQ=")

	recorder := httptest.NewRecorder()

	mux.ServeHTTP(recorder, request)

	statusCode := recorder.Result().StatusCode

	if statusCode != 200 {
		t.Errorf("Expected authentication to pass but authentication failed with statusCode: %d", statusCode)
	}
}

func TestBasicAuthFailure(t *testing.T) {
	mux := ngamux.NewNgamux()
	mux.Use(Basic(BasicConfig{
		Authorizer: func(username string, password string) bool {
			return username == "test-user" && password == "test-password"
		},
		ErrorHandler: func(rw http.ResponseWriter, err error) error {
			rw.WriteHeader(401)
			rw.Write([]byte("Unauthenticated"))
			return nil
		},
		Realm: "",
		Creds: nil,
	}))
	mux.Get("/greet", func(rw http.ResponseWriter, r *http.Request) error {
		rw.WriteHeader(200)
		rw.Write([]byte("hello"))
		return nil
	})

	request := httptest.NewRequest("GET", "/greet", nil)
	request.Header.Add("authorization", "Basic dGVzdDp0ZXN0LXBhc3N3b3Jk")

	recorder := httptest.NewRecorder()

	mux.ServeHTTP(recorder, request)

	statusCode := recorder.Result().StatusCode

	if statusCode != 401 {
		t.Errorf("Expected authentication to fail but authentication passes with statusCode: %d", statusCode)
	}

	body := recorder.Body.String()

	if body != "Unauthenticated" {
		t.Errorf("Expected authentication to fail with message \"Unauthenticated\", but it failed with message %q", body)
	}
}
