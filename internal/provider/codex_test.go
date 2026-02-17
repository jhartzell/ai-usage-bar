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
)

func TestIsCodexAuthStatus(t *testing.T) {
	if !isCodexAuthStatus(http.StatusUnauthorized) {
		t.Fatal("401 should be auth status")
	}
	if !isCodexAuthStatus(http.StatusForbidden) {
		t.Fatal("403 should be auth status")
	}
	if isCodexAuthStatus(http.StatusBadGateway) {
		t.Fatal("502 should not be auth status")
	}
}

func TestLoadCodexAuthRequiresAccessToken(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	path := filepath.Join(home, ".codex", "auth.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	if err := os.WriteFile(path, []byte(`{"tokens":{}}`), 0o600); err != nil {
		t.Fatalf("write auth: %v", err)
	}

	_, err := loadCodexAuth()
	if err == nil || !strings.Contains(err.Error(), "no Codex access token found") {
		t.Fatalf("expected missing access token error, got %v", err)
	}
}

func TestSaveCodexAuthPreservesUnknownFields(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	path := filepath.Join(home, ".codex", "auth.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	original := `{"meta":"keep-me","tokens":{"access_token":"old","refresh_token":"old"}}`
	if err := os.WriteFile(path, []byte(original), 0o600); err != nil {
		t.Fatalf("write original auth: %v", err)
	}

	auth := &codexAuth{}
	auth.Tokens.AccessToken = "new-access"
	auth.Tokens.RefreshToken = "new-refresh"
	auth.Tokens.IDToken = "new-id"

	if err := saveCodexAuth(auth); err != nil {
		t.Fatalf("save auth: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read auth: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if parsed["meta"] != "keep-me" {
		t.Fatalf("expected meta field to be preserved, got %#v", parsed["meta"])
	}

	tokens := parsed["tokens"].(map[string]any)
	if tokens["access_token"] != "new-access" {
		t.Fatalf("expected updated access token, got %#v", tokens["access_token"])
	}
	if tokens["refresh_token"] != "new-refresh" {
		t.Fatalf("expected updated refresh token, got %#v", tokens["refresh_token"])
	}
	if tokens["id_token"] != "new-id" {
		t.Fatalf("expected updated id token, got %#v", tokens["id_token"])
	}
}

func TestFetchCodexUsageSuccess(t *testing.T) {
	withMockDefaultClient(t, func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", req.Method)
		}
		if req.URL.String() != codexUsageURL {
			t.Fatalf("unexpected URL: %s", req.URL.String())
		}
		if req.Header.Get("Authorization") != "Bearer token123" {
			t.Fatalf("missing auth header: %q", req.Header.Get("Authorization"))
		}

		body := `{"email":"user@example.com","plan_type":"pro","rate_limit":{"primary_window":{"used_percent":40}}}`
		return jsonResponse(http.StatusOK, body), nil
	})

	usage, status, err := fetchCodexUsage(context.Background(), "token123")
	if err != nil {
		t.Fatalf("fetch usage error: %v", err)
	}
	if status != http.StatusOK {
		t.Fatalf("expected 200, got %d", status)
	}
	if usage.Email != "user@example.com" || usage.PlanType != "pro" {
		t.Fatalf("unexpected usage payload: %#v", usage)
	}
}

func TestFetchCodexUsageNonOKStatus(t *testing.T) {
	withMockDefaultClient(t, func(req *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusForbidden, `{}`), nil
	})

	_, status, err := fetchCodexUsage(context.Background(), "token123")
	if err != nil {
		t.Fatalf("expected nil error on non-200 status, got %v", err)
	}
	if status != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", status)
	}
}

func TestRefreshCodexAuthSuccess(t *testing.T) {
	withMockDefaultClient(t, func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", req.Method)
		}
		if req.URL.String() != codexTokenURL {
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
		if values.Get("client_id") != codexOAuthClientID {
			t.Fatalf("unexpected client id: %q", values.Get("client_id"))
		}

		return jsonResponse(http.StatusOK, `{"access_token":"new-access","refresh_token":"new-refresh","id_token":"new-id"}`), nil
	})

	auth := &codexAuth{}
	auth.Tokens.AccessToken = "old-access"
	auth.Tokens.RefreshToken = "old-refresh"
	auth.Tokens.IDToken = "old-id"

	if err := refreshCodexAuth(context.Background(), auth); err != nil {
		t.Fatalf("refresh auth: %v", err)
	}

	if auth.Tokens.AccessToken != "new-access" {
		t.Fatalf("expected updated access token, got %q", auth.Tokens.AccessToken)
	}
	if auth.Tokens.RefreshToken != "new-refresh" {
		t.Fatalf("expected updated refresh token, got %q", auth.Tokens.RefreshToken)
	}
	if auth.Tokens.IDToken != "new-id" {
		t.Fatalf("expected updated id token, got %q", auth.Tokens.IDToken)
	}
}

func TestRefreshCodexAuthRequiresRefreshToken(t *testing.T) {
	auth := &codexAuth{}
	err := refreshCodexAuth(context.Background(), auth)
	if err == nil || !strings.Contains(err.Error(), "no Codex refresh token found") {
		t.Fatalf("expected missing refresh token error, got %v", err)
	}
}
