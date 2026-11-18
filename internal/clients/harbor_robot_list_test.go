package clients

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestListRobotsSystemFiltersByLevel guards the fix for the reconcile storm
// where CreateRobot got a 409 ("robot account 0:robot-x already exists") but
// the adoption list did not contain the robot: Harbor's GET /robots serves a
// project-scoped view unless q=Level=system is requested, so system robots
// were invisible and every robot reconcile failed forever.
func TestListRobotsSystemFiltersByLevel(t *testing.T) {
	var gotQ string
	var gotPageSize string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/api/v2.0/robots" {
			gotQ = r.URL.Query().Get("q")
			gotPageSize = r.URL.Query().Get("page_size")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[
				{"id":1,"name":"robot$robot-flux","level":"system","description":"system robot"},
				{"id":2,"name":"robot$charts+proj-robot","level":"project","description":"project robot"}
			]`))
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

	robots, err := c.ListRobots(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListRobots returned error: %v", err)
	}

	if gotQ != "Level=system" {
		t.Errorf("system ListRobots must filter with q=Level=system, got q=%q", gotQ)
	}
	if gotPageSize == "" {
		t.Error("system ListRobots must set an explicit page_size")
	}

	if len(robots) != 1 {
		t.Fatalf("expected only the system-level robot, got %d entries: %v", len(robots), robots)
	}
	if robots[0].Name != "robot$robot-flux" {
		t.Errorf("unexpected robot name %q", robots[0].Name)
	}
	if robots[0].ID != "1" {
		t.Errorf("unexpected robot id %q", robots[0].ID)
	}
}

// TestAdoptExistingRobotSystemRobot covers the exact production failure:
// Harbor answers 409 on POST /robots for an existing system robot, adoption
// must resolve it via the level-filtered list instead of erroring.
func TestAdoptExistingRobotSystemRobot(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v2.0/robots":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write([]byte(`{"errors":[{"code":"CONFLICT","message":"robot account 0:robot-flux already exists"}]}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v2.0/robots":
			if r.URL.Query().Get("q") != "Level=system" {
				// Harbor's unfiltered default view omits system robots.
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`[]`))
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[{"id":7,"name":"robot$robot-flux","level":"system"}]`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
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

	name := "robot-flux"
	desc := "Robot account for Flux GitOps controller to pull images"
	robot, err := c.CreateRobot(context.Background(), &RobotSpec{
		Name:        name,
		Description: &desc,
	})
	if err != nil {
		t.Fatalf("CreateRobot on 409 conflict must adopt, got error: %v", err)
	}
	if robot == nil {
		t.Fatal("CreateRobot returned nil robot")
	}
	if robot.ID != "7" {
		t.Errorf("adopted robot ID = %q, want %q", robot.ID, "7")
	}
	if robot.Name != "robot$robot-flux" {
		t.Errorf("adopted robot Name = %q, want %q", robot.Name, "robot$robot-flux")
	}
}
