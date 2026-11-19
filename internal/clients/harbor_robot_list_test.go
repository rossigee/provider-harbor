package clients

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
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

// TestListRobotsPaginatesMultiplePages ensures ListRobots follows pagination
// and collects robots from all pages when > page_size results exist.
func TestListRobotsPaginatesMultiplePages(t *testing.T) {
	var requestedPages []int64
	var gotQ string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/api/v2.0/robots" {
			gotQ = r.URL.Query().Get("q")
			pageStr := r.URL.Query().Get("page")
			page, _ := strconv.ParseInt(pageStr, 10, 64)
			requestedPages = append(requestedPages, page)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)

			if page == 1 {
				// full page of 100 system robots
				var items []string
				for i := 0; i < 100; i++ {
					items = append(items, fmt.Sprintf(`{"id":%d,"name":"robot-sys-%03d","level":"system"}`, i+1, i))
				}
				_, _ = w.Write([]byte("[" + strings.Join(items, ",") + "]"))
				return
			}
			// page 2: 37 more
			var items []string
			for i := 0; i < 37; i++ {
				items = append(items, fmt.Sprintf(`{"id":%d,"name":"robot-sys-%03d","level":"system"}`, 100+i+1, 100+i))
			}
			_, _ = w.Write([]byte("[" + strings.Join(items, ",") + "]"))
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
		t.Errorf("expected q=Level=system, got %q", gotQ)
	}
	if len(requestedPages) != 2 {
		t.Errorf("expected 2 pages, got %d: %v", len(requestedPages), requestedPages)
	}
	if len(robots) != 137 {
		t.Fatalf("expected 137 robots total, got %d", len(robots))
	}
}

// TestListRobotsStopsOnShortLastPage verifies early exit when a page returns
// fewer items than page_size.
func TestListRobotsStopsOnShortLastPage(t *testing.T) {
	var requestedPages []int64

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/api/v2.0/robots" {
			pageStr := r.URL.Query().Get("page")
			page, _ := strconv.ParseInt(pageStr, 10, 64)
			requestedPages = append(requestedPages, page)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)

			// only 5 robots total (short page)
			_, _ = w.Write([]byte(`[
				{"id":1,"name":"robot-1","level":"system"},
				{"id":2,"name":"robot-2","level":"system"},
				{"id":3,"name":"robot-3","level":"system"},
				{"id":4,"name":"robot-4","level":"system"},
				{"id":5,"name":"robot-5","level":"system"}
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

	if len(requestedPages) != 1 {
		t.Errorf("expected only 1 page when response is short, got %d", len(requestedPages))
	}
	if len(robots) != 5 {
		t.Fatalf("expected 5 robots, got %d", len(robots))
	}
}

// TestListRobotsProjectPaginates tests pagination for project-scoped robots
// via ListRobots with a projectID.
func TestListRobotsProjectPaginates(t *testing.T) {
	var requestedPages []int64
	var gotQ string

	projectID := "123"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/api/v2.0/robots" {
			gotQ = r.URL.Query().Get("q")
			pageStr := r.URL.Query().Get("page")
			page, _ := strconv.ParseInt(pageStr, 10, 64)
			requestedPages = append(requestedPages, page)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)

			if page == 1 {
				// return full page to force second request
				var items []string
				for i := 0; i < 100; i++ {
					items = append(items, fmt.Sprintf(`{"id":%d,"name":"proj-robot-%03d","level":"project"}`, 10+i, i))
				}
				_, _ = w.Write([]byte("[" + strings.Join(items, ",") + "]"))
				return
			}
			// page 2: 1 more
			_, _ = w.Write([]byte(`[{"id":110,"name":"proj-robot-110","level":"project"}]`))
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

	robots, err := c.ListRobots(context.Background(), &projectID)
	if err != nil {
		t.Fatalf("ListRobots (project) returned error: %v", err)
	}

	if !strings.Contains(gotQ, "ProjectID=123") {
		t.Errorf("expected project filter in q, got %q", gotQ)
	}
	if len(requestedPages) != 2 {
		t.Errorf("expected 2 pages for project robots, got %d", len(requestedPages))
	}
	if len(robots) != 101 {
		t.Fatalf("expected 101 project robots, got %d", len(robots))
	}
}

// TestListRobotsPaginationErrorOnPage2 ensures we surface errors from
// non-first pages.
func TestListRobotsPaginationErrorOnPage2(t *testing.T) {
	callCount := 0

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if r.Method == http.MethodGet && r.URL.Path == "/api/v2.0/robots" {
			w.Header().Set("Content-Type", "application/json")
			if callCount == 1 {
				// full first page
				items := make([]string, 100)
				for i := range items {
					items[i] = fmt.Sprintf(`{"id":%d,"name":"r%d","level":"system"}`, i+1, i)
				}
				_, _ = w.Write([]byte("[" + strings.Join(items, ",") + "]"))
				return
			}
			// error on page 2
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"errors":[{"message":"boom"}]}`))
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

	_, err = c.ListRobots(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error from page 2, got nil")
	}
}

