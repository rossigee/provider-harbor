/*
Proof test for the real User client implementation.

Runs the actual HarborClient.{Create,Get,Update,Delete}User methods against a
stateful in-memory fake of Harbor's /api/v2.0 user API (httptest). This
exercises the real goharbor request/response path — it does NOT hit a live
Harbor and uses no real credentials. It demonstrates the methods are genuinely
implemented (issue the right HTTP verbs/paths, resolve the Harbor numeric
user_id from a username filter, and return (nil, nil) for absent users) rather
than returning hardcoded stubs.
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

func fakeHarborUsers(t *testing.T) *httptest.Server {
	t.Helper()
	var mu sync.Mutex
	type user struct {
		id       int
		username string
		email    string
		sysadmin bool
	}
	users := map[int]*user{}
	nextID := 100

	userJSON := func(u *user) string {
		sa := "false"
		if u.sysadmin {
			sa = "true"
		}
		return fmt.Sprintf(`{"user_id":%d,"username":%q,"email":%q,"sysadmin_flag":%s,"creation_time":"2026-01-01T00:00:00.000Z"}`,
			u.id, u.username, u.email, sa)
	}

	mux := http.NewServeMux()

	// POST /api/v2.0/users — create
	mux.HandleFunc("/api/v2.0/users", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		switch r.Method {
		case http.MethodPost:
			var body struct {
				Username string `json:"username"`
				Email    string `json:"email"`
				Password string `json:"password"`
				Realname string `json:"realname"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			nextID++
			users[nextID] = &user{id: nextID, username: body.Username, email: body.Email}
			w.Header().Set("Location", "/api/v2.0/users/"+itoa(nextID))
			w.WriteHeader(http.StatusCreated)
		case http.MethodGet:
			// ListUsers — supports ?q=username=<name>
			q := r.URL.Query().Get("q")
			var filterUsername string
			if strings.HasPrefix(q, "username=") {
				filterUsername = strings.TrimPrefix(q, "username=")
			}
			var items []string
			for _, u := range users {
				if filterUsername == "" || u.username == filterUsername {
					items = append(items, userJSON(u))
				}
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte("[" + strings.Join(items, ",") + "]"))
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	// /api/v2.0/users/<id>[/<subpath>] — get, update, delete, sysadmin
	// The SDK uses:
	//   GET    /users/<id>           — get user
	//   PUT    /users/<id>           — update profile (email etc.)
	//   PUT    /users/<id>/sysadmin  — set sysadmin flag
	//   DELETE /users/<id>           — delete user
	mux.HandleFunc("/api/v2.0/users/", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		path := strings.TrimPrefix(r.URL.Path, "/api/v2.0/users/")

		// Handle /api/v2.0/users/<id>/sysadmin
		if strings.HasSuffix(path, "/sysadmin") {
			idStr := strings.TrimSuffix(path, "/sysadmin")
			id, _ := strconv.Atoi(idStr)
			u, ok := users[id]
			if r.Method == http.MethodPut {
				if !ok {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				var body struct {
					SysadminFlag bool `json:"sysadmin_flag"`
				}
				_ = json.NewDecoder(r.Body).Decode(&body)
				u.sysadmin = body.SysadminFlag
				w.WriteHeader(http.StatusOK)
				return
			}
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		// plain /api/v2.0/users/<id> — GET, PUT (profile update), DELETE
		id, _ := strconv.Atoi(path)
		u, ok := users[id]
		switch r.Method {
		case http.MethodGet:
			if !ok {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"errors":[{"code":"NOT_FOUND","message":"user not found"}]}`))
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(userJSON(u)))
		case http.MethodPut:
			// SDK UpdateUserProfile sends PUT /users/<id> with the profile body.
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			var body struct {
				Email string `json:"email"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body.Email != "" {
				u.email = body.Email
			}
			w.WriteHeader(http.StatusOK)
		case http.MethodDelete:
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			delete(users, id)
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	return httptest.NewServer(mux)
}

func TestUserClient_RealCRUD(t *testing.T) {
	srv := fakeHarborUsers(t)
	defer srv.Close()
	c := newTestClient(t, srv.URL)
	ctx := context.Background()

	// Not found before creation -> (nil, nil).
	if st, err := c.GetUser(ctx, "alice"); err != nil || st != nil {
		t.Fatalf("expected (nil,nil) before create, got st=%v err=%v", st, err)
	}

	// Create -> returns the authoritative username and email from Harbor.
	st, err := c.CreateUser(ctx, &UserSpec{
		Username: "alice",
		Email:    "alice@example.com",
		Password: "S3cr3tP@ss",
	})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if st.Username != "alice" {
		t.Errorf("expected username alice, got %q (stub would return hardcoded)", st.Username)
	}
	if st.Email != "alice@example.com" {
		t.Errorf("expected email alice@example.com, got %q", st.Email)
	}
	if st.AdminFlag {
		t.Errorf("expected AdminFlag=false for non-admin user, got true")
	}

	// Get -> exists now, resolved via ListUsers+match.
	got, err := c.GetUser(ctx, "alice")
	if err != nil || got == nil {
		t.Fatalf("GetUser after create: st=%v err=%v", got, err)
	}
	if got.Username != "alice" {
		t.Errorf("expected username alice, got %q", got.Username)
	}

	// Update email.
	if _, err := c.UpdateUser(ctx, "alice", &UserSpec{
		Username: "alice",
		Email:    "alice-new@example.com",
	}); err != nil {
		t.Fatalf("UpdateUser: %v", err)
	}
	got2, err := c.GetUser(ctx, "alice")
	if err != nil || got2 == nil {
		t.Fatalf("GetUser after update: st=%v err=%v", got2, err)
	}
	if got2.Email != "alice-new@example.com" {
		t.Errorf("expected updated email alice-new@example.com, got %q", got2.Email)
	}

	// Create sysadmin user.
	stAdmin, err := c.CreateUser(ctx, &UserSpec{
		Username:  "bob-admin",
		Email:     "bob@example.com",
		Password:  "S3cr3tP@ss",
		AdminFlag: true,
	})
	if err != nil {
		t.Fatalf("CreateUser sysadmin: %v", err)
	}
	if !stAdmin.AdminFlag {
		t.Errorf("expected AdminFlag=true for sysadmin user, got false")
	}

	// Delete alice -> gone.
	if err := c.DeleteUser(ctx, "alice"); err != nil {
		t.Fatalf("DeleteUser: %v", err)
	}
	if st, err := c.GetUser(ctx, "alice"); err != nil || st != nil {
		t.Fatalf("expected (nil,nil) after delete, got st=%v err=%v", st, err)
	}

	// Delete is idempotent.
	if err := c.DeleteUser(ctx, "alice"); err != nil {
		t.Fatalf("idempotent DeleteUser: %v", err)
	}
}
