package clients

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestCreateRobotConflictAdoption covers system robot creation when Harbor
// answers 409: only an exact name match is adopted, and anything else is an
// error rather than a success with an empty ID.
func TestCreateRobotConflictAdoption(t *testing.T) {
	tests := []struct {
		name       string
		listStatus int
		listBody   string
		wantID     string
		wantErr    bool
	}{
		{
			name:       "exact robot$ name is adopted",
			listStatus: http.StatusOK,
			listBody:   `[{"id":7,"name":"robot$ci","description":""}]`,
			wantID:     "7",
		},
		{
			name:       "foreign robot with same suffix is not adopted",
			listStatus: http.StatusOK,
			listBody:   `[{"id":9,"name":"robot$prod-ci","description":""}]`,
			wantErr:    true,
		},
		{
			name:       "no matching robot is an error",
			listStatus: http.StatusOK,
			listBody:   `[]`,
			wantErr:    true,
		},
		{
			name:       "list failure is an error",
			listStatus: http.StatusInternalServerError,
			listBody:   `{"errors":[{"code":"UNKNOWN","message":"boom"}]}`,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch {
				case r.Method == http.MethodPost && r.URL.Path == "/api/v2.0/robots":
					w.WriteHeader(http.StatusConflict)
					_, _ = w.Write([]byte(`{"errors":[{"code":"CONFLICT","message":"robot already exists"}]}`))
				case r.Method == http.MethodGet && r.URL.Path == "/api/v2.0/robots":
					w.WriteHeader(tt.listStatus)
					_, _ = w.Write([]byte(tt.listBody))
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

			status, err := c.CreateRobot(context.Background(), &RobotSpec{Name: "ci"})
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got status %+v", status)
				}
				return
			}
			if err != nil {
				t.Fatalf("expected adoption, got error: %v", err)
			}
			if status.ID != tt.wantID {
				t.Errorf("expected ID %q, got %q", tt.wantID, status.ID)
			}
		})
	}
}