// TestListRobotsEmptyResult verifies handling of no robots at all.
func TestListRobotsEmptyResult(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/api/v2.0/robots" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[]`))
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
	if len(robots) != 0 {
		t.Errorf("expected 0 robots, got %d", len(robots))
	}
}

// TestListRobotsFiltersNonSystemAcrossPages ensures filtering works when
// later pages contain mixed levels.
func TestListRobotsFiltersNonSystemAcrossPages(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/api/v2.0/robots" {
			pageStr := r.URL.Query().Get("page")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			if pageStr == "1" {
				// return enough to trigger page 2 (simulate full page with mixed)
				var items []string
				for i := 0; i < 100; i++ {
					level := "project"
					if i%2 == 0 {
						level = "system"
					}
					items = append(items, fmt.Sprintf(`{"id":%d,"name":"item-%03d","level":"%s"}`, i+1, i, level))
				}
				_, _ = w.Write([]byte("[" + strings.Join(items, ",") + "]"))
				return
			}
			// page 2 with more system
			_, _ = w.Write([]byte(`[
				{"id":101,"name":"sys-extra","level":"system"},
				{"id":102,"name":"other2","level":""}
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
	// From 100 on page 1 (50 system), + 1 system on page 2 = 51
	if len(robots) < 50 {
		t.Fatalf("expected at least 50 system robots after filtering across pages, got %d", len(robots))
	}
}

// TestListRobotsFullPageThenEmpty verifies the loop requests an extra page
// when a page returns exactly pageSize items (and the next returns empty).
func TestListRobotsFullPageThenEmpty(t *testing.T) {
	var requestedPages []int64

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/api/v2.0/robots" {
			pageStr := r.URL.Query().Get("page")
			page, _ := strconv.ParseInt(pageStr, 10, 64)
			requestedPages = append(requestedPages, page)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)

			if page == 1 {
				// exactly 100 items
				items := make([]string, 100)
				for i := range items {
					items[i] = fmt.Sprintf(`{"id":%d,"name":"r%d","level":"system"}`, i+1, i)
				}
				_, _ = w.Write([]byte("[" + strings.Join(items, ",") + "]"))
				return
			}
			// empty page 2
			_, _ = w.Write([]byte(`[]`))
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

	if len(requestedPages) != 2 {
		t.Errorf("expected 2 pages (full + empty), got %d", len(requestedPages))
	}
	if len(robots) != 100 {
		t.Fatalf("expected 100 robots, got %d", len(robots))
	}
}

// TestListRobotsProjectResolveError tests the error path when a project
// reference cannot be resolved (non-numeric name that doesn't exist).
func TestListRobotsProjectResolveError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/v2.0/projects/") {
			w.WriteHeader(http.StatusNotFound)
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

	projectID := "nonexistent-project"
	_, err = c.ListRobots(context.Background(), &projectID)
	if err == nil {
		t.Fatal("expected error for unresolvable project, got nil")
	}
	if !strings.Contains(err.Error(), "failed to resolve project") {
		t.Errorf("expected resolve error, got: %v", err)
	}
}
