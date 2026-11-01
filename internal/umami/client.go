package umami

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

const requestTimeout = 30 * time.Second

var ErrNotFound = errors.New("umami: not found")

type Credentials struct {
	APIKey   string
	Username string
	Password string
}

type Client struct {
	baseURL     string
	credentials Credentials
	http        *http.Client
	mu          sync.Mutex
	token       string
}

type Website struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	Domain string  `json:"domain"`
	TeamID *string `json:"teamId,omitempty"`
}

type CreateWebsite struct {
	Name   string  `json:"name"`
	Domain string  `json:"domain"`
	TeamID *string `json:"teamId,omitempty"`
}

type UpdateWebsite struct {
	Name   string `json:"name"`
	Domain string `json:"domain"`
}

func NewClient(baseURL string, credentials Credentials) (*Client, error) {
	if strings.TrimSpace(baseURL) == "" {
		return nil, errors.New("umami: url is required")
	}
	if credentials.APIKey == "" && (credentials.Username == "" || credentials.Password == "") {
		return nil, errors.New("umami: set apiKey, or both username and password")
	}
	return &Client{
		baseURL:     strings.TrimRight(baseURL, "/"),
		credentials: credentials,
		http:        &http.Client{Timeout: requestTimeout},
	}, nil
}

func (c *Client) Verify(ctx context.Context) error {
	return c.authenticated(ctx, http.MethodGet, "/api/me", nil, nil)
}

func (c *Client) CreateWebsite(ctx context.Context, input CreateWebsite) (Website, error) {
	var website Website
	err := c.authenticated(ctx, http.MethodPost, "/api/websites", input, &website)
	return website, err
}

func (c *Client) GetWebsite(ctx context.Context, id string) (Website, error) {
	var website *Website
	if err := c.authenticated(ctx, http.MethodGet, websitePath(id), nil, &website); err != nil {
		return Website{}, err
	}
	if website == nil || website.ID == "" {
		return Website{}, ErrNotFound
	}
	return *website, nil
}

func (c *Client) UpdateWebsite(ctx context.Context, id string, input UpdateWebsite) (Website, error) {
	var website Website
	err := c.authenticated(ctx, http.MethodPost, websitePath(id), input, &website)
	return website, err
}

func (c *Client) DeleteWebsite(ctx context.Context, id string) error {
	err := c.authenticated(ctx, http.MethodDelete, websitePath(id), nil, nil)
	if errors.Is(err, ErrNotFound) {
		return nil
	}
	return err
}

func websitePath(id string) string {
	return "/api/websites/" + id
}

func (c *Client) authenticated(ctx context.Context, method, path string, input, output any) error {
	token, err := c.bearer(ctx)
	if err != nil {
		return err
	}
	return c.do(ctx, method, path, token, input, output)
}

func (c *Client) bearer(ctx context.Context) (string, error) {
	if c.credentials.APIKey != "" {
		return c.credentials.APIKey, nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token != "" {
		return c.token, nil
	}
	var session struct {
		Token             string `json:"token"`
		RequiresTwoFactor bool   `json:"requiresTwoFactor"`
	}
	login := map[string]string{"username": c.credentials.Username, "password": c.credentials.Password}
	if err := c.do(ctx, http.MethodPost, "/api/auth/login", "", login, &session); err != nil {
		return "", fmt.Errorf("umami: login: %w", err)
	}
	if session.RequiresTwoFactor {
		return "", errors.New("umami: the account requires two-factor authentication, use an API key")
	}
	c.token = session.Token
	return c.token, nil
}

func (c *Client) do(ctx context.Context, method, path, token string, input, output any) error {
	var body io.Reader
	if input != nil {
		encoded, err := json.Marshal(input)
		if err != nil {
			return err
		}
		body = bytes.NewReader(encoded)
	}
	request, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return err
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/json")
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	response, err := c.http.Do(request)
	if err != nil {
		return err
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode == http.StatusNotFound {
		return ErrNotFound
	}
	if response.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("umami: %s %s: HTTP %d", method, path, response.StatusCode)
	}
	content, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}
	if output == nil || len(bytes.TrimSpace(content)) == 0 {
		return nil
	}
	return json.Unmarshal(content, output)
}
