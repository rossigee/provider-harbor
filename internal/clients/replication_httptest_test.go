/*
Proof test for the real Replication client implementation.

Runs the actual HarborClient.{Create,List,Get,Update,Delete}ReplicationPolicy methods
against a stateful in-memory fake of Harbor's /api/v2.0 replication policy API (httptest).
This exercises the real goharbor request/response path — it does NOT hit a live Harbor
and uses no real credentials. It demonstrates the methods are genuinely implemented
(issue the right HTTP verbs/paths, parse the real policy ID, and map 404 -> not-found)
rather than returning hardcoded stubs.
*/
package clients

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// fakeReplicationServer serves a stateful in-memory Harbor replication policy API.
func fakeReplicationServer(t *testing.T) *httptest.Server {
	t.Helper()
	var mu sync.Mutex
	type policy struct {
		ID      int64
		Name    string
		Enabled bool
		DestReg string
	}
	policies := map[int64]*policy{}
	nextID := int64(10)

	mux := http.NewServeMux()

	// GET /api/v2.0/registries?name=... -> registry lookup for dest-registry id
	// resolution (replication policies reference registries by numeric id).
	mux.HandleFunc("/api/v2.0/registries", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"id":7,"name":"my-dest-registry","url":"https://dst.example.com"}]`))
	})

	// POST /api/v2.0/replication/policies -> 201 with Location header
	// GET  /api/v2.0/replication/policies -> 200 list
	mux.HandleFunc("/api/v2.0/replication/policies", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		switch r.Method {
		case http.MethodPost:
			var body struct {
				Name     string `json:"name"`
				Enabled  bool   `json:"enabled"`
				DestName string `json:"-"`
			}
			// parse body to get name and dest_registry
			var raw map[string]interface{}
			_ = json.NewDecoder(r.Body).Decode(&raw)
			if v, ok := raw["name"].(string); ok {
				body.Name = v
			}
			if v, ok := raw["enabled"].(bool); ok {
				body.Enabled = v
			}
			nextID++
			id := nextID
			destReg := ""
			if dr, ok := raw["dest_registry"].(map[string]interface{}); ok {
				if n, ok := dr["name"].(string); ok {
					destReg = n
				}
			}
			policies[id] = &policy{ID: id, Name: body.Name, Enabled: body.Enabled, DestReg: destReg}
			w.Header().Set("Location", "/api/v2.0/replication/policies/"+strconv.FormatInt(id, 10))
			w.WriteHeader(http.StatusCreated)
		case http.MethodGet:
			list := make([]map[string]interface{}, 0, len(policies))
			for _, p := range policies {
				list = append(list, map[string]interface{}{
					"id":      p.ID,
					"name":    p.Name,
					"enabled": p.Enabled,
				})
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(list)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	// GET/PUT/DELETE /api/v2.0/replication/policies/{id}
	mux.HandleFunc("/api/v2.0/replication/policies/", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		idStr := strings.TrimPrefix(r.URL.Path, "/api/v2.0/replication/policies/")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		p, ok := policies[id]
		switch r.Method {
		case http.MethodGet:
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"errors":[{"code":"NOT_FOUND","message":"policy not found"}]}`))
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id":      p.ID,
				"name":    p.Name,
				"enabled": p.Enabled,
			})
		case http.MethodPut:
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			var raw map[string]interface{}
			_ = json.NewDecoder(r.Body).Decode(&raw)
			if v, ok := raw["name"].(string); ok {
				p.Name = v
			}
			if v, ok := raw["enabled"].(bool); ok {
				p.Enabled = v
			}
			w.WriteHeader(http.StatusOK)
		case http.MethodDelete:
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			delete(policies, id)
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	return httptest.NewServer(mux)
}

func TestReplicationPolicyClient_RealCRUD(t *testing.T) {
	srv := fakeReplicationServer(t)
	defer srv.Close()
	c := newTestClient(t, srv.URL)
	ctx := context.Background()

	spec := &ReplicationPolicySpec{
		Name:    "proof-replication",
		Enabled: boolPtr(true),
		Trigger: "manual",
		DestinationReg: &ReplicationPolicyDestination{
			Name: "my-dest-registry",
			URL:  "https://registry.example.com",
		},
	}

	// Create -> returns real ID, not "1".
	st, err := c.CreateReplicationPolicy(ctx, spec)
	if err != nil {
		t.Fatalf("CreateReplicationPolicy: %v", err)
	}
	if st.ID == "1" {
		t.Error("expected real policy ID from API, got stub value '1'")
	}
	if st.ID == "" {
		t.Error("expected non-empty policy ID")
	}
	if st.Name != "proof-replication" {
		t.Errorf("expected name proof-replication, got %q", st.Name)
	}
	savedID := st.ID
	t.Logf("created policy ID: %s", savedID)

	// List -> policy appears.
	list, err := c.ListReplicationPolicies(ctx)
	if err != nil {
		t.Fatalf("ListReplicationPolicies: %v", err)
	}
	found := false
	for _, p := range list {
		if p.Name == "proof-replication" {
			found = true
		}
	}
	if !found {
		t.Error("expected policy to appear in list")
	}

	// GetReplicationPolicy by ID.
	got, err := c.GetReplicationPolicy(ctx, savedID)
	if err != nil {
		t.Fatalf("GetReplicationPolicy: %v", err)
	}
	if got == nil {
		t.Fatal("expected policy to exist")
	}
	if got.Name != "proof-replication" {
		t.Errorf("expected name proof-replication, got %q", got.Name)
	}

	// Update -> change name (a non-omitempty field), verify round-trip.
	// Note: SDK model has enabled bool with omitempty so false is never sent;
	// we verify the update path succeeds (200 OK) and re-read returns the
	// policy (showing GET after PUT still works and ID is stable).
	updSpec := &ReplicationPolicySpec{
		Name:    "proof-replication-updated",
		Enabled: boolPtr(true),
		Trigger: "manual",
		DestinationReg: &ReplicationPolicyDestination{
			Name: "my-dest-registry",
			URL:  "https://registry.example.com",
		},
	}
	if _, err := c.UpdateReplicationPolicy(ctx, savedID, updSpec); err != nil {
		t.Fatalf("UpdateReplicationPolicy: %v", err)
	}
	got2, err := c.GetReplicationPolicy(ctx, savedID)
	if err != nil {
		t.Fatalf("GetReplicationPolicy after update: %v", err)
	}
	if got2.Name != "proof-replication-updated" {
		t.Errorf("expected name proof-replication-updated after update, got %q", got2.Name)
	}

	// Delete -> gone.
	if err := c.DeleteReplicationPolicy(ctx, savedID); err != nil {
		t.Fatalf("DeleteReplicationPolicy: %v", err)
	}
	gone, err := c.GetReplicationPolicy(ctx, savedID)
	if err != nil || gone != nil {
		t.Fatalf("expected (nil,nil) after delete, got gone=%v err=%v", gone, err)
	}

	// Idempotent delete -> no error.
	if err := c.DeleteReplicationPolicy(ctx, savedID); err != nil {
		t.Errorf("second delete should be idempotent, got %v", err)
	}
}

func boolPtr(b bool) *bool { return &b }
