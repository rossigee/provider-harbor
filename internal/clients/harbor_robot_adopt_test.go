package clients

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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
