package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// mockServer simulates the Hook Service API for acceptance tests.
type mockServer struct {
	mu     sync.Mutex
	groups map[string]mockGroup
	nextID int
}

type mockGroup struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Type        string   `json:"type"`
	Users       []string `json:"-"`
	Apps        []string `json:"-"`
}

type apiResponse struct {
	Data interface{} `json:"data"`
}

func newMockServer() *mockServer {
	return &mockServer{
		groups: make(map[string]mockGroup),
		nextID: 1,
	}
}

func (m *mockServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")

	// Route: GET /api/v0/authz/groups
	if r.Method == http.MethodGet && r.URL.Path == "/api/v0/authz/groups" {
		groups := make([]mockGroup, 0, len(m.groups))
		for _, g := range m.groups {
			groups = append(groups, g)
		}
		json.NewEncoder(w).Encode(apiResponse{Data: groups})
		return
	}

	// Route: POST /api/v0/authz/groups
	if r.Method == http.MethodPost && r.URL.Path == "/api/v0/authz/groups" {
		var req struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			Type        string `json:"type"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		id := fmt.Sprintf("group-%d", m.nextID)
		m.nextID++
		g := mockGroup{ID: id, Name: req.Name, Description: req.Description, Type: req.Type}
		m.groups[id] = g
		json.NewEncoder(w).Encode(apiResponse{Data: []mockGroup{g}})
		return
	}

	// Extract group ID from paths like /api/v0/authz/groups/{id}[/...]
	var groupID, subResource, subID string
	pathParts := splitPath(r.URL.Path, "/api/v0/authz/groups/")
	if len(pathParts) >= 1 {
		groupID = pathParts[0]
	}
	if len(pathParts) >= 2 {
		subResource = pathParts[1]
	}
	if len(pathParts) >= 3 {
		subID = pathParts[2]
	}

	if groupID == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// Route: GET /api/v0/authz/groups/{id}
	if r.Method == http.MethodGet && subResource == "" {
		g, ok := m.groups[groupID]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(apiResponse{Data: []mockGroup{g}})
		return
	}

	// Route: DELETE /api/v0/authz/groups/{id}
	if r.Method == http.MethodDelete && subResource == "" {
		delete(m.groups, groupID)
		w.WriteHeader(http.StatusNoContent)
		return
	}

	g, ok := m.groups[groupID]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// Route: GET /api/v0/authz/groups/{id}/users
	if r.Method == http.MethodGet && subResource == "users" {
		json.NewEncoder(w).Encode(apiResponse{Data: g.Users})
		return
	}

	// Route: POST /api/v0/authz/groups/{id}/users
	if r.Method == http.MethodPost && subResource == "users" {
		var emails []string
		json.NewDecoder(r.Body).Decode(&emails)
		existing := make(map[string]bool, len(g.Users))
		for _, u := range g.Users {
			existing[u] = true
		}
		for _, e := range emails {
			if !existing[e] {
				g.Users = append(g.Users, e)
			}
		}
		m.groups[groupID] = g
		w.WriteHeader(http.StatusOK)
		return
	}

	// Route: DELETE /api/v0/authz/groups/{id}/users/{email}
	if r.Method == http.MethodDelete && subResource == "users" && subID != "" {
		filtered := g.Users[:0]
		for _, u := range g.Users {
			if u != subID {
				filtered = append(filtered, u)
			}
		}
		g.Users = filtered
		m.groups[groupID] = g
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Route: GET /api/v0/authz/groups/{id}/apps
	if r.Method == http.MethodGet && subResource == "apps" {
		json.NewEncoder(w).Encode(apiResponse{Data: g.Apps})
		return
	}

	// Route: POST /api/v0/authz/groups/{id}/apps
	if r.Method == http.MethodPost && subResource == "apps" {
		var req struct {
			ClientID string `json:"client_id"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		existing := make(map[string]bool, len(g.Apps))
		for _, a := range g.Apps {
			existing[a] = true
		}
		if !existing[req.ClientID] {
			g.Apps = append(g.Apps, req.ClientID)
		}
		m.groups[groupID] = g
		w.WriteHeader(http.StatusOK)
		return
	}

	// Route: DELETE /api/v0/authz/groups/{id}/apps/{client_id}
	if r.Method == http.MethodDelete && subResource == "apps" && subID != "" {
		filtered := g.Apps[:0]
		for _, a := range g.Apps {
			if a != subID {
				filtered = append(filtered, a)
			}
		}
		g.Apps = filtered
		m.groups[groupID] = g
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.WriteHeader(http.StatusNotFound)
}

func splitPath(path, prefix string) []string {
	if len(path) <= len(prefix) {
		return nil
	}
	rest := path[len(prefix):]
	var parts []string
	current := ""
	for _, c := range rest {
		if c == '/' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

func TestAccGroupResource(t *testing.T) {
	mock := newMockServer()
	srv := httptest.NewServer(mock)
	defer srv.Close()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(srv.URL) + `
resource "hookservice_group" "test" {
  name        = "test-group"
  description = "Test group"
  type        = "local"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("hookservice_group.test", "id"),
					resource.TestCheckResourceAttr("hookservice_group.test", "name", "test-group"),
					resource.TestCheckResourceAttr("hookservice_group.test", "description", "Test group"),
					resource.TestCheckResourceAttr("hookservice_group.test", "type", "local"),
				),
			},
			// Update the group name.
			{
				Config: testAccProviderConfig(srv.URL) + `
resource "hookservice_group" "test" {
  name        = "updated-group"
  description = "Updated description"
  type        = "local"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("hookservice_group.test", "id"),
					resource.TestCheckResourceAttr("hookservice_group.test", "name", "updated-group"),
					resource.TestCheckResourceAttr("hookservice_group.test", "description", "Updated description"),
				),
			},
		},
	})
}

