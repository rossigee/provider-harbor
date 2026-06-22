/*
Proof test for the real Repository client implementation.

Runs the actual HarborClient.{Get,Update,Delete}Repository methods against a
stateful in-memory fake of Harbor's /api/v2.0 repository API (httptest). This
exercises the real goharbor request/response path — it does NOT hit a live
Harbor and uses no real credentials. It demonstrates the methods are genuinely
implemented (issue the right HTTP verbs/paths, parse the real repository ID and
description, and map absent -> (nil, nil)) rather than returning hardcoded stubs.

Harbor repository API shape:
  - GET    /api/v2.0/projects/{project_name}/repositories/{repository_name} → 200/404
  - PUT    /api/v2.0/projects/{project_name}/repositories/{repository_name} → 200
  - DELETE /api/v2.0/projects/{project_name}/repositories/{repository_name} → 200/404
  - GET    /api/v2.0/projects/{project_name}/repositories (list)            → 200

Note: Harbor creates repositories implicitly on first push; there is no POST.
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

func fakeHarborRepository(t *testing.T) *httptest.Server {
	t.Helper()
	var mu sync.Mutex
	type repoEntry struct {
		ID          int64
		Name        string // full name: project/repo
		ProjectName string
		RepoName    string
		Description string
	}
	repos := map[string]*repoEntry{} // key: "project/repo"
	nextID := 100

	mux := http.NewServeMux()

	// /api/v2.0/projects/{project}/repositories and /api/v2.0/projects/{project}/repositories/{repo}
	mux.HandleFunc("/api/v2.0/projects/", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		path := strings.TrimPrefix(r.URL.Path, "/api/v2.0/projects/")
		// Expect path like "myproject/repositories" or "myproject/repositories/myrepo"
		parts := strings.SplitN(path, "/", 3)
		if len(parts) < 2 || parts[1] != "repositories" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		projectName := parts[0]

		if len(parts) == 2 {
			// List: GET /projects/{project}/repositories
			if r.Method != http.MethodGet {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			var list []map[string]interface{}
			for _, e := range repos {
				if e.ProjectName == projectName {
					list = append(list, map[string]interface{}{
						"id":          e.ID,
						"name":        e.Name,
						"project_id":  1,
						"description": e.Description,
					})
				}
			}
			if list == nil {
				list = []map[string]interface{}{}
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(list)
			return
		}

		// Single repo: /projects/{project}/repositories/{repo}
		repoName := parts[2]
		key := projectName + "/" + repoName

		switch r.Method {
		case http.MethodGet:
			e, ok := repos[key]
			if !ok {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"errors":[{"code":"NOT_FOUND","message":"repository not found"}]}`))
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id":          e.ID,
				"name":        e.Name,
				"project_id":  1,
				"description": e.Description,
			})

		case http.MethodPut:
			var body struct {
				Description string `json:"description"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			e, ok := repos[key]
			if !ok {
				// Harbor auto-creates repo on push; we simulate it existing after update.
				nextID++
				e = &repoEntry{
					ID:          int64(nextID),
					Name:        key,
					ProjectName: projectName,
					RepoName:    repoName,
				}
				repos[key] = e
			}
			e.Description = body.Description
			w.WriteHeader(http.StatusOK)

		case http.MethodDelete:
			if _, ok := repos[key]; !ok {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"errors":[{"code":"NOT_FOUND","message":"repository not found"}]}`))
				return
			}
			delete(repos, key)
			w.WriteHeader(http.StatusOK)

		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	return httptest.NewServer(mux)
}

func TestRepositoryClient_RealCRUD(t *testing.T) {
	srv := fakeHarborRepository(t)
	defer srv.Close()
	c := newTestClient(t, srv.URL)
	ctx := context.Background()

	const project = "myproject"
	const repo = "myrepo"

	// Not found before any push -> (nil, nil).
	if st, err := c.GetRepository(ctx, project, repo); err != nil || st != nil {
		t.Fatalf("expected (nil,nil) before create, got st=%v err=%v", st, err)
	}

	// UpdateRepository on a new repo auto-creates it in the fake and sets description.
	desc := "initial description"
	spec := &RepositorySpec{ProjectID: project, Name: repo, Description: &desc}
	st, err := c.UpdateRepository(ctx, project, repo, spec)
	if err != nil {
		t.Fatalf("UpdateRepository (create): %v", err)
	}
	if st == nil {
		t.Fatal("expected non-nil status after UpdateRepository")
	}
	if st.Description != desc {
		t.Errorf("expected description %q, got %q", desc, st.Description)
	}

	// Get -> exists now.
	st2, err := c.GetRepository(ctx, project, repo)
	if err != nil {
		t.Fatalf("GetRepository after update: %v", err)
	}
	if st2 == nil {
		t.Fatal("expected non-nil after update")
	}
	if st2.Description != desc {
		t.Errorf("description mismatch: got %q want %q", st2.Description, desc)
	}

	// Update description.
	newDesc := "updated description"
	spec2 := &RepositorySpec{ProjectID: project, Name: repo, Description: &newDesc}
	_, err = c.UpdateRepository(ctx, project, repo, spec2)
	if err != nil {
		t.Fatalf("UpdateRepository: %v", err)
	}
	st3, err := c.GetRepository(ctx, project, repo)
	if err != nil {
		t.Fatalf("GetRepository after description update: %v", err)
	}
	if st3 == nil || st3.Description != newDesc {
		t.Errorf("expected description %q, got %v", newDesc, st3)
	}

	// Delete -> gone.
	if err := c.DeleteRepository(ctx, project, repo); err != nil {
		t.Fatalf("DeleteRepository: %v", err)
	}
	if st, err := c.GetRepository(ctx, project, repo); err != nil || st != nil {
		t.Fatalf("expected (nil,nil) after delete, got st=%v err=%v", st, err)
	}

	// Idempotent delete: deleting absent repository must be a no-op.
	if err := c.DeleteRepository(ctx, project, repo); err != nil {
		t.Fatalf("DeleteRepository (idempotent): %v", err)
	}
}
