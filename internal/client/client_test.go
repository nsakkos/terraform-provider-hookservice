package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *Client) {
	t.Helper()
	srv := httptest.NewServer(handler)
	c := NewClient(srv.URL, "test-token")
	return srv, c
}

func jsonResponse(t *testing.T, w http.ResponseWriter, statusCode int, data interface{}) {
	t.Helper()
	resp := APIResponse{}
	b, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}
	resp.Data = b
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		t.Fatal(err)
	}
}

func TestNewClient(t *testing.T) {
	c := NewClient("http://example.com/", "my-token")
	if c.Host != "http://example.com" {
		t.Errorf("expected trailing slash stripped, got %s", c.Host)
	}
	if c.Token != "my-token" {
		t.Errorf("expected token my-token, got %s", c.Token)
	}
}

func TestListGroups(t *testing.T) {
	groups := []Group{
		{ID: "1", Name: "group1", Description: "desc1", Type: "local"},
		{ID: "2", Name: "group2", Description: "desc2", Type: "local"},
	}

	srv, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v0/authz/groups" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("missing or wrong auth header: %s", r.Header.Get("Authorization"))
		}
		jsonResponse(t, w, http.StatusOK, groups)
	})
	defer srv.Close()

	result, err := c.ListGroups()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(result))
	}
	if result[0].Name != "group1" {
		t.Errorf("expected group1, got %s", result[0].Name)
	}
}

func TestCreateGroup(t *testing.T) {
	srv, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v0/authz/groups" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected JSON content type, got %s", r.Header.Get("Content-Type"))
		}

		var req GroupCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		if req.Name != "test-group" {
			t.Errorf("expected name test-group, got %s", req.Name)
		}
		if req.Description != "test desc" {
			t.Errorf("expected description 'test desc', got %s", req.Description)
		}
		if req.Type != "local" {
			t.Errorf("expected type local, got %s", req.Type)
		}

		groups := []Group{{ID: "abc-123", Name: req.Name, Description: req.Description, Type: req.Type}}
		jsonResponse(t, w, http.StatusOK, groups)
	})
	defer srv.Close()

	group, err := c.CreateGroup("test-group", "test desc", "local")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if group.ID != "abc-123" {
		t.Errorf("expected ID abc-123, got %s", group.ID)
	}
	if group.Name != "test-group" {
		t.Errorf("expected name test-group, got %s", group.Name)
	}
}

func TestGetGroup(t *testing.T) {
	t.Run("found as array", func(t *testing.T) {
		srv, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/v0/authz/groups/abc-123" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			groups := []Group{{ID: "abc-123", Name: "g1", Description: "d1", Type: "local"}}
			jsonResponse(t, w, http.StatusOK, groups)
		})
		defer srv.Close()

		group, err := c.GetGroup("abc-123")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if group == nil {
			t.Fatal("expected group, got nil")
		}
		if group.ID != "abc-123" {
			t.Errorf("expected ID abc-123, got %s", group.ID)
		}
	})

	t.Run("found as object", func(t *testing.T) {
		srv, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			group := Group{ID: "abc-123", Name: "g1", Description: "d1", Type: "local"}
			jsonResponse(t, w, http.StatusOK, group)
		})
		defer srv.Close()

		group, err := c.GetGroup("abc-123")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if group == nil {
			t.Fatal("expected group, got nil")
		}
	})

	t.Run("not found", func(t *testing.T) {
		srv, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})
		defer srv.Close()

		group, err := c.GetGroup("missing")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if group != nil {
			t.Errorf("expected nil, got %+v", group)
		}
	})
}

