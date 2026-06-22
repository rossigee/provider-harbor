/*
Proof test for the real Registry client implementation.

Runs the actual HarborClient.{Create,Get,Update,Delete}Registry methods against a
stateful in-memory fake of Harbor's /api/v2.0 registry API (httptest). This
exercises the real goharbor request/response path — it does NOT hit a live
Harbor and uses no real credentials. It demonstrates the methods are genuinely
implemented (issue the right HTTP verbs/paths, parse the real registry ID and
URL/type from Harbor responses, and map absent -> (nil, nil)) rather than
returning hardcoded stubs.

Key API shape:
  - POST /api/v2.0/registries         → 201 Created, Location header (no body)
  - GET  /api/v2.0/registries?name=N  → 200 OK, []Registry (list+match by name)
  - PUT  /api/v2.0/registries/{id}    → 200 OK
  - DELETE /api/v2.0/registries/{id}  → 200 OK
  - GET /api/v2.0/registries/{id}     → not used by our client (we use list+match)
*/
package clients

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func fakeHarborRegistry(t *testing.T) *httptest.Server {
	t.Helper()
	var mu sync.Mutex
	type regEntry struct {
		ID          int64
		Name        string
		Type        string
		URL         string
		Description string
	}
	registries := map[string]*regEntry{}
	nextID := 10

	mux := http.NewServeMux()

	// POST /api/v2.0/registries — create
	mux.HandleFunc("/api/v2.0/registries", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		switch r.Method {
		case http.MethodGet:
			// GET /api/v2.0/registries?name=... — list with optional name filter
			filterName := r.URL.Query().Get("name")
			var list []map[string]interface{}
			for _, e := range registries {
				if filterName == "" || e.Name == filterName {
					list = append(list, map[string]interface{}{
						"id":          e.ID,
						"name":        e.Name,
						"type":        e.Type,
						"url":         e.URL,
						"description": e.Description,
					})
				}
			}
			if list == nil {
				list = []map[string]interface{}{}
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(list)

		case http.MethodPost:
			var body struct {
				Name        string `json:"name"`
				Type        string `json:"type"`
				URL         string `json:"url"`
				Description string `json:"description"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			nextID++
			registries[body.Name] = &regEntry{
				ID:          int64(nextID),
				Name:        body.Name,
				Type:        body.Type,
				URL:         body.URL,
				Description: body.Description,
			}
			w.Header().Set("Location", "/api/v2.0/registries/"+itoa(nextID))
			w.WriteHeader(http.StatusCreated)

		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	// /api/v2.0/registries/{id} — get, update, delete by numeric ID
	mux.HandleFunc("/api/v2.0/registries/", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		idStr := strings.TrimPrefix(r.URL.Path, "/api/v2.0/registries/")
		// Find entry by ID
		var entry *regEntry
		for _, e := range registries {
			if itoa(int(e.ID)) == idStr {
				entry = e
				break
			}
		}

		switch r.Method {
		case http.MethodGet:
			if entry == nil {
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"errors":[{"code":"NOT_FOUND","message":"registry not found"}]}`))
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id":   entry.ID,
				"name": entry.Name,
				"type": entry.Type,
				"url":  entry.URL,
			})

		case http.MethodPut:
			if entry == nil {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			var body struct {
				URL         *string `json:"url"`
				Description *string `json:"description"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body.URL != nil {
				entry.URL = *body.URL
			}
			if body.Description != nil {
				entry.Description = *body.Description
			}
			w.WriteHeader(http.StatusOK)

		case http.MethodDelete:
			if entry == nil {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			delete(registries, entry.Name)
			w.WriteHeader(http.StatusOK)

		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	return httptest.NewServer(mux)
}

func TestRegistryClient_RealCRUD(t *testing.T) {
	srv := fakeHarborRegistry(t)
	defer srv.Close()
	c := newTestClient(t, srv.URL)
	ctx := context.Background()

	// Not found before creation -> (nil, nil).
	if st, err := c.GetRegistry(ctx, "proof-reg"); err != nil || st != nil {
		t.Fatalf("expected (nil,nil) before create, got st=%v err=%v", st, err)
	}

	// Create -> returns the authoritative ID from Harbor.
	st, err := c.CreateRegistry(ctx, &RegistrySpec{
		Name: "proof-reg",
		Type: "docker-hub",
		URL:  "https://docker.io",
	})
	if err != nil {
		t.Fatalf("CreateRegistry: %v", err)
	}
	if st.ID == 0 {
		t.Errorf("expected real registry ID from API, got 0 (stub would be 0 or hardcoded)")
	}
	if st.Name != "proof-reg" {
		t.Errorf("expected name proof-reg, got %q", st.Name)
	}
	if st.URL != "https://docker.io" {
		t.Errorf("expected URL https://docker.io, got %q", st.URL)
	}

	savedID := st.ID

	// Get -> exists now.
	st2, err := c.GetRegistry(ctx, "proof-reg")
	if err != nil {
		t.Fatalf("GetRegistry after create: %v", err)
	}
	if st2 == nil {
		t.Fatal("expected non-nil after create")
	}
	if st2.ID != savedID {
		t.Errorf("ID mismatch: got %d want %d", st2.ID, savedID)
	}

	// Update URL, then verify round-trip.
	_, err = c.UpdateRegistry(ctx, "proof-reg", &RegistrySpec{
		Name: "proof-reg",
		Type: "docker-hub",
		URL:  "https://updated.docker.io",
	})
	if err != nil {
		t.Fatalf("UpdateRegistry: %v", err)
	}
	st3, err := c.GetRegistry(ctx, "proof-reg")
	if err != nil {
		t.Fatalf("GetRegistry after update: %v", err)
	}
	if st3 == nil {
		t.Fatal("expected non-nil after update")
	}
	if st3.URL != "https://updated.docker.io" {
		t.Errorf("expected updated URL, got %q", st3.URL)
	}

	// Delete -> gone.
	if err := c.DeleteRegistry(ctx, "proof-reg"); err != nil {
		t.Fatalf("DeleteRegistry: %v", err)
	}
	if st, err := c.GetRegistry(ctx, "proof-reg"); err != nil || st != nil {
		t.Fatalf("expected (nil,nil) after delete, got st=%v err=%v", st, err)
	}

	// Idempotent delete: deleting absent registry must be a no-op.
	if err := c.DeleteRegistry(ctx, "proof-reg"); err != nil {
		t.Fatalf("DeleteRegistry (idempotent): %v", err)
	}
}
