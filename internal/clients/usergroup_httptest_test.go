/*
Proof test for the real UserGroup client implementation.

Runs the actual HarborClient.{Create,List,Get,Update,Delete}UserGroup methods
against a stateful in-memory fake of Harbor's /api/v2.0 usergroup API
(httptest). This exercises the real goharbor request/response path — it does
NOT hit a live Harbor and uses no real credentials. It demonstrates the methods
are genuinely implemented (issue the right HTTP verbs/paths, parse the real
group ID from the Location header, and map 404 -> not-found) rather than
returning hardcoded stubs.
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

func fakeHarborUserGroups(t *testing.T) *httptest.Server {
	t.Helper()
	var mu sync.Mutex
	type group struct {
		id          int
		groupName   string
		groupType   int64
		ldapGroupDn string
	}
	groups := map[int]*group{}
	nextID := 9

	groupJSON := func(g *group) string {
		return fmt.Sprintf(`{"id":%d,"group_name":%q,"group_type":%d,"ldap_group_dn":%q}`,
			g.id, g.groupName, g.groupType, g.ldapGroupDn)
	}

	mux := http.NewServeMux()

	// Collection: GET /api/v2.0/usergroups, POST /api/v2.0/usergroups
	mux.HandleFunc("/api/v2.0/usergroups", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		switch r.Method {
		case http.MethodPost:
			var body struct {
				GroupName   string `json:"group_name"`
				GroupType   int64  `json:"group_type"`
				LdapGroupDn string `json:"ldap_group_dn"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			nextID++
			groups[nextID] = &group{
				id:          nextID,
				groupName:   body.GroupName,
				groupType:   body.GroupType,
				ldapGroupDn: body.LdapGroupDn,
			}
			w.Header().Set("Location", "/api/v2.0/usergroups/"+strconv.Itoa(nextID))
			w.WriteHeader(http.StatusCreated)
		case http.MethodGet:
			items := make([]string, 0, len(groups))
			for _, g := range groups {
				items = append(items, groupJSON(g))
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte("[" + strings.Join(items, ",") + "]"))
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	// Item: GET/PUT/DELETE /api/v2.0/usergroups/{id}
	mux.HandleFunc("/api/v2.0/usergroups/", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		idStr := strings.TrimPrefix(r.URL.Path, "/api/v2.0/usergroups/")
		id, _ := strconv.Atoi(idStr)
		g, ok := groups[id]
		switch r.Method {
		case http.MethodGet:
			if !ok {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"errors":[{"code":"NOT_FOUND","message":"user group not found"}]}`))
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(groupJSON(g)))
		case http.MethodPut:
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			var body struct {
				GroupName   string `json:"group_name"`
				GroupType   int64  `json:"group_type"`
				LdapGroupDn string `json:"ldap_group_dn"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			g.groupName = body.GroupName
			g.groupType = body.GroupType
			g.ldapGroupDn = body.LdapGroupDn
			w.WriteHeader(http.StatusOK)
		case http.MethodDelete:
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			delete(groups, id)
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	return httptest.NewServer(mux)
}

func TestUserGroupClient_RealCRUD(t *testing.T) {
	srv := fakeHarborUserGroups(t)
	defer srv.Close()
	c := newTestClient(t, srv.URL)
	ctx := context.Background()

	// GetUserGroup for non-existent ID -> (nil, nil).
	if st, err := c.GetUserGroup(ctx, 999); err != nil || st != nil {
		t.Fatalf("expected (nil,nil) for missing group, got st=%v err=%v", st, err)
	}

	// ListUserGroups returns empty list before any groups exist.
	all, err := c.ListUserGroups(ctx)
	if err != nil {
		t.Fatalf("ListUserGroups (empty): %v", err)
	}
	if len(all) != 0 {
		t.Errorf("expected empty list before create, got %d items", len(all))
	}

	// Create OIDC group -> returns the authoritative ID parsed from Location header.
	st, err := c.CreateUserGroup(ctx, &UserGroupSpec{
		GroupName: "devs",
		GroupType: 3, // OIDC
	})
	if err != nil {
		t.Fatalf("CreateUserGroup: %v", err)
	}
	if st.ID != 10 {
		t.Errorf("expected real group ID 10 from API, got %d (stub would be 1)", st.ID)
	}
	if st.GroupName != "devs" {
		t.Errorf("expected group name devs, got %q", st.GroupName)
	}
	if st.GroupType != 3 {
		t.Errorf("expected group type 3 (OIDC), got %d", st.GroupType)
	}

	// GetUserGroup by ID -> exists now.
	got, err := c.GetUserGroup(ctx, st.ID)
	if err != nil || got == nil {
		t.Fatalf("GetUserGroup after create: st=%v err=%v", got, err)
	}
	if got.GroupName != "devs" {
		t.Errorf("expected name devs, got %q", got.GroupName)
	}

	// ListUserGroups -> finds the new group.
	all, err = c.ListUserGroups(ctx)
	if err != nil {
		t.Fatalf("ListUserGroups: %v", err)
	}
	if len(all) != 1 || all[0].GroupName != "devs" {
		t.Errorf("expected one group named devs, got %+v", all)
	}

	// Update group name and type.
	updated, err := c.UpdateUserGroup(ctx, st.ID, &UserGroupSpec{
		GroupName: "admins",
		GroupType: 2, // HTTP
	})
	if err != nil {
		t.Fatalf("UpdateUserGroup: %v", err)
	}
	if updated.GroupName != "admins" {
		t.Errorf("expected name admins after update, got %q", updated.GroupName)
	}
	if updated.GroupType != 2 {
		t.Errorf("expected type 2 (HTTP) after update, got %d", updated.GroupType)
	}

	// Get after update confirms round-trip.
	got2, err := c.GetUserGroup(ctx, st.ID)
	if err != nil || got2 == nil {
		t.Fatalf("GetUserGroup after update: %v", err)
	}
	if got2.GroupName != "admins" {
		t.Errorf("expected admins after update, got %q", got2.GroupName)
	}

	// Delete -> gone.
	if err := c.DeleteUserGroup(ctx, st.ID); err != nil {
		t.Fatalf("DeleteUserGroup: %v", err)
	}
	if st2, err := c.GetUserGroup(ctx, st.ID); err != nil || st2 != nil {
		t.Fatalf("expected (nil,nil) after delete, got st=%v err=%v", st2, err)
	}

	// Delete is idempotent on 404.
	if err := c.DeleteUserGroup(ctx, st.ID); err != nil {
		t.Fatalf("idempotent DeleteUserGroup: %v", err)
	}
}

func TestUserGroupClient_LdapGroup(t *testing.T) {
	srv := fakeHarborUserGroups(t)
	defer srv.Close()
	c := newTestClient(t, srv.URL)
	ctx := context.Background()

	dn := "cn=devs,dc=example,dc=com"
	st, err := c.CreateUserGroup(ctx, &UserGroupSpec{
		GroupName:   "ldap-devs",
		GroupType:   1, // LDAP
		LdapGroupDn: &dn,
	})
	if err != nil {
		t.Fatalf("CreateUserGroup (LDAP): %v", err)
	}
	if st.LdapGroupDn != dn {
		t.Errorf("expected ldap_group_dn %q, got %q", dn, st.LdapGroupDn)
	}
}