func TestDeleteGroup(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		srv, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodDelete {
				t.Errorf("expected DELETE, got %s", r.Method)
			}
			if r.URL.Path != "/api/v0/authz/groups/abc-123" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			w.WriteHeader(http.StatusNoContent)
		})
		defer srv.Close()

		err := c.DeleteGroup("abc-123")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("not found is ok", func(t *testing.T) {
		srv, c := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})
		defer srv.Close()

		err := c.DeleteGroup("missing")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("server error", func(t *testing.T) {
		srv, c := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("internal error"))
		})
		defer srv.Close()

		err := c.DeleteGroup("abc-123")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestAddGroupUsers(t *testing.T) {
	srv, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v0/authz/groups/g1/users" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var emails []string
		if err := json.NewDecoder(r.Body).Decode(&emails); err != nil {
			t.Fatal(err)
		}
		if len(emails) != 2 {
			t.Errorf("expected 2 emails, got %d", len(emails))
		}
		if emails[0] != "a@b.com" || emails[1] != "c@d.com" {
			t.Errorf("unexpected emails: %v", emails)
		}

		w.WriteHeader(http.StatusOK)
	})
	defer srv.Close()

	err := c.AddGroupUsers("g1", []string{"a@b.com", "c@d.com"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetGroupUsers(t *testing.T) {
	t.Run("string array", func(t *testing.T) {
		srv, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/v0/authz/groups/g1/users" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			users := []string{"a@b.com", "c@d.com"}
			jsonResponse(t, w, http.StatusOK, users)
		})
		defer srv.Close()

		users, err := c.GetGroupUsers("g1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(users) != 2 {
			t.Fatalf("expected 2 users, got %d", len(users))
		}
	})

	t.Run("object array with email", func(t *testing.T) {
		srv, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			users := []map[string]interface{}{
				{"email": "a@b.com", "name": "A"},
				{"email": "c@d.com", "name": "C"},
			}
			jsonResponse(t, w, http.StatusOK, users)
		})
		defer srv.Close()

		users, err := c.GetGroupUsers("g1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(users) != 2 {
			t.Fatalf("expected 2 users, got %d", len(users))
		}
		if users[0] != "a@b.com" {
			t.Errorf("expected a@b.com, got %s", users[0])
		}
	})
}

func TestRemoveGroupUser(t *testing.T) {
	srv, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/api/v0/authz/groups/g1/users/a@b.com" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	defer srv.Close()

	err := c.RemoveGroupUser("g1", "a@b.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAddGroupApp(t *testing.T) {
	srv, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v0/authz/groups/g1/apps" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var req AppAccessRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		if req.ClientID != "client-abc" {
			t.Errorf("expected client_id client-abc, got %s", req.ClientID)
		}

		w.WriteHeader(http.StatusOK)
	})
	defer srv.Close()

	err := c.AddGroupApp("g1", "client-abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetGroupApps(t *testing.T) {
	t.Run("string array", func(t *testing.T) {
		srv, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/v0/authz/groups/g1/apps" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			apps := []string{"client-1", "client-2"}
			jsonResponse(t, w, http.StatusOK, apps)
		})
		defer srv.Close()

		apps, err := c.GetGroupApps("g1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(apps) != 2 {
			t.Fatalf("expected 2 apps, got %d", len(apps))
		}
	})

	t.Run("object array with client_id", func(t *testing.T) {
		srv, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			apps := []map[string]interface{}{
				{"client_id": "client-1", "name": "App1"},
				{"client_id": "client-2", "name": "App2"},
			}
			jsonResponse(t, w, http.StatusOK, apps)
		})
		defer srv.Close()

		apps, err := c.GetGroupApps("g1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(apps) != 2 {
			t.Fatalf("expected 2 apps, got %d", len(apps))
		}
		if apps[0] != "client-1" {
			t.Errorf("expected client-1, got %s", apps[0])
		}
	})
}

func TestRemoveGroupApp(t *testing.T) {
	srv, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/api/v0/authz/groups/g1/apps/client-abc" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	defer srv.Close()

	err := c.RemoveGroupApp("g1", "client-abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListGroups_Error(t *testing.T) {
	srv, c := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	})
	defer srv.Close()

	_, err := c.ListGroups()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCreateGroup_Error(t *testing.T) {
	srv, c := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("bad request"))
	})
	defer srv.Close()

	_, err := c.CreateGroup("x", "y", "local")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCreateGroup_EmptyResponse(t *testing.T) {
	srv, c := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		jsonResponse(t, w, http.StatusOK, []Group{})
	})
	defer srv.Close()

	_, err := c.CreateGroup("x", "y", "local")
	if err == nil {
		t.Fatal("expected error for empty response, got nil")
	}
}

func TestAddGroupUsers_Error(t *testing.T) {
	srv, c := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("forbidden"))
	})
	defer srv.Close()

	err := c.AddGroupUsers("g1", []string{"a@b.com"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestAddGroupApp_Error(t *testing.T) {
	srv, c := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("forbidden"))
	})
	defer srv.Close()

	err := c.AddGroupApp("g1", "client-abc")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestNoAuthHeader_WhenTokenEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			t.Errorf("expected no auth header, got %s", r.Header.Get("Authorization"))
		}
		jsonResponse(t, w, http.StatusOK, []Group{})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "")
	_, err := c.ListGroups()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
