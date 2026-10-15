package clients

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestDeleteRobotNotFound ensures deleting an already-absent robot (e.g. its
// parent project was deleted first) succeeds instead of wedging the managed
// reconciler's finalizer loop.
func TestDeleteRobotNotFound(t *testing.T) {
	tests := []struct {
		name        string
		status      int
		expectError bool
	}{
		{name: "robot already gone returns success", status: http.StatusNotFound, expectError: false},
		{name: "server error still surfaces", status: http.StatusInternalServerError, expectError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/api/v2.0/robots/") {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(tt.status)
					_, _ = w.Write([]byte(`{"errors":[{"code":"NOT_FOUND","message":"robot not found"}]}`))
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

			err = c.DeleteRobot(context.Background(), "123")
			if tt.expectError && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("expected nil error for already-deleted robot, got: %v", err)
			}
		})
	}
}
