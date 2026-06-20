/*
Proof test for the real Project client implementation.

Runs the actual HarborClient.{Create,Get,Update,Delete}Project methods against a
stateful in-memory fake of Harbor's /api/v2.0 project API (httptest). This
exercises the real goharbor request/response path — it does NOT hit a live
Harbor and uses no real credentials. It demonstrates the methods are genuinely
implemented (issue the right HTTP verbs/paths, parse the real project ID and
visibility, and map 404 -> not-found) rather than returning hardcoded stubs.
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

func fakeHarbor(t *testing.T) *httptest.Server {
	t.Helper()
	var mu sync.Mutex
	// name -> public
	projects := map[string]bool{}
	nextID := 41

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2.0/projects", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		switch r.Method {
		case http.MethodPost:
			var body struct {
				ProjectName string `json:"project_name"`
				Public      *bool  `json:"public"`
				Metadata    *struct {
					Public string `json:"public"`
				} `json:"metadata"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			pub := body.Metadata != nil && strings.EqualFold(body.Metadata.Public, "true")
			projects[body.ProjectName] = pub
			nextID++
			w.Header().Set("Location", "/api/v2.0/projects/"+itoa(nextID))
			w.WriteHeader(http.StatusCreated)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/v2.0/projects/", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		name := strings.TrimPrefix(r.URL.Path, "/api/v2.0/projects/")
		pub, ok := projects[name]
		switch r.Method {
		case http.MethodGet:
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"errors":[{"code":"NOT_FOUND","message":"project not found"}]}`))
				return
			}
			pubStr := "false"
			if pub {
				pubStr = "true"
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"project_id":42,"name":"` + name + `","owner_name":"admin","repo_count":0,"metadata":{"public":"` + pubStr + `"}}`))
		case http.MethodPut:
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			var body struct {
				Metadata *struct {
					Public string `json:"public"`
				} `json:"metadata"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body.Metadata != nil {
				projects[name] = strings.EqualFold(body.Metadata.Public, "true")
			}
			w.WriteHeader(http.StatusOK)
		case http.MethodDelete:
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			delete(projects, name)
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	return httptest.NewServer(mux)
}

func itoa(i int) string {
	const d = "0123456789"
	if i == 0 {
		return "0"
	}
	var b []byte
	for i > 0 {
		b = append([]byte{d[i%10]}, b...)
		i /= 10
	}
	return string(b)
}

func newTestClient(t *testing.T, url string) *HarborClient {
	t.Helper()
	c, err := NewHarborClient(&HarborConfig{URL: url, Username: "admin", Password: "x", Insecure: true})
	if err != nil {
		t.Fatalf("NewHarborClient: %v", err)
	}
	return c
}

func TestProjectClient_RealCRUD(t *testing.T) {
	srv := fakeHarbor(t)
	defer srv.Close()
	c := newTestClient(t, srv.URL)
	ctx := context.Background()

	// Not found before creation -> (nil, nil).
	if st, err := c.GetProject(ctx, "proof-proj"); err != nil || st != nil {
		t.Fatalf("expected (nil,nil) before create, got st=%v err=%v", st, err)
	}

	// Create -> returns the authoritative ID + visibility parsed from Harbor.
	st, err := c.CreateProject(ctx, &ProjectSpec{Name: "proof-proj", Public: true})
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	if st.ID != "42" {
		t.Errorf("expected real project ID 42 from API, got %q (stub would be \"1\")", st.ID)
	}
	if !st.Public {
		t.Errorf("expected Public=true parsed from Harbor metadata")
	}
	if st.Name != "proof-proj" {
		t.Errorf("expected name proof-proj, got %q", st.Name)
	}

	// Get -> exists now.
	if _, err := c.GetProject(ctx, "proof-proj"); err != nil {
		t.Fatalf("GetProject after create: %v", err)
	}

	// Update visibility public->private, then confirm it round-trips.
	if _, err := c.UpdateProject(ctx, "proof-proj", &ProjectSpec{Name: "proof-proj", Public: false}); err != nil {
		t.Fatalf("UpdateProject: %v", err)
	}
	st2, err := c.GetProject(ctx, "proof-proj")
	if err != nil {
		t.Fatalf("GetProject after update: %v", err)
	}
	if st2.Public {
		t.Errorf("expected Public=false after update")
	}

	// Delete -> gone (drift path: subsequent Get must be not-found).
	if err := c.DeleteProject(ctx, "proof-proj"); err != nil {
		t.Fatalf("DeleteProject: %v", err)
	}
	if st, err := c.GetProject(ctx, "proof-proj"); err != nil || st != nil {
		t.Fatalf("expected (nil,nil) after delete, got st=%v err=%v", st, err)
	}
}
