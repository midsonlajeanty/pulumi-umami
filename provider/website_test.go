package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/blang/semver"
	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/integration"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource/urn"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
)

const websiteToken = "umami:index:Website"

type fakeUmami struct {
	mu       sync.Mutex
	websites map[string]map[string]any
	next     int
}

func newFakeUmami(t *testing.T) (*fakeUmami, *httptest.Server) {
	t.Helper()
	fake := &fakeUmami{websites: map[string]map[string]any{}}
	server := httptest.NewServer(fake)
	t.Cleanup(server.Close)
	return fake, server
}

func (f *fakeUmami) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Authorization") != "Bearer test-key" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	switch {
	case r.URL.Path == "/api/me":
		writeJSON(w, map[string]any{"authType": "api-key"})
	case r.URL.Path == "/api/websites" && r.Method == http.MethodPost:
		f.next++
		website := decodeBody(r)
		website["id"] = fmt.Sprintf("site-%d", f.next)
		f.websites[website["id"].(string)] = website
		writeJSON(w, website)
	case strings.HasPrefix(r.URL.Path, "/api/websites/"):
		id := strings.TrimPrefix(r.URL.Path, "/api/websites/")
		website, ok := f.websites[id]
		switch {
		case r.Method == http.MethodGet && !ok:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte("null"))
		case r.Method == http.MethodGet:
			writeJSON(w, website)
		case r.Method == http.MethodPost && ok:
			for key, value := range decodeBody(r) {
				website[key] = value
			}
			writeJSON(w, website)
		case r.Method == http.MethodDelete:
			delete(f.websites, id)
			writeJSON(w, map[string]any{"ok": true})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func decodeBody(r *http.Request) map[string]any {
	body := map[string]any{}
	_ = json.NewDecoder(r.Body).Decode(&body)
	return body
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}

func newServer(t *testing.T, url string) integration.Server {
	t.Helper()
	built, err := New()
	if err != nil {
		t.Fatal(err)
	}
	server, err := integration.NewServer(context.Background(), Name, semver.MustParse("0.0.1"), integration.WithProvider(built))
	if err != nil {
		t.Fatal(err)
	}
	err = server.Configure(p.ConfigureRequest{Args: property.NewMap(map[string]property.Value{
		"url":    property.New(url),
		"apiKey": property.New("test-key"),
	})})
	if err != nil {
		t.Fatal(err)
	}
	return server
}

func websiteInputs(name, domain string) property.Map {
	return property.NewMap(map[string]property.Value{
		"name":   property.New(name),
		"domain": property.New(domain),
	})
}

func TestWebsiteLifecycle(t *testing.T) {
	fake, umamiServer := newFakeUmami(t)
	server := newServer(t, umamiServer.URL)

	integration.LifeCycleTest{
		Resource: websiteToken,
		Create: integration.Operation{
			Inputs: websiteInputs("Example", "example.com"),
			Hook: func(_, output property.Map) {
				if got := output.Get("name").AsString(); got != "Example" {
					t.Errorf("name = %q", got)
				}
				if len(fake.websites) != 1 {
					t.Errorf("websites = %d", len(fake.websites))
				}
			},
		},
		Updates: []integration.Operation{{
			Inputs: websiteInputs("Renamed", "renamed.example.com"),
			Hook: func(_, output property.Map) {
				if got := output.Get("domain").AsString(); got != "renamed.example.com" {
					t.Errorf("domain = %q", got)
				}
				for _, website := range fake.websites {
					if website["name"] != "Renamed" {
						t.Errorf("remote name = %v", website["name"])
					}
				}
			},
		}},
	}.Run(t, server)

	if len(fake.websites) != 0 {
		t.Errorf("websites left after delete = %d", len(fake.websites))
	}
}

func TestWebsiteReadReportsADeletedWebsite(t *testing.T) {
	_, umamiServer := newFakeUmami(t)
	server := newServer(t, umamiServer.URL)

	response, err := server.Read(p.ReadRequest{
		Urn: urn.New("test", "provider", "", websiteToken, "test"),
		ID:  "missing",
	})
	if err != nil {
		t.Fatal(err)
	}
	if response.ID != "" {
		t.Errorf("id = %q, expected empty for a deleted website", response.ID)
	}
}

func TestWebsiteReadImportsAnExistingWebsite(t *testing.T) {
	fake, umamiServer := newFakeUmami(t)
	fake.websites["site-7"] = map[string]any{"id": "site-7", "name": "Imported", "domain": "imported.example.com"}
	server := newServer(t, umamiServer.URL)

	response, err := server.Read(p.ReadRequest{
		Urn: urn.New("test", "provider", "", websiteToken, "test"),
		ID:  "site-7",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := response.Inputs.Get("domain").AsString(); got != "imported.example.com" {
		t.Errorf("domain = %q", got)
	}
}

func TestConfigureRejectsBadCredentials(t *testing.T) {
	_, umamiServer := newFakeUmami(t)
	built, err := New()
	if err != nil {
		t.Fatal(err)
	}
	server, err := integration.NewServer(context.Background(), Name, semver.MustParse("0.0.1"), integration.WithProvider(built))
	if err != nil {
		t.Fatal(err)
	}
	err = server.Configure(p.ConfigureRequest{Args: property.NewMap(map[string]property.Value{
		"url":    property.New(umamiServer.URL),
		"apiKey": property.New("wrong"),
	})})
	if err == nil {
		t.Fatal("expected configure to fail with rejected credentials")
	}
}
