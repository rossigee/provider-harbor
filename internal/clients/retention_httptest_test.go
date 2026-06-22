/*
Proof test for the real Retention client implementation.

Runs the actual HarborClient.{Create,List,Get,Update,Delete}RetentionPolicy methods
against a stateful in-memory fake of Harbor's /api/v2.0 retention + project APIs
(httptest). The fake simulates Harbor's real behaviour: creating a retention policy
automatically links it to the project via project metadata retention_id; listing
reads that metadata to find the policy.

Does NOT hit a live Harbor; uses no real credentials.
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

func fakeRetentionServer(t *testing.T) *httptest.Server {
	t.Helper()
	var mu sync.Mutex

	// A single fake project with ID 5.
	const projectID = int64(5)
	projectRetentionID := int64(0) // 0 means no retention bound yet

	type retentionPolicy struct {
		ID        int64
		ProjectID int64
		Algorithm string
	}
	retentions := map[int64]*retentionPolicy{}
	nextRetentionID := int64(100)

	mux := http.NewServeMux()

	// GET /api/v2.0/projects/{nameOrID} – used by ListRetentionPolicies to read retention_id.
	// Harbor accepts both numeric IDs and project names here.
	mux.HandleFunc("/api/v2.0/projects/", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		// Accept /api/v2.0/projects/5 or /api/v2.0/projects/testproject.
		retIDStr := ""
		if projectRetentionID != 0 {
			retIDStr = strconv.FormatInt(projectRetentionID, 10)
		}
		meta := map[string]interface{}{}
		if retIDStr != "" {
			meta["retention_id"] = retIDStr
		}
		resp := map[string]interface{}{
			"project_id": projectID,
			"name":       "testproject",
			"metadata":   meta,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	// POST /api/v2.0/retentions – create policy, return Location.
	// GET-by-list is not a real Harbor endpoint; see ListRetentionPolicies (reads project metadata).
	mux.HandleFunc("/api/v2.0/retentions", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var raw map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&raw)

		// Extract project ID from scope.ref to simulate Harbor linking the policy.
		scopeRef := int64(0)
		if scope, ok := raw["scope"].(map[string]interface{}); ok {
			if ref, ok := scope["ref"].(float64); ok {
				scopeRef = int64(ref)
			}
		}

		nextRetentionID++
		id := nextRetentionID
		retentions[id] = &retentionPolicy{
			ID:        id,
			ProjectID: scopeRef,
			Algorithm: "or",
		}
		// Simulate Harbor linking this retention to the project.
		if scopeRef == projectID {
			projectRetentionID = id
		}
		w.Header().Set("Location", "/api/v2.0/retentions/"+strconv.FormatInt(id, 10))
		w.WriteHeader(http.StatusCreated)
	})

	// GET/PUT/DELETE /api/v2.0/retentions/{id}
	mux.HandleFunc("/api/v2.0/retentions/", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		idStr := strings.TrimPrefix(r.URL.Path, "/api/v2.0/retentions/")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		p, ok := retentions[id]
		switch r.Method {
		case http.MethodGet:
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"errors":[{"code":"NOT_FOUND","message":"retention not found"}]}`))
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id":        p.ID,
				"algorithm": p.Algorithm,
				"scope": map[string]interface{}{
					"level": "project",
					"ref":   p.ProjectID,
				},
				"rules":   []interface{}{},
				"trigger": map[string]interface{}{"kind": "Schedule"},
			})
		case http.MethodPut:
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			// Accept update; nothing to change in this minimal fake.
			w.WriteHeader(http.StatusOK)
		case http.MethodDelete:
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			// Unlink from project.
			if projectRetentionID == id {
				projectRetentionID = 0
			}
			delete(retentions, id)
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	return httptest.NewServer(mux)
}

func TestRetentionPolicyClient_RealCRUD(t *testing.T) {
	srv := fakeRetentionServer(t)
	defer srv.Close()
	c := newTestClient(t, srv.URL)
	ctx := context.Background()

	const projectIDStr = "5" // matches fakeRetentionServer's projectID constant

	// ListRetentionPolicies before create -> empty.
	list, err := c.ListRetentionPolicies(ctx, projectIDStr)
	if err != nil {
		t.Fatalf("ListRetentionPolicies before create: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("expected empty list before create, got %d", len(list))
	}

	spec := &RetentionPolicySpec{
		ProjectID: projectIDStr,
		Trigger:   "manual",
		Rules: []RetentionPolicyRule{
			{
				RuleType:     "latestPushedK",
				TagSelectors: []string{"**"},
				Parameters:   map[string]interface{}{"latestPushedK": "10"},
			},
		},
	}

	// Create -> returns real ID, not "1".
	st, err := c.CreateRetentionPolicy(ctx, spec)
	if err != nil {
		t.Fatalf("CreateRetentionPolicy: %v", err)
	}
	if st.ID == "1" {
		t.Error("expected real policy ID from API, got stub value '1'")
	}
	if st.ID == "" {
		t.Error("expected non-empty policy ID")
	}
	if st.ProjectID != projectIDStr {
		t.Errorf("expected ProjectID %q, got %q", projectIDStr, st.ProjectID)
	}
	savedID := st.ID
	t.Logf("created retention ID: %s", savedID)

	// ListRetentionPolicies after create -> policy appears (reads project metadata).
	list2, err := c.ListRetentionPolicies(ctx, projectIDStr)
	if err != nil {
		t.Fatalf("ListRetentionPolicies after create: %v", err)
	}
	if len(list2) != 1 {
		t.Errorf("expected 1 retention policy after create, got %d", len(list2))
	}
	if len(list2) > 0 && list2[0].ID != savedID {
		t.Errorf("expected policy ID %q in list, got %q", savedID, list2[0].ID)
	}

	// GetRetentionPolicy by ID.
	got, err := c.GetRetentionPolicy(ctx, projectIDStr, savedID)
	if err != nil {
		t.Fatalf("GetRetentionPolicy: %v", err)
	}
	if got == nil {
		t.Fatal("expected policy to exist")
	}

	// UpdateRetentionPolicy -> no error, policy still accessible.
	if _, err := c.UpdateRetentionPolicy(ctx, projectIDStr, savedID, spec); err != nil {
		t.Fatalf("UpdateRetentionPolicy: %v", err)
	}
	got2, err := c.GetRetentionPolicy(ctx, projectIDStr, savedID)
	if err != nil || got2 == nil {
		t.Fatalf("GetRetentionPolicy after update: got=%v err=%v", got2, err)
	}

	// DeleteRetentionPolicy -> gone; list is empty again.
	if err := c.DeleteRetentionPolicy(ctx, projectIDStr, savedID); err != nil {
		t.Fatalf("DeleteRetentionPolicy: %v", err)
	}
	gone, err := c.GetRetentionPolicy(ctx, projectIDStr, savedID)
	if err != nil || gone != nil {
		t.Fatalf("expected (nil,nil) after delete, got gone=%v err=%v", gone, err)
	}

	list3, err := c.ListRetentionPolicies(ctx, projectIDStr)
	if err != nil {
		t.Fatalf("ListRetentionPolicies after delete: %v", err)
	}
	if len(list3) != 0 {
		t.Errorf("expected empty list after delete, got %d", len(list3))
	}

	// Idempotent delete -> no error.
	if err := c.DeleteRetentionPolicy(ctx, projectIDStr, savedID); err != nil {
		t.Errorf("second delete should be idempotent, got %v", err)
	}
}
