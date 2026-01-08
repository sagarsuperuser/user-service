package web

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	apiv1 "github.com/sagarsuperuser/userprofile/server/routes/api/v1"
	"github.com/sagarsuperuser/userprofile/server/settings"
)

type APIClient struct {
	baseURL string
	client  *http.Client
}

func NewAPIClient(s *settings.Settings) *APIClient {
	return &APIClient{
		baseURL: fmt.Sprintf("http://127.0.0.1:%d", s.Port),
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (c *APIClient) Request(ctx context.Context, incoming *http.Request, method, endpoint string, payload any) (*http.Response, error) {
	var body io.Reader
	if payload != nil {
		buf := new(bytes.Buffer)
		if err := json.NewEncoder(buf).Encode(payload); err != nil {
			return nil, err
		}
		body = buf
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+endpoint, body)
	if err != nil {
		return nil, err
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if incoming != nil {
		// forward full Cookie header
		if ck := incoming.Header.Get("Cookie"); ck != "" {
			req.Header.Set("Cookie", ck)
		}
	}

	return c.client.Do(req)
}

func (c *APIClient) FetchCurrentUser(ctx context.Context, r *http.Request) (*apiv1.UserResp, int, error) {
	resp, err := c.Request(ctx, r, http.MethodGet, "/user/me", nil)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		io.Copy(io.Discard, resp.Body)
		return nil, resp.StatusCode, nil
	}

	var user apiv1.UserResp
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, 0, err
	}
	return &user, http.StatusOK, nil
}
