package ai

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestChatStreamParsesSSE(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/chat/completions") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		flusher := w.(http.Flusher)
		frames := []string{
			`data: {"choices":[{"delta":{"content":"CREATE "}}]}`,
			``,
			`: keep-alive comment`,
			`data: {"choices":[{"delta":{"content":"TABLE users"}}]}`,
			`data: {"choices":[{"delta":{},"finish_reason":"stop"}]}`,
			`data: [DONE]`,
			`data: {"choices":[{"delta":{"content":"MUST NOT APPEAR"}}]}`,
		}
		for _, f := range frames {
			_, _ = w.Write([]byte(f + "\n\n"))
			flusher.Flush()
		}
	}))
	defer srv.Close()

	var deltas []string
	client := NewClient(&Config{
		Provider: "openai",
		BaseURL:  srv.URL,
		APIKey:   "test-key",
		Model:    "test-model",
	})
	content, err := client.ChatStream("sys", "user", func(d string) { deltas = append(deltas, d) })
	if err != nil {
		t.Fatalf("ChatStream: %v", err)
	}
	if content != "CREATE TABLE users" {
		t.Fatalf("content = %q, want %q", content, "CREATE TABLE users")
	}
	if strings.Join(deltas, "") != content {
		t.Fatalf("deltas %v do not assemble to content %q", deltas, content)
	}
}

func TestChatStreamReportsAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"invalid api key","type":"auth"}}`))
	}))
	defer srv.Close()

	client := NewClient(&Config{Provider: "openai", BaseURL: srv.URL, APIKey: "bad", Model: "m"})
	_, err := client.ChatStream("sys", "user", nil)
	if err == nil || !strings.Contains(err.Error(), "invalid api key") {
		t.Fatalf("expected API error, got %v", err)
	}
}

func TestChatNonStreamStillWorks(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"hi"},"finish_reason":"stop"}]}`))
	}))
	defer srv.Close()

	client := NewClient(&Config{Provider: "openai", BaseURL: srv.URL, APIKey: "k", Model: "m"})
	content, err := client.Chat("sys", "user")
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if content != "hi" {
		t.Fatalf("content = %q", content)
	}
}

func TestConfigPersistRoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("AppData", dir)   // windows
	t.Setenv("XDG_CONFIG_HOME", dir) // linux/darwin

	saved := &Config{Provider: "deepseek", BaseURL: "https://api.deepseek.com/v1", APIKey: "sk-x", Model: "deepseek-chat", Temperature: 0.3, MaxTokens: 2048}
	if err := SaveConfig(saved); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	loaded := LoadConfig()
	if loaded.Provider != "deepseek" || loaded.Model != "deepseek-chat" || loaded.APIKey != "sk-x" {
		t.Fatalf("loaded config mismatch: %+v", loaded)
	}
	if loaded.Temperature != 0.3 || loaded.MaxTokens != 2048 {
		t.Fatalf("numeric fields mismatch: %+v", loaded)
	}
}

func TestLoadConfigFallsBackToDefault(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("AppData", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)

	cfg := LoadConfig()
	def := DefaultConfig()
	if cfg.Provider != def.Provider || cfg.Model != def.Model {
		t.Fatalf("missing file should fall back to defaults, got %+v", cfg)
	}
}
