package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestClaudeCreditsToDollars(t *testing.T) {
	if got := claudeCreditsToDollars(5000); got != 50 {
		t.Fatalf("expected 5000 credits -> 50 dollars, got %v", got)
	}
}

func TestClaudeTokenExpired(t *testing.T) {
	creds := &claudeCredentials{}

	if claudeTokenExpired(creds) {
		t.Fatal("expected token with missing expiry to be treated as not expired")
	}

	creds.ClaudeAiOauth.ExpiresAt = time.Now().Add(20 * time.Second).UnixMilli()
	if !claudeTokenExpired(creds) {
		t.Fatal("expected token within 30s buffer to be treated as expired")
	}

	creds.ClaudeAiOauth.ExpiresAt = time.Now().Add(2 * time.Minute).UnixMilli()
	if claudeTokenExpired(creds) {
		t.Fatal("expected token beyond buffer to be treated as valid")
	}
}

func TestIsClaudeAuthStatus(t *testing.T) {
	if !isClaudeAuthStatus(http.StatusUnauthorized) {
		t.Fatal("401 should be auth status")
	}
	if !isClaudeAuthStatus(http.StatusForbidden) {
		t.Fatal("403 should be auth status")
	}
	if isClaudeAuthStatus(http.StatusInternalServerError) {
		t.Fatal("500 should not be auth status")
	}
}

func TestLoadClaudeCredentialsRequiresAccessToken(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	path := filepath.Join(home, ".claude", ".credentials.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	if err := os.WriteFile(path, []byte(`{"claudeAiOauth":{}}`), 0o600); err != nil {
		t.Fatalf("write creds: %v", err)
	}

	_, err := loadClaudeCredentials()
	if err == nil || !strings.Contains(err.Error(), "no Claude OAuth access token found") {
		t.Fatalf("expected missing access token error, got %v", err)
	}
}

func TestSaveClaudeCredentialsPreservesUnknownFields(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	path := filepath.Join(home, ".claude", ".credentials.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	original := `{"custom":"keep-me","claudeAiOauth":{"accessToken":"old","refreshToken":"old","subscriptionType":"max"}}`
	if err := os.WriteFile(path, []byte(original), 0o600); err != nil {
		t.Fatalf("write original creds: %v", err)
	}

	creds := &claudeCredentials{}
	creds.ClaudeAiOauth.AccessToken = "new-access"
	creds.ClaudeAiOauth.RefreshToken = "new-refresh"
	creds.ClaudeAiOauth.ExpiresAt = 123456

	if err := saveClaudeCredentials(creds); err != nil {
		t.Fatalf("save creds: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read creds: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if parsed["custom"] != "keep-me" {
		t.Fatalf("expected custom field to be preserved, got %#v", parsed["custom"])
	}

	oauth := parsed["claudeAiOauth"].(map[string]any)
	if oauth["accessToken"] != "new-access" {
		t.Fatalf("expected updated access token, got %#v", oauth["accessToken"])
	}
	if oauth["refreshToken"] != "new-refresh" {
		t.Fatalf("expected updated refresh token, got %#v", oauth["refreshToken"])
	}
}

func TestFetchClaudeUsageSuccess(t *testing.T) {
	withMockDefaultClient(t, func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", req.Method)
		}
		if req.URL.String() != claudeUsageURL {
			t.Fatalf("unexpected URL: %s", req.URL.String())
		}
		if req.Header.Get("Authorization") != "Bearer token123" {
			t.Fatalf("missing auth header: %q", req.Header.Get("Authorization"))
		}
		if req.Header.Get("anthropic-beta") != "oauth-2025-04-20" {
			t.Fatalf("missing beta header: %q", req.Header.Get("anthropic-beta"))
		}

		return jsonResponse(http.StatusOK, `{"five_hour":{"utilization":42,"resets_at":"2026-02-18T10:00:00Z"}}`), nil
	})

	usage, status, err := fetchClaudeUsage(context.Background(), "token123")
	if err != nil {
		t.Fatalf("fetch usage error: %v", err)
	}
	if status != http.StatusOK {
		t.Fatalf("expected 200, got %d", status)
	}
	if usage.FiveHour == nil || usage.FiveHour.Utilization != 42 {
		t.Fatalf("unexpected usage payload: %#v", usage)
	}
}

func TestFetchClaudeUsageNonOKStatus(t *testing.T) {
	withMockDefaultClient(t, func(req *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusUnauthorized, `{}`), nil
	})

	_, status, err := fetchClaudeUsage(context.Background(), "token123")
	if err != nil {
		t.Fatalf("expected nil error on non-200 status, got %v", err)
	}
	if status != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", status)
	}
}

func TestRefreshClaudeAuthSuccess(t *testing.T) {
	withMockDefaultClient(t, func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", req.Method)
		}
		if req.URL.String() != claudeTokenURL {
			t.Fatalf("unexpected URL: %s", req.URL.String())
		}

		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		values, err := url.ParseQuery(string(body))
		if err != nil {
			t.Fatalf("parse form body: %v", err)
		}
		if values.Get("refresh_token") != "old-refresh" {
			t.Fatalf("unexpected refresh token in form: %q", values.Get("refresh_token"))
		}
		if values.Get("grant_type") != "refresh_token" {
			t.Fatalf("unexpected grant type: %q", values.Get("grant_type"))
		}
		if values.Get("client_id") != claudeOAuthClientID {
			t.Fatalf("unexpected client id: %q", values.Get("client_id"))
		}

		if req.Header.Get("User-Agent") != claudeUserAgent {
			t.Fatalf("unexpected User-Agent: %q", req.Header.Get("User-Agent"))
		}

		return jsonResponse(http.StatusOK, `{"access_token":"new-access","refresh_token":"new-refresh","expires_in":120}`), nil
	})

	creds := &claudeCredentials{}
	creds.ClaudeAiOauth.RefreshToken = "old-refresh"
	creds.ClaudeAiOauth.AccessToken = "old-access"

	if err := refreshClaudeAuth(context.Background(), creds); err != nil {
		t.Fatalf("refresh auth: %v", err)
	}

	if creds.ClaudeAiOauth.AccessToken != "new-access" {
		t.Fatalf("expected updated access token, got %q", creds.ClaudeAiOauth.AccessToken)
	}
	if creds.ClaudeAiOauth.RefreshToken != "new-refresh" {
		t.Fatalf("expected updated refresh token, got %q", creds.ClaudeAiOauth.RefreshToken)
	}
	if creds.ClaudeAiOauth.ExpiresAt <= time.Now().UnixMilli() {
		t.Fatalf("expected ExpiresAt in the future, got %d", creds.ClaudeAiOauth.ExpiresAt)
	}
}

func TestRefreshClaudeAuthRequiresRefreshToken(t *testing.T) {
	creds := &claudeCredentials{}
	err := refreshClaudeAuth(context.Background(), creds)
	if err == nil || !strings.Contains(err.Error(), "no Claude refresh token found") {
		t.Fatalf("expected missing refresh token error, got %v", err)
	}
}
