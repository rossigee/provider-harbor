package clients

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pkg/errors"
)

// TestCreateRobotConflictAdoptionSystem covers 409 on system robot create:
// client lists robots and adopts on exact / robot$ / robot- name match.
func TestCreateRobotConflictAdoptionSystem(t *testing.T) {
	var sawPost, sawList int

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/api/v2.0/robots" {
			sawPost++
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write([]byte(`{"errors":[{"code":"CONFLICT","message":"robot account already exists"}]}`))
			return
		}
		if r.Method == http.MethodGet && r.URL.Path == "/api/v2.0/robots" {
			sawList++
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[{"id":7,"name":"robot$my-robot","level":"system","description":"adopted system robot"}]`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c, err := NewHarborClient(&HarborConfig{
		URL:      srv.URL,
		Username: "admin",
		Password: "Harbor12345",
		Insecure: true,
	})
	if err != nil {
		t.Fatalf("NewHarborClient returned error: %v", err)
	}

	desc := "Robot for test"
	robot, err := c.CreateRobot(context.Background(), &RobotSpec{
		Name:        "my-robot",
		Description: &desc,
	})
	if err != nil {
		t.Fatalf("CreateRobot on 409 conflict must adopt without error, got: %v", err)
	}
	if sawPost != 1 || sawList != 1 {
		t.Errorf("expected 1 POST and 1 GET, got post=%d list=%d", sawPost, sawList)
	}
	if robot == nil {
		t.Fatal("CreateRobot returned nil robot")
	}
	if robot.ID != "7" {
		t.Errorf("adopted robot ID = %q, want %q", robot.ID, "7")
	}
	if robot.Name != "robot$my-robot" {
		t.Errorf("adopted robot Name = %q, want %q", robot.Name, "robot$my-robot")
	}
}

// TestCreateRobotConflictAdoptionProject covers 409 on project robot create path.
// Note: current 409 logic in createProjectRobot calls ListRobots(ctx, nil) (not project scoped).
func TestCreateRobotConflictAdoptionProject(t *testing.T) {
	var sawPost, sawList, sawProjectGet int
	projectRef := "123"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/v2.0/projects/") {
			sawProjectGet++
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"project_id":123,"name":"testproj","metadata":{"public":"false"}}`))
			return
		}
		if r.Method == http.MethodPost && r.URL.Path == "/api/v2.0/robots" {
			sawPost++
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write([]byte(`{"errors":[{"code":"CONFLICT","message":"robot account already exists"}]}`))
			return
		}
		if r.Method == http.MethodGet && r.URL.Path == "/api/v2.0/robots" {
			sawList++
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			// list returns a name form that the (simple) adopt matcher will hit
			_, _ = w.Write([]byte(`[{"id":42,"name":"robot$my-proj-robot","level":"project","description":""}]`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c, err := NewHarborClient(&HarborConfig{
		URL:      srv.URL,
		Username: "admin",
		Password: "Harbor12345",
		Insecure: true,
	})
	if err != nil {
		t.Fatalf("NewHarborClient returned error: %v", err)
	}

	pid := projectRef
	robot, err := c.CreateRobot(context.Background(), &RobotSpec{
		Name:      "my-proj-robot",
		ProjectID: &pid,
		Permissions: []RobotPermission{
			{Namespace: "repository", Access: []string{"pull"}},
		},
	})
	if err != nil {
		t.Fatalf("CreateRobot project on 409 must adopt, got error: %v", err)
	}
	if sawPost != 1 {
		t.Errorf("expected 1 create POST, got %d", sawPost)
	}
	if sawList < 1 {
		t.Error("expected list call during adopt")
	}
	if sawProjectGet < 1 {
		t.Error("expected project resolve GET")
	}
	if robot == nil || robot.ID != "42" {
		t.Fatalf("expected adopted id 42, got %+v", robot)
	}
}

// TestCreateRobotConflictNoExactMatchReturnsStub verifies that when list succeeds
// but no name variant matches, we return a stub (ID empty) instead of error.
// This prevents reconcile failure; later Observe will decide.
func TestCreateRobotConflictNoExactMatchReturnsStub(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/api/v2.0/robots" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write([]byte(`{"errors":[{"code":"CONFLICT"}]}`))
			return
		}
		if r.Method == http.MethodGet && r.URL.Path == "/api/v2.0/robots" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			// foreign robot with similar suffix but not exact match variants
			_, _ = w.Write([]byte(`[{"id":9,"name":"robot$prod-my-robot","level":"system"}]`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c, err := NewHarborClient(&HarborConfig{
		URL:      srv.URL,
		Username: "admin",
		Password: "Harbor12345",
		Insecure: true,
	})
	if err != nil {
		t.Fatalf("NewHarborClient returned error: %v", err)
	}

	robot, err := c.CreateRobot(context.Background(), &RobotSpec{Name: "my-robot"})
	if err != nil {
		t.Fatalf("CreateRobot on 409 with no match must succeed with stub, got: %v", err)
	}
	if robot == nil {
		t.Fatal("nil robot")
	}
	if robot.ID != "" {
		t.Errorf("expected empty ID for no-match stub, got %q", robot.ID)
	}
	if robot.Name != "my-robot" {
		t.Errorf("stub name = %q, want %q", robot.Name, "my-robot")
	}
}

// TestCreateRobotConflictListErrorReturnsStub ensures list failure during 409 adopt
// still returns a stub (not a hard error) so controller can proceed.
func TestCreateRobotConflictListErrorReturnsStub(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/api/v2.0/robots" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write([]byte(`{"errors":[{"code":"CONFLICT"}]}`))
			return
		}
		if r.Method == http.MethodGet && r.URL.Path == "/api/v2.0/robots" {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"errors":[{"message":"list failed during adopt"}]}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c, err := NewHarborClient(&HarborConfig{
		URL:      srv.URL,
		Username: "admin",
		Password: "Harbor12345",
		Insecure: true,
	})
	if err != nil {
		t.Fatalf("NewHarborClient returned error: %v", err)
	}

	robot, err := c.CreateRobot(context.Background(), &RobotSpec{Name: "my-robot"})
	if err != nil {
		t.Fatalf("CreateRobot must not propagate list error on 409, got: %v", err)
	}
	if robot.ID != "" || robot.Name != "my-robot" {
		t.Errorf("expected stub on list failure during adopt, got %+v", robot)
	}
}

// TestCreateRobotConflictMatchesAllNameVariants ensures the three match forms
// in adopt logic are exercised (exact, robot$, robot-).
func TestCreateRobotConflictMatchesAllNameVariants(t *testing.T) {
	cases := []struct {
		name       string
		listName   string
		specName   string
		wantAdopt  bool
	}{
		{"exact match", "my-robot", "my-robot", true},
		{"robot$ prefix", "robot$my-robot", "my-robot", true},
		{"robot- prefix", "robot-my-robot", "my-robot", true},
		{"no match foreign", "robot$other-my-robot", "my-robot", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodPost && r.URL.Path == "/api/v2.0/robots" {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusConflict)
					_, _ = w.Write([]byte(`{"errors":[{"code":"CONFLICT"}]}`))
					return
				}
				if r.Method == http.MethodGet && r.URL.Path == "/api/v2.0/robots" {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`[{"id":1,"name":"` + tc.listName + `","level":"system"}]`))
					return
				}
				w.WriteHeader(http.StatusNotFound)
			}))
			defer srv.Close()

			c, err := NewHarborClient(&HarborConfig{
				URL:      srv.URL,
				Username: "admin",
				Password: "Harbor12345",
				Insecure: true,
			})
			if err != nil {
				t.Fatalf("client: %v", err)
			}

			robot, err := c.CreateRobot(context.Background(), &RobotSpec{Name: tc.specName})
			if err != nil {
				t.Fatalf("CreateRobot: %v", err)
			}
			if tc.wantAdopt {
				if robot.ID == "" {
					t.Errorf("expected adoption for %s but got stub", tc.name)
				}
			} else {
				if robot.ID != "" {
					t.Errorf("expected stub (no adopt) for %s, got id=%s", tc.name, robot.ID)
				}
			}
		})
	}
}

