package amx

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSwitchVideo_Success(t *testing.T) {
	expectedPath := "/web/module/DVX-Switcher-Dashboard/com.amx.dvx/hcontrol"
	expectedBody := `set {"path":"/switcher/1/output/video/input","value":"4"}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != expectedPath {
			t.Errorf("expected path %q, got %q", expectedPath, r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		if got := r.Header.Get("Content-Type"); got != "application/x-www-form-urlencoded; charset=UTF-8" {
			t.Errorf("unexpected Content-Type: %s", got)
		}
		if got := r.Header.Get("X-Requested-With"); got != "XMLHttpRequest" {
			t.Errorf("unexpected X-Requested-With: %s", got)
		}
		if got := r.Header.Get("Origin"); got == "" {
			t.Errorf("expected Origin header, got empty")
		}
		if got := r.Header.Get("Referer"); got == "" {
			t.Errorf("expected Referer header, got empty")
		}

		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read body: %v", err)
		}
		if string(bodyBytes) != expectedBody {
			t.Errorf("expected body %q, got %q", expectedBody, string(bodyBytes))
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer server.Close()

	client, err := NewClient(Config{
		Host:    server.URL,
		Timeout: 2 * time.Second,
	})
	if err != nil {
		t.Fatalf("unexpected error creating client: %v", err)
	}

	err = client.SwitchVideo(context.Background(), 1, 4)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
}

func TestSwitchVideo_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal device error"))
	}))
	defer server.Close()

	client, err := NewClient(Config{Host: server.URL})
	if err != nil {
		t.Fatalf("unexpected error creating client: %v", err)
	}

	err = client.SwitchVideo(context.Background(), 1, 2)
	if err == nil {
		t.Fatal("expected error on HTTP 500, got nil")
	}
}

func TestSwitchVideo_Validation(t *testing.T) {
	client, err := NewClient(Config{Host: "http://127.0.0.1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Invalid output (0)
	if err := client.SwitchVideo(context.Background(), 0, 1); err == nil {
		t.Errorf("expected error for output 0, got nil")
	}

	// Negative output (-1)
	if err := client.SwitchVideo(context.Background(), -1, 1); err == nil {
		t.Errorf("expected error for negative output, got nil")
	}

	// Negative input (-1)
	if err := client.SwitchVideo(context.Background(), 1, -1); err == nil {
		t.Errorf("expected error for negative input, got nil")
	}
}

func TestPing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := NewClient(Config{Host: server.URL})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := client.Ping(context.Background()); err != nil {
		t.Fatalf("ping failed: %v", err)
	}
}
