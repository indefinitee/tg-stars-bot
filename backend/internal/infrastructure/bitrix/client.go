package bitrix

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"tg-stars-bot/internal/domain"
)

const (
	defaultTimeout = 30 * time.Second
)

// Client implements domain.BitrixClient for Bitrix24 API
type Client struct {
	baseURL string
	webhook string
	client  *http.Client
}

// Config holds Bitrix24 client configuration
type Config struct {
	BaseURL string
	Webhook string
}

// NewClient creates a new Bitrix24 client
func NewClient(cfg Config) *Client {
	return &Client{
		baseURL: cfg.BaseURL,
		webhook: cfg.Webhook,
		client: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

// Employee represents a Bitrix24 employee
type Employee struct {
	ID            int    `json:"ID"`
	UF_PHONE      string `json:"UF_PHONE,omitempty"`
	EMAIL         string `json:"EMAIL,omitempty"`
	NAME          string `json:"NAME"`
	LAST_NAME     string `json:"LAST_NAME,omitempty"`
	WORK_POSITION string `json:"WORK_POSITION,omitempty"`
}

// BitrixResponse represents a generic Bitrix24 API response
type BitrixResponse struct {
	Result json.RawMessage `json:"result"`
	Total  int             `json:"total"`
	Next   int             `json:"next"`
	Error  string          `json:"error"`
	Time   float64         `json:"time"`
}

// GetEmployees retrieves all employees from Bitrix24
func (c *Client) GetEmployees(ctx context.Context) ([]*domain.User, error) {
	url := fmt.Sprintf("%s%s", c.baseURL, c.webhook)

	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return nil, err
	}

	// Add Bitrix24 API method and parameters
	q := req.URL.Query()
	q.Add("method", "user.get")
	q.Add("fields[ACTIVE]", "true")
	q.Add("fields[UF_DEPARTMENT]", "1") // Only employees with department
	req.URL.RawQuery = q.Encode()

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result BitrixResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if result.Error != "" {
		return nil, fmt.Errorf("bitrix error: %s", result.Error)
	}

	var employees []Employee
	if err := json.Unmarshal(result.Result, &employees); err != nil {
		return nil, err
	}

	users := make([]*domain.User, 0, len(employees))
	for _, emp := range employees {
		user := &domain.User{
			BitrixID:  emp.ID,
			Username:  emp.NAME,
			FirstName: emp.NAME,
			LastName:  emp.LAST_NAME,
			Email:     emp.EMAIL,
			IsActive:  true,
		}
		users = append(users, user)
	}

	return users, nil
}

// SyncUsers synchronizes users from Bitrix24 to the local database
func (c *Client) SyncUsers(ctx context.Context) error {
	// This method is just a marker - actual sync is done in the use case
	// by iterating over users and calling userRepo.UpsertFromBitrix
	return nil
}