// --- Additional coverage for success paths, validation, non-409 errors ---

// TestCreateRobotSuccessSystem covers the happy path for system-level robot creation.
func TestCreateRobotSuccessSystem(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/api/v2.0/robots" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{
				"id": 123,
				"name": "robot$flux-ci",
				"secret": "super-secret-123",
				"creation_time": "2026-11-19T10:00:00Z",
				"update_time": "2026-11-19T10:00:00Z",
				"description": "CI robot"
			}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c, err := NewHarborClient(&HarborConfig{
		URL:      srv.URL,
		Username: "admin",
		Password: "Harbor12345",
		Insecure: true,
	})
	if err != nil {
		t.Fatalf("NewHarborClient: %v", err)
	}

	desc := "CI robot for Flux"
	expires := int64(30)
	robot, err := c.CreateRobot(context.Background(), &RobotSpec{
		Name:        "flux-ci",
		Description: &desc,
		ExpiresIn:   &expires,
		Permissions: []RobotPermission{
			{Namespace: "library", Access: []string{"pull", "push"}},
		},
	})
	if err != nil {
		t.Fatalf("CreateRobot success: %v", err)
	}
	if robot.ID != "123" || robot.Name != "robot$flux-ci" || robot.Secret != "super-secret-123" {
		t.Errorf("unexpected success robot: %+v", robot)
	}
}

