package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const apiBasePath = "/api/v0"

// Client is an HTTP client for the Hook Service API.
type Client struct {
	Host       string
	Token      string
	HTTPClient *http.Client
}

// NewClient creates a new Hook Service API client.
func NewClient(host, token string) *Client {
	host = strings.TrimRight(host, "/")
	token = strings.TrimSpace(token)
	return &Client{
		Host:       host,
		Token:      token,
		HTTPClient: &http.Client{},
	}
}

// NewClientWithHTTPClient creates a new Hook Service API client with a custom HTTP client.
// Used with OAuth2 client credentials flow where the HTTP client handles token management.
func NewClientWithHTTPClient(host string, httpClient *http.Client) *Client {
	host = strings.TrimRight(host, "/")
	return &Client{
		Host:       host,
		HTTPClient: httpClient,
	}
}

// APIResponse represents a standard API response wrapper.
type APIResponse struct {
	Data    json.RawMessage `json:"data"`
	Message string          `json:"message,omitempty"`
	Status  int             `json:"status,omitempty"`
}

// Group represents a group in the Hook Service.
type Group struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
}

// GroupCreateRequest represents the request body for creating a group.
type GroupCreateRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
}

// AppAccessRequest represents the request body for granting app access.
type AppAccessRequest struct {
	ClientID string `json:"client_id"`
}

func (c *Client) doRequest(method, path string, body interface{}) ([]byte, int, error) {
	url := fmt.Sprintf("%s%s%s", c.Host, apiBasePath, path)

	var reqBody io.Reader
	if body != nil {
		jsonBytes, err := json.Marshal(body)
		if err != nil {
			return nil, 0, fmt.Errorf("error marshalling request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonBytes)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, 0, fmt.Errorf("error creating request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("error reading response: %w", err)
	}

	return respBody, resp.StatusCode, nil
}

// ListGroups retrieves all groups.
func (c *Client) ListGroups() ([]Group, error) {
	respBody, statusCode, err := c.doRequest(http.MethodGet, "/authz/groups", nil)
	if err != nil {
		return nil, err
	}

	if statusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d: %s", statusCode, string(respBody))
	}

	var apiResp APIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	var groups []Group
	if err := json.Unmarshal(apiResp.Data, &groups); err != nil {
		return nil, fmt.Errorf("error decoding groups: %w", err)
	}

	return groups, nil
}

// CreateGroup creates a new group.
func (c *Client) CreateGroup(name, description, groupType string) (*Group, error) {
	reqBody := GroupCreateRequest{
		Name:        name,
		Description: description,
		Type:        groupType,
	}

	respBody, statusCode, err := c.doRequest(http.MethodPost, "/authz/groups", reqBody)
	if err != nil {
		return nil, err
	}

	if statusCode != http.StatusOK && statusCode != http.StatusCreated {
		return nil, fmt.Errorf("unexpected status code %d: %s", statusCode, string(respBody))
	}

	var apiResp APIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	var groups []Group
	if err := json.Unmarshal(apiResp.Data, &groups); err != nil {
		return nil, fmt.Errorf("error decoding group: %w", err)
	}

	if len(groups) == 0 {
		return nil, fmt.Errorf("no group returned in response")
	}

	return &groups[0], nil
}

// GetGroup retrieves a single group by ID.
func (c *Client) GetGroup(groupID string) (*Group, error) {
	respBody, statusCode, err := c.doRequest(http.MethodGet, fmt.Sprintf("/authz/groups/%s", groupID), nil)
	if err != nil {
		return nil, err
	}

	if statusCode == http.StatusNotFound {
		return nil, nil
	}

	if statusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d: %s", statusCode, string(respBody))
	}

	var apiResp APIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	// The response may return a single group or an array with one group.
	// Try array first (consistent with create response), then single object.
	var groups []Group
	if err := json.Unmarshal(apiResp.Data, &groups); err == nil && len(groups) > 0 {
		return &groups[0], nil
	}

	var group Group
	if err := json.Unmarshal(apiResp.Data, &group); err != nil {
		return nil, fmt.Errorf("error decoding group: %w", err)
	}

	return &group, nil
}

// DeleteGroup deletes a group by ID.
func (c *Client) DeleteGroup(groupID string) error {
	respBody, statusCode, err := c.doRequest(http.MethodDelete, fmt.Sprintf("/authz/groups/%s", groupID), nil)
	if err != nil {
		return err
	}

	if statusCode != http.StatusOK && statusCode != http.StatusNoContent && statusCode != http.StatusNotFound {
		return fmt.Errorf("unexpected status code %d: %s", statusCode, string(respBody))
	}

	return nil
}

