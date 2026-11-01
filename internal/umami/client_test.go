package umami

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

const testAPIKey = "api-key"

func newServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return server
}

func TestNewClientRequiresCredentials(t *testing.T) {
	if _, err := NewClient("http://umami", Credentials{Username: "admin"}); err == nil {
		t.Fatal("expected an error without a password")
	}
	if _, err := NewClient("", Credentials{APIKey: testAPIKey}); err == nil {
		t.Fatal("expected an error without a url")
	}
}

func TestCreateWebsiteSendsTheAPIKey(t *testing.T) {
	server := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer "+testAPIKey {
			t.Errorf("authorization = %q", got)
		}
		var input CreateWebsite
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(Website{ID: "1", Name: input.Name, Domain: input.Domain})
	})
	client, err := NewClient(server.URL+"/", Credentials{APIKey: testAPIKey})
	if err != nil {
		t.Fatal(err)
	}
	website, err := client.CreateWebsite(context.Background(), CreateWebsite{Name: "app", Domain: "app.test"})
	if err != nil {
		t.Fatal(err)
	}
	if website.ID != "1" || website.Domain != "app.test" {
		t.Fatalf("website = %+v", website)
	}
}

func TestPasswordLoginIsCached(t *testing.T) {
	logins := 0
	server := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/auth/login" {
			logins++
			_ = json.NewEncoder(w).Encode(map[string]string{"token": "session"})
			return
		}
		if got := r.Header.Get("Authorization"); got != "Bearer session" {
			t.Errorf("authorization = %q", got)
		}
		_ = json.NewEncoder(w).Encode(Website{ID: "1"})
	})
	client, err := NewClient(server.URL, Credentials{Username: "admin", Password: "secret"})
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if _, err := client.GetWebsite(context.Background(), "1"); err != nil {
			t.Fatal(err)
		}
	}
	if logins != 1 {
		t.Fatalf("logins = %d", logins)
	}
}

func TestTwoFactorAccountsAreRefused(t *testing.T) {
	server := newServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"requiresTwoFactor": true, "partialToken": "x"})
	})
	client, err := NewClient(server.URL, Credentials{Username: "admin", Password: "secret"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.GetWebsite(context.Background(), "1"); err == nil {
		t.Fatal("expected an error")
	}
}

func TestMissingWebsites(t *testing.T) {
	for name, handler := range map[string]http.HandlerFunc{
		"http 404":  func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNotFound) },
		"null body": func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("null")) },
		"no body":   func(http.ResponseWriter, *http.Request) {},
	} {
		t.Run(name, func(t *testing.T) {
			client, err := NewClient(newServer(t, handler).URL, Credentials{APIKey: testAPIKey})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := client.GetWebsite(context.Background(), "1"); !errors.Is(err, ErrNotFound) {
				t.Fatalf("err = %v", err)
			}
		})
	}
}

func TestDeleteIsIdempotent(t *testing.T) {
	server := newServer(t, func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNotFound) })
	client, err := NewClient(server.URL, Credentials{APIKey: testAPIKey})
	if err != nil {
		t.Fatal(err)
	}
	if err := client.DeleteWebsite(context.Background(), "1"); err != nil {
		t.Fatal(err)
	}
}

func TestServerErrorsDoNotLeakTheBody(t *testing.T) {
	server := newServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("internal detail"))
	})
	client, err := NewClient(server.URL, Credentials{APIKey: testAPIKey})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.GetWebsite(context.Background(), "1")
	if err == nil || err.Error() != "umami: GET /api/websites/1: HTTP 500" {
		t.Fatalf("err = %v", err)
	}
}

func TestVerifyReportsRejectedCredentials(t *testing.T) {
	server := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/me" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.WriteHeader(http.StatusUnauthorized)
	})
	client, err := NewClient(server.URL, Credentials{APIKey: testAPIKey})
	if err != nil {
		t.Fatal(err)
	}
	if err := client.Verify(context.Background()); err == nil {
		t.Fatal("expected an error for rejected credentials")
	}
}