// TestCreateRobotSuccessProject covers happy path for project robot (with resolve).
func TestCreateRobotSuccessProject(t *testing.T) {
	var sawProjectGet int
	projectRef := "myproj"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/v2.0/projects/") {
			sawProjectGet++
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"project_id":55,"name":"myproj","metadata":{"public":"true"}}`))
			return
		}
		if r.Method == http.MethodPost && r.URL.Path == "/api/v2.0/robots" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{
				"id": 456,
				"name": "robot$myproj+puller",
				"secret": "proj-secret",
				"creation_time": "2026-11-19T11:00:00Z"
			}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c, err := NewHarborClient(&HarborConfig{
		URL:      srv.URL,
		Username: "admin",
		Password: "Harbor12345",
		Insecure: true,
	})
	if err != nil {
		t.Fatalf("client: %v", err)
	}

	pid := projectRef
	robot, err := c.CreateRobot(context.Background(), &RobotSpec{
		Name: "puller",
		ProjectID: &pid,
		Permissions: []RobotPermission{
			{Access: []string{"pull"}},
		},
	})
	if err != nil {
		t.Fatalf("project create success: %v", err)
	}
	if sawProjectGet < 1 {
		t.Error("expected project resolve")
	}
	if robot.ID != "456" || robot.ProjectID == nil || *robot.ProjectID != "myproj" {
		t.Errorf("bad project robot: %+v", robot)
	}
}

// TestCreateRobotValidationErrors covers early returns.
func TestCreateRobotValidationErrors(t *testing.T) {
	// Use a mock that satisfies resolves for project path so we reach permission checks.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/v2.0/projects/") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"project_id":1,"name":"p1","metadata":{"public":"false"}}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c, err := NewHarborClient(&HarborConfig{
		URL:      srv.URL,
		Username: "admin",
		Password: "Harbor12345",
		Insecure: true,
	})
	if err != nil {
		t.Fatalf("client: %v", err)
	}

	_, err = c.CreateRobot(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "spec is required") {
		t.Errorf("nil spec: %v", err)
	}

	_, err = c.CreateRobot(context.Background(), &RobotSpec{Name: ""})
	if err == nil || !strings.Contains(err.Error(), "name is required") {
		t.Errorf("empty name: %v", err)
	}

	pid := "p1"
	_, err = c.CreateRobot(context.Background(), &RobotSpec{
		Name:      "x",
		ProjectID: &pid,
		// no Permissions
	})
	if err == nil || !strings.Contains(err.Error(), "permission") {
		t.Errorf("no perms for project: %v", err)
	}

	emptyAccess := []string{}
	_, err = c.CreateRobot(context.Background(), &RobotSpec{
		Name:      "x",
		ProjectID: &pid,
		Permissions: []RobotPermission{{Access: emptyAccess}},
	})
	if err == nil || !strings.Contains(err.Error(), "access list cannot be empty") {
		t.Errorf("empty access: %v", err)
	}
}

// TestCreateRobotNonConflictError covers generic create failure (not 409).
func TestCreateRobotNonConflictError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/api/v2.0/robots" {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"errors":[{"message":"boom"}]}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c, _ := NewHarborClient(&HarborConfig{
		URL:      srv.URL, Username: "admin", Password: "Harbor12345", Insecure: true,
	})

	_, err := c.CreateRobot(context.Background(), &RobotSpec{Name: "failme"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "failed to create robot account") {
		t.Errorf("wrong error: %v", err)
	}
}

// TestCreateRobotProjectResolveError covers resolve failures before the POST in project path.
func TestCreateRobotProjectResolveError(t *testing.T) {
	projectRef := "gone"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/v2.0/projects/") {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"errors":[{"code":"NOT_FOUND"}]}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c, _ := NewHarborClient(&HarborConfig{
		URL:      srv.URL, Username: "admin", Password: "Harbor12345", Insecure: true,
	})

	pid := projectRef
	_, err := c.CreateRobot(context.Background(), &RobotSpec{
		Name:      "x",
		ProjectID: &pid,
		Permissions: []RobotPermission{{Access: []string{"pull"}}},
	})
	if err == nil || !strings.Contains(err.Error(), "failed to resolve project") {
		t.Errorf("expected resolve error, got: %v", err)
	}
}

// TestIsConflictErr exercises the conflict detection helper (used by 409 adoption).
func TestIsConflictErr(t *testing.T) {
	if isConflictErr(nil) {
		t.Error("nil is not conflict")
	}
	// string form (what our httptest + SDK typically produces)
	err409 := errors.New("response status code does not match ... (status 409): {}")
	if !isConflictErr(err409) {
		t.Error("string 409 should be detected")
	}
	errOther := errors.New("status 500")
	if isConflictErr(errOther) {
		t.Error("500 should not be conflict")
	}
}

// TestCreateRobotProjectCustomResourceAndExpires covers the resource remapping
// logic and ExpiresIn in the project create path (non-adopt).
func TestCreateRobotProjectCustomResourceAndExpires(t *testing.T) {
	projectRef := "proj42"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/v2.0/projects/") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"project_id":42,"name":"proj42","metadata":{"public":"false"}}`))
			return
		}
		if r.Method == http.MethodPost && r.URL.Path == "/api/v2.0/robots" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":789,"name":"robot$proj42+custom","secret":"s","creation_time":"2026-11-19T12:00:00Z"}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c, _ := NewHarborClient(&HarborConfig{
		URL:      srv.URL,
		Username: "admin",
		Password: "Harbor12345",
		Insecure: true,
	})

	expires := int64(90)
	pid := projectRef
	robot, err := c.CreateRobot(context.Background(), &RobotSpec{
		Name:        "custom",
		ProjectID:   &pid,
		ExpiresIn:   &expires,
		Description: func() *string { s := "custom res"; return &s }(),
		Permissions: []RobotPermission{
			{Namespace: "my-custom-ns", Access: []string{"pull"}},
		},
	})
	if err != nil {
		t.Fatalf("project create with custom resource: %v", err)
	}
	if robot.ID != "789" {
		t.Errorf("bad id: %s", robot.ID)
	}
}

