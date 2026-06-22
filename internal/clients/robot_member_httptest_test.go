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

	"k8s.io/utils/ptr"
)

// robotInspector reports the recorded level and per-permission kinds of a created
// robot by id, so tests can assert what the client actually sent to Harbor.
type robotInspector func(id int) (level string, kinds []string, ok bool)

func fakeHarborRobots(t *testing.T) (*httptest.Server, robotInspector) {
	t.Helper()
	var mu sync.Mutex
	type robot struct {
		id          int
		name        string
		description string
		namespace   string // project name carried on the permission
		level       string
		permKinds   []string // the kind of each created permission, in order
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
				Level       string `json:"level"`
				Permissions []struct {
					Kind      string `json:"kind"`
					Namespace string `json:"namespace"`
				} `json:"permissions"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			nextID++
			ns := ""
			if len(body.Permissions) > 0 {
				ns = body.Permissions[0].Namespace
			}
			kinds := make([]string, 0, len(body.Permissions))
			for _, p := range body.Permissions {
				kinds = append(kinds, p.Kind)
			}
			full := fmt.Sprintf("robot$%s+%s", ns, body.Name)
			robots[nextID] = &robot{id: nextID, name: full, description: body.Description, namespace: ns, level: body.Level, permKinds: kinds}
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
	// GET /projects/{id} resolves a numeric project id to its name, so a numeric
	// projectId in the spec is turned into the project NAME for the robot
	// permission namespace. Project 16 -> "tenant-acme".
	mux.HandleFunc("/api/v2.0/projects/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		idStr := strings.TrimPrefix(r.URL.Path, "/api/v2.0/projects/")
		w.Header().Set("Content-Type", "application/json")
		if idStr == "16" {
			_, _ = fmt.Fprint(w, `{"project_id":16,"name":"tenant-acme"}`)
			return
		}
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"errors":[{"code":"NOT_FOUND","message":"project not found"}]}`))
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
	inspect := func(id int) (string, []string, bool) {
		mu.Lock()
		defer mu.Unlock()
		rb, ok := robots[id]
		if !ok {
			return "", nil, false
		}
		return rb.level, rb.permKinds, true
	}
	return httptest.NewServer(mux), inspect
}

func TestRobotClient_RealCRUD(t *testing.T) {
	srv, inspect := fakeHarborRobots(t)
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

	// The client created a project-level robot with a project-kind permission.
	id, _ := strconv.Atoi(st.ID)
	if level, kinds, ok := inspect(id); !ok || level != "project" || len(kinds) != 1 || kinds[0] != "project" {
		t.Errorf("expected level=project with one project-kind permission, got level=%q kinds=%v ok=%v", level, kinds, ok)
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

// TestRobotClient_NumericProjectIDResolvedToName proves the field contract: a
// numeric projectId (the Harbor project id, e.g. "16") is resolved to the project
// NAME via GET /projects/{id} and that name is what lands in the robot permission
// namespace — not the literal "16" (which would 404 createRobotNotFound).
func TestRobotClient_NumericProjectIDResolvedToName(t *testing.T) {
	srv, _ := fakeHarborRobots(t)
	defer srv.Close()
	c := newTestClient(t, srv.URL)
	ctx := context.Background()

	numericID := "16"
	st, err := c.CreateRobot(ctx, &RobotSpec{
		Name:        "ci",
		ProjectID:   &numericID,
		Permissions: []RobotPermission{{Namespace: "repository", Access: []string{"pull"}}},
	})
	if err != nil {
		t.Fatalf("CreateRobot with numeric projectId: %v", err)
	}

	// Read it back: the observed ProjectID is the permission namespace, which must
	// be the resolved project NAME "tenant-acme", proving id->name resolution ran.
	got, err := c.GetRobot(ctx, st.ID)
	if err != nil || got == nil {
		t.Fatalf("GetRobot after create: st=%v err=%v", got, err)
	}
	if got.ProjectID == nil || *got.ProjectID != "tenant-acme" {
		t.Errorf("expected permission namespace resolved to project name %q, got %v (numeric id used verbatim would be %q)", "tenant-acme", got.ProjectID, numericID)
	}
	// The Harbor full name encodes the namespace as robot$<name>+<short>.
	if !strings.Contains(got.Name, "robot$tenant-acme+") {
		t.Errorf("expected robot full name to carry resolved project name, got %q", got.Name)
	}
}

// TestRobotClient_SystemLevel proves a system-level robot is created with
// Level=system and that each permission honours its own scope kind: a system-kind
// permission (namespace "/") and a project-kind permission (a specific project).
func TestRobotClient_SystemLevel(t *testing.T) {
	srv, inspect := fakeHarborRobots(t)
	defer srv.Close()
	c := newTestClient(t, srv.URL)
	ctx := context.Background()

	st, err := c.CreateRobot(ctx, &RobotSpec{
		Name:  "platform-ci",
		Level: "system",
		// No projectId: system robots are not scoped to a single project.
		Permissions: []RobotPermission{
			{Kind: ptr.To("system"), Namespace: "robot", Access: []string{"create"}},
			{Kind: ptr.To("project"), Scope: ptr.To("tenant-acme"), Namespace: "repository", Access: []string{"pull", "push"}},
		},
	})
	if err != nil {
		t.Fatalf("CreateRobot system: %v", err)
	}

	id, _ := strconv.Atoi(st.ID)
	level, kinds, ok := inspect(id)
	if !ok {
		t.Fatalf("robot %d not recorded", id)
	}
	if level != "system" {
		t.Errorf("expected level=system, got %q", level)
	}
	if len(kinds) != 2 || kinds[0] != "system" || kinds[1] != "project" {
		t.Errorf("expected permission kinds [system project], got %v", kinds)
	}

	// Get by id works the same way for a system robot.
	if got, err := c.GetRobot(ctx, st.ID); err != nil || got == nil {
		t.Fatalf("GetRobot system: st=%v err=%v", got, err)
	}
}

// TestRobotClient_CreateConflict proves a 409 from Harbor on create is mapped to
// an actionable "cannot be imported / delete the existing robot" error — robots
// are never adopted or recreated by the controller.
func TestRobotClient_CreateConflict(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2.0/robots", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"errors":[{"code":"CONFLICT","message":"robot already exists"}]}`))
	})
	// resolveProjectName -> GET /projects/{name}; return a project so resolution
	// succeeds and we reach the create call.
	mux.HandleFunc("/api/v2.0/projects/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"project_id":1,"name":"tenant-acme"}`)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	c := newTestClient(t, srv.URL)

	proj := "tenant-acme"
	_, err := c.CreateRobot(context.Background(), &RobotSpec{
		Name:        "ci",
		ProjectID:   &proj,
		Permissions: []RobotPermission{{Namespace: "repository", Access: []string{"pull"}}},
	})
	if err == nil {
		t.Fatal("expected an error on 409 create conflict")
	}
	if !strings.Contains(err.Error(), "cannot be imported") || !strings.Contains(err.Error(), "delete the existing robot") {
		t.Errorf("expected actionable import-conflict error, got %q", err.Error())
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
