package amx

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client defines the interface for interacting with the AMX DVX switcher.
type Client interface {
	SwitchVideo(ctx context.Context, output int, input int) error
	Ping(ctx context.Context) error
	GetHost() string
}

// Config holds connection parameters for the AMX switcher.
type Config struct {
	Host       string
	Timeout    time.Duration
	Insecure   bool
	UserAgent  string
}

type client struct {
	httpClient *http.Client
	baseURL    string
	origin     string
	referer    string
	userAgent  string
}

const (
	defaultEndpoint  = "/web/module/DVX-Switcher-Dashboard/com.amx.dvx/hcontrol"
	dashboardPath    = "/web/module/DVX-Switcher-Dashboard/com.amx.dvx/switcher.dashboard"
	defaultUserAgent = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
)

// NewClient initializes a new AMX DVX client.
func NewClient(cfg Config) (Client, error) {
	if cfg.Host == "" {
		return nil, fmt.Errorf("amx host cannot be empty")
	}

	host := strings.TrimRight(cfg.Host, "/")
	if !strings.HasPrefix(host, "http://") && !strings.HasPrefix(host, "https://") {
		host = "http://" + host
	}

	u, err := url.Parse(host)
	if err != nil {
		return nil, fmt.Errorf("invalid amx host URL: %w", err)
	}

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	ua := cfg.UserAgent
	if ua == "" {
		ua = defaultUserAgent
	}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true, // switcher self-signed certs
		},
	}

	httpClient := &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}

	origin := fmt.Sprintf("%s://%s", u.Scheme, u.Host)
	referer := origin + dashboardPath

	return &client{
		httpClient: httpClient,
		baseURL:    host,
		origin:     origin,
		referer:    referer,
		userAgent:  ua,
	}, nil
}

func (c *client) GetHost() string {
	return c.baseURL
}

type switchPayload struct {
	Path  string `json:"path"`
	Value string `json:"value"`
}

// SwitchVideo sends the command to route an input to a specific output.
// e.g. output 1, input 4 -> set {"path":"/switcher/1/output/video/input","value":"4"}
func (c *client) SwitchVideo(ctx context.Context, output int, input int) error {
	if output <= 0 {
		return fmt.Errorf("invalid output channel: %d (must be > 0)", output)
	}
	if input < 0 {
		return fmt.Errorf("invalid input channel: %d (must be >= 0)", input)
	}

	payloadObj := switchPayload{
		Path:  fmt.Sprintf("/switcher/%d/output/video/input", output),
		Value: fmt.Sprintf("%d", input),
	}

	jsonBytes, err := json.Marshal(payloadObj)
	if err != nil {
		return fmt.Errorf("failed to marshal switch payload: %w", err)
	}

	bodyStr := "set " + string(jsonBytes)
	reqURL := c.baseURL + defaultEndpoint

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, strings.NewReader(bodyStr))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Match headers expected by AMX DVX WebControl AJAX interface
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("Origin", c.origin)
	req.Header.Set("Referer", c.referer)
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to communicate with AMX switcher at %s: %w", reqURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("AMX switcher returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	return nil
}

// Ping checks whether the AMX switcher HTTP service is reachable.
func (c *client) Ping(ctx context.Context) error {
	reqURL := c.baseURL + defaultEndpoint
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create ping request: %w", err)
	}
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("ping failed for %s: %w", c.baseURL, err)
	}
	defer resp.Body.Close()

	// As long as the host responds (even with 200, 404, or 405 Method Not Allowed), network connectivity is confirmed
	if resp.StatusCode >= 500 {
		return fmt.Errorf("AMX switcher server error HTTP %d", resp.StatusCode)
	}

	return nil
}