// TestCreateRobotProjectAdoptRobotDashPrefix covers 409 adopt in project path
// matching the "robot-" form (after trim).
func TestCreateRobotProjectAdoptRobotDashPrefix(t *testing.T) {
	projectRef := "p"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/v2.0/projects/") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"project_id":1,"name":"p","metadata":{"public":"false"}}`))
			return
		}
		if r.Method == http.MethodPost && r.URL.Path == "/api/v2.0/robots" {
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write([]byte(`{"errors":[{"code":"CONFLICT"}]}`))
			return
		}
		if r.Method == http.MethodGet && r.URL.Path == "/api/v2.0/robots" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[{"id":55,"name":"robot-myprojrobot","level":"project"}]`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c, _ := NewHarborClient(&HarborConfig{
		URL:      srv.URL, Username: "admin", Password: "Harbor12345", Insecure: true,
	})

	pid := projectRef
	robot, err := c.CreateRobot(context.Background(), &RobotSpec{
		Name:        "myprojrobot",
		ProjectID:   &pid,
		Permissions: []RobotPermission{{Access: []string{"pull"}}},
	})
	if err != nil {
		t.Fatalf("adopt project robot-: %v", err)
	}
	if robot.ID != "55" {
		t.Errorf("expected adopt id 55, got %s", robot.ID)
	}
}

// TestCreateRobotProjectNonConflictError covers the project-specific error path
// (not 409).
func TestCreateRobotProjectNonConflictError(t *testing.T) {
	projectRef := "p"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/v2.0/projects/") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"project_id":1,"name":"p"}`))
			return
		}
		if r.Method == http.MethodPost && r.URL.Path == "/api/v2.0/robots" {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"errors":[{"message":"proj boom"}]}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c, _ := NewHarborClient(&HarborConfig{
		URL:      srv.URL, Username: "admin", Password: "Harbor12345", Insecure: true,
	})

	pid := projectRef
	_, err := c.CreateRobot(context.Background(), &RobotSpec{
		Name:        "x",
		ProjectID:   &pid,
		Permissions: []RobotPermission{{Access: []string{"pull"}}},
	})
	if err == nil || !strings.Contains(err.Error(), "failed to create project robot account") {
		t.Errorf("expected project error wrap, got: %v", err)
	}
}