func TestAccGroupUsersResource(t *testing.T) {
	mock := newMockServer()
	srv := httptest.NewServer(mock)
	defer srv.Close()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(srv.URL) + `
resource "hookservice_group" "test" {
  name        = "users-test"
  description = "Group for user tests"
}

resource "hookservice_group_users" "test" {
  group_id = hookservice_group.test.id
  emails   = ["alice@example.com", "bob@example.com"]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("hookservice_group_users.test", "emails.#", "2"),
				),
			},
			// Add a user.
			{
				Config: testAccProviderConfig(srv.URL) + `
resource "hookservice_group" "test" {
  name        = "users-test"
  description = "Group for user tests"
}

resource "hookservice_group_users" "test" {
  group_id = hookservice_group.test.id
  emails   = ["alice@example.com", "bob@example.com", "charlie@example.com"]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("hookservice_group_users.test", "emails.#", "3"),
				),
			},
			// Remove a user.
			{
				Config: testAccProviderConfig(srv.URL) + `
resource "hookservice_group" "test" {
  name        = "users-test"
  description = "Group for user tests"
}

resource "hookservice_group_users" "test" {
  group_id = hookservice_group.test.id
  emails   = ["alice@example.com"]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("hookservice_group_users.test", "emails.#", "1"),
				),
			},
		},
	})
}

func TestAccGroupAppResource(t *testing.T) {
	mock := newMockServer()
	srv := httptest.NewServer(mock)
	defer srv.Close()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(srv.URL) + `
resource "hookservice_group" "test" {
  name        = "app-test"
  description = "Group for app tests"
}

resource "hookservice_group_app" "test" {
  group_id  = hookservice_group.test.id
  client_id = "my-client-id"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("hookservice_group_app.test", "client_id", "my-client-id"),
				),
			},
		},
	})
}

func TestAccGroupsDataSource(t *testing.T) {
	mock := newMockServer()
	srv := httptest.NewServer(mock)
	defer srv.Close()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(srv.URL) + `
resource "hookservice_group" "a" {
  name        = "group-a"
  description = "Group A"
}

resource "hookservice_group" "b" {
  name        = "group-b"
  description = "Group B"
}

data "hookservice_groups" "all" {
  depends_on = [hookservice_group.a, hookservice_group.b]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.hookservice_groups.all", "groups.#", "2"),
				),
			},
		},
	})
}

func testAccProviderConfig(url string) string {
	return fmt.Sprintf(`
provider "hookservice" {
  host = %q
}
`, url)
}
