/*
Proof test for the real Webhook client implementation.

Runs the actual HarborClient.{Create,List,Get,Update,Delete}Webhook methods
against a stateful in-memory fake of Harbor's /api/v2.0 webhook policy API
(httptest). This exercises the real goharbor request/response path — it does
NOT hit a live Harbor and uses no real credentials. It demonstrates the methods
are genuinely implemented (issue the right HTTP verbs/paths, parse the real
policy ID from the list response, and map 404 -> not-found) rather than
returning hardcoded stubs.
*/
package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
)

func fakeHarborWebhooks(t *testing.T) *httptest.Server {
	t.Helper()
	var mu sync.Mutex

	type policy struct {
		id         int
		name       string
		address    string
		eventTypes []string
		enabled    bool
	}

	policies := map[int]*policy{}
	nextID := 9

	policyJSON := func(p *policy) string {
		evts := make([]string, 0, len(p.eventTypes))
		for _, e := range p.eventTypes {
			evts = append(evts, `"`+e+`"`)
		}
		return fmt.Sprintf(
			`{"id":%d,"name":%q,"enabled":%v,"event_types":[%s],"targets":[{"type":"http","address":%q}],"creation_time":"2026-01-01T00:00:00Z","update_time":"2026-01-01T00:00:00Z"}`,
			p.id, p.name, p.enabled, strings.Join(evts, ","), p.address,
		)
	}

	mux := http.NewServeMux()

	// /api/v2.0/projects/myproject/webhook/policies — list + create
	mux.HandleFunc("/api/v2.0/projects/myproject/webhook/policies", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		switch r.Method {
		case http.MethodPost:
			var body struct {
				Name       string   `json:"name"`
				Enabled    bool     `json:"enabled"`
				EventTypes []string `json:"event_types"`
				Targets    []struct {
					Address string `json:"address"`
				} `json:"targets"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			nextID++
			addr := ""
			if len(body.Targets) > 0 {
				addr = body.Targets[0].Address
			}
			policies[nextID] = &policy{
				id:         nextID,
				name:       body.Name,
				address:    addr,
				eventTypes: body.EventTypes,
				enabled:    body.Enabled,
			}
			w.WriteHeader(http.StatusCreated)
		case http.MethodGet:
			items := make([]string, 0, len(policies))
			for _, p := range policies {
				items = append(items, policyJSON(p))
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte("[" + strings.Join(items, ",") + "]"))
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	// /api/v2.0/projects/myproject/webhook/policies/<id> — get, put, delete
	mux.HandleFunc("/api/v2.0/projects/myproject/webhook/policies/", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		idStr := strings.TrimPrefix(r.URL.Path, "/api/v2.0/projects/myproject/webhook/policies/")
		id, _ := strconv.Atoi(idStr)
		p, ok := policies[id]
		switch r.Method {
		case http.MethodGet:
			if !ok {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"errors":[{"code":"NOT_FOUND","message":"webhook policy not found"}]}`))
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(policyJSON(p)))
		case http.MethodPut:
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			var body struct {
				Enabled    bool     `json:"enabled"`
				EventTypes []string `json:"event_types"`
				Targets    []struct {
					Address string `json:"address"`
				} `json:"targets"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			p.enabled = body.Enabled
			p.eventTypes = body.EventTypes
			if len(body.Targets) > 0 {
				p.address = body.Targets[0].Address
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

func TestWebhookClient_RealCRUD(t *testing.T) {
	srv := fakeHarborWebhooks(t)
	defer srv.Close()
	c := newTestClient(t, srv.URL)
	ctx := context.Background()
	proj := "myproject"

	// List before creation -> empty.
	list, err := c.ListWebhooks(ctx, proj)
	if err != nil {
		t.Fatalf("ListWebhooks before create: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("expected empty list before create, got %d items", len(list))
	}

	// Create -> returns the authoritative policy ID (not the stub hardcoded "1").
	st, err := c.CreateWebhook(ctx, &WebhookSpec{
		ProjectID:  proj,
		Name:       "proof-hook",
		URL:        "https://listener.example.com/events",
		EventTypes: []string{"PUSH_ARTIFACT", "DELETE_ARTIFACT"},
	})
	if err != nil {
		t.Fatalf("CreateWebhook: %v", err)
	}
	if st.ID != "10" {
		t.Errorf("expected real policy ID 10 from API (stub would be \"1\"), got %q", st.ID)
	}
	if st.Name != "proof-hook" {
		t.Errorf("expected name proof-hook, got %q", st.Name)
	}
	if st.URL != "https://listener.example.com/events" {
		t.Errorf("expected URL round-trip, got %q", st.URL)
	}
	if len(st.EventTypes) != 2 {
		t.Errorf("expected 2 event types, got %d", len(st.EventTypes))
	}

	// Get by ID -> exists.
	got, err := c.GetWebhook(ctx, proj, st.ID)
	if err != nil || got == nil {
		t.Fatalf("GetWebhook after create: got=%v err=%v", got, err)
	}
	if got.ID != st.ID {
		t.Errorf("GetWebhook ID mismatch: %q != %q", got.ID, st.ID)
	}

	// Update URL -> confirm round-trip via Get.
	updated, err := c.UpdateWebhook(ctx, proj, st.ID, &WebhookSpec{
		ProjectID:  proj,
		Name:       "proof-hook",
		URL:        "https://new-listener.example.com/events",
		EventTypes: []string{"PUSH_ARTIFACT"},
	})
	if err != nil {
		t.Fatalf("UpdateWebhook: %v", err)
	}
	if updated.URL != "https://new-listener.example.com/events" {
		t.Errorf("expected updated URL, got %q", updated.URL)
	}

	// Delete -> gone (idempotent on second call).
	if err := c.DeleteWebhook(ctx, proj, st.ID); err != nil {
		t.Fatalf("DeleteWebhook: %v", err)
	}
	if got, err := c.GetWebhook(ctx, proj, st.ID); err != nil || got != nil {
		t.Fatalf("expected (nil,nil) after delete, got got=%v err=%v", got, err)
	}
	// Second delete must be idempotent (404 -> nil).
	if err := c.DeleteWebhook(ctx, proj, st.ID); err != nil {
		t.Fatalf("idempotent DeleteWebhook: %v", err)
	}
}
