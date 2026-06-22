/*
Proof tests for the real Robot and Member client implementations.

Each runs the actual HarborClient methods against a stateful in-memory fake of
Harbor's /api/v2.0 API (httptest), exercising the real goharbor request/response
path. They demonstrate the methods are genuinely implemented — issue the right
HTTP verbs/paths, parse the real ids/secret, and map 404 -> not-found — rather
than returning hardcoded stubs. No live Harbor, no real credentials.
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

func fakeHarborRobots(t *testing.T) *httptest.Server {
	t.Helper()
	var mu sync.Mutex
	type robot struct {
		id          int
		name        string
		description string
		namespace   string // project name carried on the permission
	}
	robots := map[int]*robot{}
	nextID := 67

	robotJSON := func(rb *robot) string {
		return fmt.Sprintf(`{"id":%d,"name":%q,"description":%q,"creation_time":"2026-01-01T00:00:00Z","update_time":"2026-01-01T00:00:00Z","permissions":[{"kind":"project","namespace":%q,"access":[{"resource":"repository","action":"pull"}]}]}`,
			rb.id, rb.name, rb.description, rb.namespace)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2.0/robots", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		switch r.Method {
		case http.MethodPost:
			var body struct {
				Name        string `json:"name"`
				Description string `json:"description"`
				Permissions []struct {
					Namespace string `json:"namespace"`
				} `json:"permissions"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			nextID++
			ns := ""
			if len(body.Permissions) > 0 {
				ns = body.Permissions[0].Namespace
			}
			full := fmt.Sprintf("robot$%s+%s", ns, body.Name)
			robots[nextID] = &robot{id: nextID, name: full, description: body.Description, namespace: ns}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = fmt.Fprintf(w, `{"id":%d,"name":%q,"secret":"generated-secret-xyz","creation_time":"2026-01-01T00:00:00Z"}`, nextID, full)
		case http.MethodGet:
			items := make([]string, 0, len(robots))
			for _, rb := range robots {
				items = append(items, robotJSON(rb))
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte("[" + strings.Join(items, ",") + "]"))
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/v2.0/robots/", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		id, _ := strconv.Atoi(strings.TrimPrefix(r.URL.Path, "/api/v2.0/robots/"))
		rb, ok := robots[id]
		switch r.Method {
		case http.MethodGet:
			if !ok {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"errors":[{"code":"NOT_FOUND","message":"robot not found"}]}`))
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(robotJSON(rb)))
		case http.MethodPut:
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			var body struct {
				Description string `json:"description"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			rb.description = body.Description
			w.WriteHeader(http.StatusOK)
		case http.MethodDelete:
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			delete(robots, id)
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	return httptest.NewServer(mux)
}

func TestRobotClient_RealCRUD(t *testing.T) {
	srv := fakeHarborRobots(t)
	defer srv.Close()
	c := newTestClient(t, srv.URL)
	ctx := context.Background()
	proj := "tenant-acme"

	// Not found before creation -> (nil, nil).
	if st, err := c.GetRobot(ctx, "999"); err != nil || st != nil {
		t.Fatalf("expected (nil,nil) for missing robot, got st=%v err=%v", st, err)
	}

	// Create -> returns the authoritative ID and the one-time secret.
	st, err := c.CreateRobot(ctx, &RobotSpec{
		Name:        "ci",
		ProjectID:   &proj,
		Permissions: []RobotPermission{{Namespace: "repository", Access: []string{"pull", "push"}}},
	})
	if err != nil {
		t.Fatalf("CreateRobot: %v", err)
	}
	if st.ID != "68" {
		t.Errorf("expected real robot ID 68 from API, got %q (stub would be \"1\")", st.ID)
	}
	if st.Secret != "generated-secret-xyz" {
		t.Errorf("expected real one-time secret from API, got %q (stub would be \"robot-secret-token\")", st.Secret)
	}

	// Get by id -> exists, project parsed from permission namespace.
	got, err := c.GetRobot(ctx, st.ID)
	if err != nil || got == nil {
		t.Fatalf("GetRobot after create: st=%v err=%v", got, err)
	}
	if got.ProjectID == nil || *got.ProjectID != proj {
		t.Errorf("expected project %q parsed from permission, got %v", proj, got.ProjectID)
	}

	// List scoped to the project finds it by suffix-able full name.
	robots, err := c.ListRobots(ctx, &proj)
	if err != nil {
		t.Fatalf("ListRobots: %v", err)
	}
	if len(robots) != 1 || !strings.HasSuffix(robots[0].Name, "+ci") {
		t.Errorf("expected one robot with name suffix +ci, got %+v", robots)
	}

	// Delete -> gone.
	if err := c.DeleteRobot(ctx, st.ID); err != nil {
		t.Fatalf("DeleteRobot: %v", err)
	}
	if got, err := c.GetRobot(ctx, st.ID); err != nil || got != nil {
		t.Fatalf("expected (nil,nil) after delete, got st=%v err=%v", got, err)
	}
	// Delete is idempotent.
	if err := c.DeleteRobot(ctx, st.ID); err != nil {
		t.Fatalf("idempotent DeleteRobot: %v", err)
	}
}

func fakeHarborMembers(t *testing.T) *httptest.Server {
	t.Helper()
	var mu sync.Mutex
	type member struct {
		id       int
		name     string
		roleID   int
		entityTp string
	}
	members := map[int]*member{}
	nextID := 0

	memberJSON := func(m *member) string {
		return fmt.Sprintf(`{"id":%d,"entity_name":%q,"entity_type":%q,"role_id":%d,"project_id":1}`,
			m.id, m.name, m.entityTp, m.roleID)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2.0/projects/1/members", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		switch r.Method {
		case http.MethodPost:
			var body struct {
				RoleID     int `json:"role_id"`
				MemberUser *struct {
					Username string `json:"username"`
				} `json:"member_user"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			nextID++
			name := ""
			if body.MemberUser != nil {
				name = body.MemberUser.Username
			}
			members[nextID] = &member{id: nextID, name: name, roleID: body.RoleID, entityTp: "u"}
			w.Header().Set("Location", "/api/v2.0/projects/1/members/"+itoa(nextID))
			w.WriteHeader(http.StatusCreated)
		case http.MethodGet:
			items := make([]string, 0, len(members))
			for _, m := range members {
				items = append(items, memberJSON(m))
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte("[" + strings.Join(items, ",") + "]"))
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/v2.0/projects/1/members/", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		id, _ := strconv.Atoi(strings.TrimPrefix(r.URL.Path, "/api/v2.0/projects/1/members/"))
		m, ok := members[id]
		switch r.Method {
		case http.MethodPut:
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			var body struct {
				RoleID int `json:"role_id"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			m.roleID = body.RoleID
			w.WriteHeader(http.StatusOK)
		case http.MethodDelete:
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			delete(members, id)
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	return httptest.NewServer(mux)
}

func TestMemberClient_RealCRUD(t *testing.T) {
	srv := fakeHarborMembers(t)
	defer srv.Close()
	c := newTestClient(t, srv.URL)
	ctx := context.Background()

	// Not found before creation -> (nil, nil).
	if st, err := c.GetProjectMember(ctx, "1", "alice"); err != nil || st != nil {
		t.Fatalf("expected (nil,nil) before add, got st=%v err=%v", st, err)
	}

	// Add -> developer (role_id 2).
	if err := c.AddProjectMember(ctx, "1", "alice", "developer"); err != nil {
		t.Fatalf("AddProjectMember: %v", err)
	}

	// Get -> resolves real numeric id, maps role_id 2 back to "developer".
	st, err := c.GetProjectMember(ctx, "1", "alice")
	if err != nil || st == nil {
		t.Fatalf("GetProjectMember after add: st=%v err=%v", st, err)
	}
	if st.ID != "1" {
		t.Errorf("expected real member id 1, got %q", st.ID)
	}
	if st.Role != "developer" {
		t.Errorf("expected role developer mapped from role_id 2, got %q", st.Role)
	}
	if st.MemberType != "user" {
		t.Errorf("expected member type user (from entity_type u), got %q", st.MemberType)
	}

	// Update role developer -> maintainer (role_id 4), confirm round-trip.
	if err := c.UpdateProjectMember(ctx, "1", "alice", "maintainer"); err != nil {
		t.Fatalf("UpdateProjectMember: %v", err)
	}
	st2, err := c.GetProjectMember(ctx, "1", "alice")
	if err != nil || st2 == nil {
		t.Fatalf("GetProjectMember after update: st=%v err=%v", st2, err)
	}
	if st2.Role != "maintainer" {
		t.Errorf("expected role maintainer after update, got %q", st2.Role)
	}

	// Unknown role -> validation error, no API call.
	if err := c.AddProjectMember(ctx, "1", "bob", "wizard"); err == nil {
		t.Errorf("expected error for unknown role")
	}

	// Delete -> gone, idempotent.
	if err := c.DeleteProjectMember(ctx, "1", "alice"); err != nil {
		t.Fatalf("DeleteProjectMember: %v", err)
	}
	if st, err := c.GetProjectMember(ctx, "1", "alice"); err != nil || st != nil {
		t.Fatalf("expected (nil,nil) after delete, got st=%v err=%v", st, err)
	}
	if err := c.DeleteProjectMember(ctx, "1", "alice"); err != nil {
		t.Fatalf("idempotent DeleteProjectMember: %v", err)
	}
}
