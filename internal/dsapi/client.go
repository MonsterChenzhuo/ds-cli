package dsapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type Client struct {
	profile Profile
	http    *http.Client
}

type Response struct {
	HTTPStatus int             `json:"http_status"`
	Body       json.RawMessage `json:"body"`
}

func NewClient(profile Profile) *Client {
	timeout := profile.Timeout
	if timeout == 0 {
		timeout = defaultTimeout
	}
	return &Client{
		profile: profile,
		http:    &http.Client{Timeout: timeout},
	}
}

func NormalizeBaseURL(raw string) string {
	s := strings.TrimRight(strings.TrimSpace(raw), "/")
	s = strings.TrimSuffix(s, "/ui")
	return strings.TrimRight(s, "/")
}

func (c *Client) JSON(ctx context.Context, method, path string, body any) (*Response, error) {
	if err := c.ensureAuth(ctx); err != nil {
		return nil, err
	}
	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		r = bytes.NewReader(b)
	}
	return c.do(ctx, method, path, r, "application/json")
}

func (c *Client) Form(ctx context.Context, method, path string, values url.Values) (*Response, error) {
	if err := c.ensureAuth(ctx); err != nil {
		return nil, err
	}
	if values == nil {
		values = url.Values{}
	}
	if method == http.MethodGet {
		if encoded := values.Encode(); encoded != "" {
			if strings.Contains(path, "?") {
				path += "&" + encoded
			} else {
				path += "?" + encoded
			}
		}
		return c.do(ctx, method, path, nil, "")
	}
	return c.do(ctx, method, path, strings.NewReader(values.Encode()), "application/x-www-form-urlencoded")
}

func (c *Client) ensureAuth(ctx context.Context) error {
	if c.profile.Token != "" || c.profile.SessionID != "" {
		return nil
	}
	if c.profile.Username == "" || c.profile.Password == "" {
		return errors.New("authentication is required: configure token, session_id, or username/password")
	}
	values := url.Values{}
	values.Set("userName", c.profile.Username)
	values.Set("userPassword", c.profile.Password)
	resp, err := c.doWithoutAuth(ctx, http.MethodPost, "/login", strings.NewReader(values.Encode()), "application/x-www-form-urlencoded")
	if err != nil {
		return err
	}
	var decoded struct {
		Code int `json:"code"`
		Data struct {
			SessionID string `json:"sessionId"`
		} `json:"data"`
		Msg string `json:"msg"`
	}
	if err := json.Unmarshal(resp.Body, &decoded); err != nil {
		return err
	}
	if decoded.Code != 0 {
		return fmt.Errorf("login failed: code=%d msg=%s", decoded.Code, decoded.Msg)
	}
	if decoded.Data.SessionID == "" {
		return errors.New("login failed: response did not contain data.sessionId")
	}
	c.profile.SessionID = decoded.Data.SessionID
	return nil
}

func (c *Client) do(ctx context.Context, method, path string, body io.Reader, contentType string) (*Response, error) {
	req, err := c.newRequest(ctx, method, path, body, contentType)
	if err != nil {
		return nil, err
	}
	if c.profile.Token != "" {
		req.Header.Set("token", c.profile.Token)
	} else if c.profile.SessionID != "" {
		req.Header.Set("sessionId", c.profile.SessionID)
	}
	return c.roundTrip(req)
}

func (c *Client) doWithoutAuth(ctx context.Context, method, path string, body io.Reader, contentType string) (*Response, error) {
	req, err := c.newRequest(ctx, method, path, body, contentType)
	if err != nil {
		return nil, err
	}
	return c.roundTrip(req)
}

func (c *Client) newRequest(ctx context.Context, method, path string, body io.Reader, contentType string) (*http.Request, error) {
	u := c.profile.APIURL + "/" + strings.TrimLeft(path, "/")
	req, err := http.NewRequestWithContext(ctx, method, u, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	return req, nil
}

func (c *Client) roundTrip(req *http.Request) (*Response, error) {
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	out := &Response{HTTPStatus: resp.StatusCode, Body: json.RawMessage(b)}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return out, fmt.Errorf("dolphinscheduler api returned HTTP %d: %s", resp.StatusCode, trimBody(b))
	}
	if err := checkResultCode(b); err != nil {
		return out, err
	}
	return out, nil
}

func checkResultCode(b []byte) error {
	var decoded struct {
		Code *int   `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := json.Unmarshal(b, &decoded); err != nil {
		return err
	}
	if decoded.Code != nil && *decoded.Code != 0 {
		return fmt.Errorf("dolphinscheduler api returned code=%d msg=%s", *decoded.Code, decoded.Msg)
	}
	return nil
}

func trimBody(b []byte) string {
	s := string(b)
	if len(s) > 500 {
		return s[:500] + "..."
	}
	return s
}