// GetGroupUsers retrieves all users in a group.
func (c *Client) GetGroupUsers(groupID string) ([]string, error) {
	respBody, statusCode, err := c.doRequest(http.MethodGet, fmt.Sprintf("/authz/groups/%s/users", groupID), nil)
	if err != nil {
		return nil, err
	}

	if statusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d: %s", statusCode, string(respBody))
	}

	var apiResp APIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	var users []string
	if err := json.Unmarshal(apiResp.Data, &users); err != nil {
		// The API might return user objects instead of plain strings.
		// Try to extract email fields from objects.
		users = nil
		var userObjects []map[string]interface{}
		if err2 := json.Unmarshal(apiResp.Data, &userObjects); err2 != nil {
			return nil, fmt.Errorf("error decoding users: %w (also tried objects: %w)", err, err2)
		}
		for _, u := range userObjects {
			if email, ok := u["email"].(string); ok {
				users = append(users, email)
			} else if id, ok := u["id"].(string); ok {
				users = append(users, id)
			}
		}
	}

	return users, nil
}

// AddGroupUsers adds users to a group.
func (c *Client) AddGroupUsers(groupID string, emails []string) error {
	respBody, statusCode, err := c.doRequest(http.MethodPost, fmt.Sprintf("/authz/groups/%s/users", groupID), emails)
	if err != nil {
		return err
	}

	if statusCode != http.StatusOK && statusCode != http.StatusCreated {
		return fmt.Errorf("unexpected status code %d: %s", statusCode, string(respBody))
	}

	return nil
}

// RemoveGroupUser removes a user from a group.
func (c *Client) RemoveGroupUser(groupID, email string) error {
	respBody, statusCode, err := c.doRequest(http.MethodDelete, fmt.Sprintf("/authz/groups/%s/users/%s", groupID, email), nil)
	if err != nil {
		return err
	}

	if statusCode != http.StatusOK && statusCode != http.StatusNoContent && statusCode != http.StatusNotFound {
		return fmt.Errorf("unexpected status code %d: %s", statusCode, string(respBody))
	}

	return nil
}

// GetGroupApps retrieves all apps for a group.
func (c *Client) GetGroupApps(groupID string) ([]string, error) {
	respBody, statusCode, err := c.doRequest(http.MethodGet, fmt.Sprintf("/authz/groups/%s/apps", groupID), nil)
	if err != nil {
		return nil, err
	}

	if statusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d: %s", statusCode, string(respBody))
	}

	var apiResp APIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	var clientIDs []string
	if err := json.Unmarshal(apiResp.Data, &clientIDs); err != nil {
		// The API might return app objects instead of plain strings.
		clientIDs = nil
		var appObjects []map[string]interface{}
		if err2 := json.Unmarshal(apiResp.Data, &appObjects); err2 != nil {
			return nil, fmt.Errorf("error decoding apps: %w (also tried objects: %w)", err, err2)
		}
		for _, a := range appObjects {
			if cid, ok := a["client_id"].(string); ok {
				clientIDs = append(clientIDs, cid)
			} else if id, ok := a["id"].(string); ok {
				clientIDs = append(clientIDs, id)
			}
		}
	}

	return clientIDs, nil
}

// AddGroupApp grants a group access to an application.
func (c *Client) AddGroupApp(groupID, clientID string) error {
	reqBody := AppAccessRequest{ClientID: clientID}

	respBody, statusCode, err := c.doRequest(http.MethodPost, fmt.Sprintf("/authz/groups/%s/apps", groupID), reqBody)
	if err != nil {
		return err
	}

	if statusCode != http.StatusOK && statusCode != http.StatusCreated {
		return fmt.Errorf("unexpected status code %d: %s", statusCode, string(respBody))
	}

	return nil
}

// RemoveGroupApp removes an application from a group.
func (c *Client) RemoveGroupApp(groupID, clientID string) error {
	respBody, statusCode, err := c.doRequest(http.MethodDelete, fmt.Sprintf("/authz/groups/%s/apps/%s", groupID, clientID), nil)
	if err != nil {
		return err
	}

	if statusCode != http.StatusOK && statusCode != http.StatusNoContent && statusCode != http.StatusNotFound {
		return fmt.Errorf("unexpected status code %d: %s", statusCode, string(respBody))
	}

	return nil
}
