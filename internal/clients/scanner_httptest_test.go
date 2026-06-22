/*
Proof test for the real Scanner registration client implementation.

Runs the actual HarborClient.{Create,Get,Update,Delete,List}ScannerRegistration
methods against a stateful in-memory fake of Harbor's /api/v2.0 scanner API
(httptest). This exercises the real goharbor request/response path — it does
NOT hit a live Harbor and uses no real credentials. It demonstrates the methods
are genuinely implemented (issue the right HTTP verbs/paths, extract the UUID
from the Location header, and return (nil, nil) for absent registrations)
rather than returning hardcoded stubs.
*/
package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func fakeHarborScanners(t *testing.T) *httptest.Server {
	t.Helper()
	var mu sync.Mutex
	type scanner struct {
		uuid        string
		name        string
		url         string
		description string
		auth        string
	}
	scanners := map[string]*scanner{}
	nextUUID := 0

	newUUID := func() string {
		nextUUID++
		return fmt.Sprintf("uuid-%04d", nextUUID)
	}

	scannerJSON := func(s *scanner) string {
		return fmt.Sprintf(`{"uuid":%q,"name":%q,"url":%q,"description":%q,"auth":%q,"create_time":"2026-01-01T00:00:00.000Z","update_time":"2026-01-01T00:00:00.000Z"}`,
			s.uuid, s.name, s.url, s.description, s.auth)
	}

	mux := http.NewServeMux()

	// POST/GET /api/v2.0/scanners — create / list
	mux.HandleFunc("/api/v2.0/scanners", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		switch r.Method {
		case http.MethodPost:
			var body struct {
				Name        string `json:"name"`
				URL         string `json:"url"`
				Description string `json:"description"`
				Auth        string `json:"auth"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			uuid := newUUID()
			scanners[uuid] = &scanner{
				uuid:        uuid,
				name:        body.Name,
				url:         body.URL,
				description: body.Description,
				auth:        body.Auth,
			}
			w.Header().Set("Location", "/api/v2.0/scanners/"+uuid)
			w.WriteHeader(http.StatusCreated)
		case http.MethodGet:
			var items []string
			for _, s := range scanners {
				items = append(items, scannerJSON(s))
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte("[" + strings.Join(items, ",") + "]"))
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	// /api/v2.0/scanners/<uuid> — get, update, delete
	mux.HandleFunc("/api/v2.0/scanners/", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		uuid := strings.TrimPrefix(r.URL.Path, "/api/v2.0/scanners/")
		s, ok := scanners[uuid]
		switch r.Method {
		case http.MethodGet:
			if !ok {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"errors":[{"code":"NOT_FOUND","message":"scanner not found"}]}`))
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(scannerJSON(s)))
		case http.MethodPut:
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			var body struct {
				Name        string `json:"name"`
				URL         string `json:"url"`
				Description string `json:"description"`
				Auth        string `json:"auth"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body.Name != "" {
				s.name = body.Name
			}
			if body.URL != "" {
				s.url = body.URL
			}
			s.description = body.Description
			s.auth = body.Auth
			w.WriteHeader(http.StatusOK)
		case http.MethodDelete:
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			delete(scanners, uuid)
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	return httptest.NewServer(mux)
}

func TestScannerClient_RealCRUD(t *testing.T) {
	srv := fakeHarborScanners(t)
	defer srv.Close()
	c := newTestClient(t, srv.URL)
	ctx := context.Background()

	// Not found before creation -> (nil, nil).
	if st, err := c.GetScannerRegistration(ctx, "nonexistent-uuid"); err != nil || st != nil {
		t.Fatalf("expected (nil,nil) before create, got st=%v err=%v", st, err)
	}

	// List before create -> empty.
	all, err := c.ListScannerRegistrations(ctx)
	if err != nil {
		t.Fatalf("ListScannerRegistrations before create: %v", err)
	}
	if len(all) != 0 {
		t.Errorf("expected empty list before create, got %d items", len(all))
	}

	// Create -> returns authoritative UUID and state from Harbor.
	desc := "Trivy scanner"
	st, err := c.CreateScannerRegistration(ctx, &ScannerSpec{
		Name:        "trivy",
		URL:         "https://trivy.example.com",
		Description: &desc,
	})
	if err != nil {
		t.Fatalf("CreateScannerRegistration: %v", err)
	}
	if st.UUID == "" {
		t.Fatal("expected non-empty UUID from Harbor Location header (stub would be empty)")
	}
	if !strings.HasPrefix(st.UUID, "uuid-") {
		t.Errorf("expected uuid- prefix from fake server, got %q (stub would return hardcoded)", st.UUID)
	}
	if st.Name != "trivy" {
		t.Errorf("expected name trivy, got %q", st.Name)
	}
	if st.URL != "https://trivy.example.com" {
		t.Errorf("expected URL https://trivy.example.com, got %q", st.URL)
	}

	uuid := st.UUID

	// Get by UUID -> exists now.
	got, err := c.GetScannerRegistration(ctx, uuid)
	if err != nil || got == nil {
		t.Fatalf("GetScannerRegistration after create: st=%v err=%v", got, err)
	}
	if got.UUID != uuid {
		t.Errorf("expected UUID %q, got %q", uuid, got.UUID)
	}

	// List -> contains the created scanner.
	all, err = c.ListScannerRegistrations(ctx)
	if err != nil {
		t.Fatalf("ListScannerRegistrations after create: %v", err)
	}
	if len(all) != 1 {
		t.Errorf("expected 1 scanner in list, got %d", len(all))
	}
	if all[0].UUID != uuid {
		t.Errorf("listed scanner UUID %q != expected %q", all[0].UUID, uuid)
	}

	// Update URL.
	if _, err := c.UpdateScannerRegistration(ctx, uuid, &ScannerSpec{
		Name: "trivy",
		URL:  "https://trivy-v2.example.com",
	}); err != nil {
		t.Fatalf("UpdateScannerRegistration: %v", err)
	}
	got2, err := c.GetScannerRegistration(ctx, uuid)
	if err != nil || got2 == nil {
		t.Fatalf("GetScannerRegistration after update: st=%v err=%v", got2, err)
	}
	if got2.URL != "https://trivy-v2.example.com" {
		t.Errorf("expected updated URL https://trivy-v2.example.com, got %q", got2.URL)
	}

	// Delete -> gone.
	if err := c.DeleteScannerRegistration(ctx, uuid); err != nil {
		t.Fatalf("DeleteScannerRegistration: %v", err)
	}
	if st, err := c.GetScannerRegistration(ctx, uuid); err != nil || st != nil {
		t.Fatalf("expected (nil,nil) after delete, got st=%v err=%v", st, err)
	}

	// Delete is idempotent.
	if err := c.DeleteScannerRegistration(ctx, uuid); err != nil {
		t.Fatalf("idempotent DeleteScannerRegistration: %v", err)
	}
}
