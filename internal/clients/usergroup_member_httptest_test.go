/*
Proof tests for the real id-keyed UserMember and GroupMember client paths.

Each runs the actual HarborClient methods against a stateful in-memory fake of
Harbor's /api/v2.0 project-member API (httptest), exercising the real goharbor
request/response path. They assert the create body carries member_user.username
(user) vs member_group.group_name+group_type (group), that the real member id is
parsed from the Location header, and that get-by-id / delete-by-id round-trip.
No live Harbor, no real credentials.
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

// fakeHarborProjectMembers returns an httptest server modelling project 1's
// members with full CRUD: POST (user or group), GET list, GET by id, PUT by id,
// DELETE by id. lastBody captures the most recent create payload for assertions.
func fakeHarborProjectMembers(t *testing.T, lastBody *map[string]interface{}) *httptest.Server {
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
			var raw map[string]interface{}
			_ = json.NewDecoder(r.Body).Decode(&raw)
			if lastBody != nil {
				*lastBody = raw
			}
			var body struct {
				RoleID     int `json:"role_id"`
				MemberUser *struct {
					Username string `json:"username"`
				} `json:"member_user"`
				MemberGroup *struct {
					GroupName string `json:"group_name"`
					GroupType int    `json:"group_type"`
				} `json:"member_group"`
			}
			b, _ := json.Marshal(raw)
			_ = json.Unmarshal(b, &body)
			nextID++
			name := ""
			entityTp := "u"
			switch {
			case body.MemberUser != nil:
				name = body.MemberUser.Username
				entityTp = "u"
			case body.MemberGroup != nil:
				name = body.MemberGroup.GroupName
				entityTp = "g"
			}
			members[nextID] = &member{id: nextID, name: name, roleID: body.RoleID, entityTp: entityTp}
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
		case http.MethodGet:
			if !ok {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"errors":[{"code":"NOT_FOUND","message":"member not found"}]}`))
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(memberJSON(m)))
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

func TestUserMemberClient_RealCRUD(t *testing.T) {
	var body map[string]interface{}
	srv := fakeHarborProjectMembers(t, &body)
	defer srv.Close()
	c := newTestClient(t, srv.URL)
	ctx := context.Background()

	// Not found before creation -> (nil, nil).
	if st, err := c.FindProjectMember(ctx, "1", "u", "alice"); err != nil || st != nil {
		t.Fatalf("expected (nil,nil) before add, got st=%v err=%v", st, err)
	}

	// Add user member -> returns the real Harbor member id (from Location).
	id, err := c.AddProjectUserMember(ctx, "1", "alice", "developer")
	if err != nil {
		t.Fatalf("AddProjectUserMember: %v", err)
	}
	if id != "1" {
		t.Errorf("expected real member id 1 from Location, got %q", id)
	}

	// Assert the create body carried member_user.username (not member_group).
	if body["member_user"] == nil {
		t.Errorf("expected member_user in create body, got %v", body)
	}
	if mu, ok := body["member_user"].(map[string]interface{}); !ok || mu["username"] != "alice" {
		t.Errorf("expected member_user.username=alice, got %v", body["member_user"])
	}
	if body["member_group"] != nil {
		t.Errorf("user create body must not carry member_group, got %v", body["member_group"])
	}

	// Get by id -> exists, role_id 2 maps to "developer", entity_type u -> user.
	st, err := c.GetProjectMemberByID(ctx, "1", id)
	if err != nil || st == nil {
		t.Fatalf("GetProjectMemberByID after add: st=%v err=%v", st, err)
	}
	if st.ID != "1" {
		t.Errorf("expected member id 1, got %q", st.ID)
	}
	if st.Role != "developer" {
		t.Errorf("expected role developer, got %q", st.Role)
	}
	if st.MemberType != "user" {
		t.Errorf("expected member type user, got %q", st.MemberType)
	}

	// Update role by id developer -> maintainer, confirm round-trip.
	if err := c.UpdateProjectMemberByID(ctx, "1", id, "maintainer"); err != nil {
		t.Fatalf("UpdateProjectMemberByID: %v", err)
	}
	st2, err := c.GetProjectMemberByID(ctx, "1", id)
	if err != nil || st2 == nil {
		t.Fatalf("GetProjectMemberByID after update: st=%v err=%v", st2, err)
	}
	if st2.Role != "maintainer" {
		t.Errorf("expected role maintainer after update, got %q", st2.Role)
	}

	// Adoption: FindProjectMember matches by entity name.
	if adopted, err := c.FindProjectMember(ctx, "1", "u", "alice"); err != nil || adopted == nil || adopted.ID != id {
		t.Fatalf("FindProjectMember should adopt alice with id %q, got %v err=%v", id, adopted, err)
	}

	// Delete by id -> gone (get-by-id now (nil,nil)), idempotent.
	if err := c.DeleteProjectMemberByID(ctx, "1", id); err != nil {
		t.Fatalf("DeleteProjectMemberByID: %v", err)
	}
	if st, err := c.GetProjectMemberByID(ctx, "1", id); err != nil || st != nil {
		t.Fatalf("expected (nil,nil) after delete, got st=%v err=%v", st, err)
	}
	if err := c.DeleteProjectMemberByID(ctx, "1", id); err != nil {
		t.Fatalf("idempotent DeleteProjectMemberByID: %v", err)
	}

	// Unknown role -> validation error, no API call.
	if _, err := c.AddProjectUserMember(ctx, "1", "bob", "wizard"); err == nil {
		t.Errorf("expected error for unknown role")
	}
}

func TestGroupMemberClient_RealCRUD(t *testing.T) {
	var body map[string]interface{}
	srv := fakeHarborProjectMembers(t, &body)
	defer srv.Close()
	c := newTestClient(t, srv.URL)
	ctx := context.Background()

	// Not found before creation -> (nil, nil).
	if st, err := c.FindProjectMember(ctx, "1", "g", "platform-admins"); err != nil || st != nil {
		t.Fatalf("expected (nil,nil) before add, got st=%v err=%v", st, err)
	}

	// Add group member (OIDC group_type 3) -> returns the real Harbor member id.
	id, err := c.AddProjectGroupMember(ctx, "1", "platform-admins", 3, "guest")
	if err != nil {
		t.Fatalf("AddProjectGroupMember: %v", err)
	}
	if id != "1" {
		t.Errorf("expected real member id 1 from Location, got %q", id)
	}

	// Assert the create body carried member_group.group_name + group_type.
	if body["member_user"] != nil {
		t.Errorf("group create body must not carry member_user, got %v", body["member_user"])
	}
	mg, ok := body["member_group"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected member_group in create body, got %v", body)
	}
	if mg["group_name"] != "platform-admins" {
		t.Errorf("expected member_group.group_name=platform-admins, got %v", mg["group_name"])
	}
	if gt, _ := mg["group_type"].(float64); int(gt) != 3 {
		t.Errorf("expected member_group.group_type=3, got %v", mg["group_type"])
	}

	// Get by id -> exists, role_id 3 maps to "guest", entity_type g -> group.
	st, err := c.GetProjectMemberByID(ctx, "1", id)
	if err != nil || st == nil {
		t.Fatalf("GetProjectMemberByID after add: st=%v err=%v", st, err)
	}
	if st.Role != "guest" {
		t.Errorf("expected role guest, got %q", st.Role)
	}
	if st.MemberType != "group" {
		t.Errorf("expected member type group, got %q", st.MemberType)
	}

	// Update role by id guest -> developer, confirm round-trip.
	if err := c.UpdateProjectMemberByID(ctx, "1", id, "developer"); err != nil {
		t.Fatalf("UpdateProjectMemberByID: %v", err)
	}
	st2, err := c.GetProjectMemberByID(ctx, "1", id)
	if err != nil || st2 == nil {
		t.Fatalf("GetProjectMemberByID after update: st=%v err=%v", st2, err)
	}
	if st2.Role != "developer" {
		t.Errorf("expected role developer after update, got %q", st2.Role)
	}

	// Adoption by group entity name.
	if adopted, err := c.FindProjectMember(ctx, "1", "g", "platform-admins"); err != nil || adopted == nil || adopted.ID != id {
		t.Fatalf("FindProjectMember should adopt group with id %q, got %v err=%v", id, adopted, err)
	}

	// Delete by id -> gone, idempotent.
	if err := c.DeleteProjectMemberByID(ctx, "1", id); err != nil {
		t.Fatalf("DeleteProjectMemberByID: %v", err)
	}
	if st, err := c.GetProjectMemberByID(ctx, "1", id); err != nil || st != nil {
		t.Fatalf("expected (nil,nil) after delete, got st=%v err=%v", st, err)
	}
	if err := c.DeleteProjectMemberByID(ctx, "1", id); err != nil {
		t.Fatalf("idempotent DeleteProjectMemberByID: %v", err)
	}
}
