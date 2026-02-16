package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Codex struct{}

func (Codex) Name() string { return "Codex" }

type codexAuth struct {
	Tokens struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		IDToken      string `json:"id_token"`
	} `json:"tokens"`
}

type codexRefreshResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token"`
}

type codexUsageResponse struct {
	Email     string          `json:"email"`
	PlanType  string          `json:"plan_type"`
	RateLimit *codexRateLimit `json:"rate_limit"`
}

type codexRateLimit struct {
	Allowed         bool         `json:"allowed"`
	LimitReached    bool         `json:"limit_reached"`
	PrimaryWindow   *codexWindow `json:"primary_window"`
	SecondaryWindow *codexWindow `json:"secondary_window"`
}

type codexWindow struct {
	UsedPercent     float64 `json:"used_percent"`
	LimitWindowSecs int     `json:"limit_window_seconds"`
	ResetAfterSecs  int     `json:"reset_after_seconds"`
	ResetAt         int64   `json:"reset_at"`
}

const (
	codexUsageURL        = "https://chatgpt.com/backend-api/wham/usage"
	codexTokenURL        = "https://auth.openai.com/oauth/token"
	codexOAuthClientID   = "app_EMoamEEZ73f0CkXaXp7hrann"
	codexAuthFailedError = "codex auth expired; run `codex login`"
)

func (c Codex) Fetch(ctx context.Context) Result {
	r := Result{Name: "Codex"}

	auth, err := loadCodexAuth()
	if err != nil {
		r.Error = err
		r.Short = "?"
		return r
	}

	usage, status, err := fetchCodexUsage(ctx, auth.Tokens.AccessToken)
	if err != nil {
		r.Error = err
		r.Short = "?"
		return r
	}

	if isCodexAuthStatus(status) {
		if err := refreshCodexAuth(ctx, auth); err != nil {
			r.Error = fmt.Errorf("%s (%v)", codexAuthFailedError, err)
			r.Short = "!"
			return r
		}

		if err := saveCodexAuth(auth); err != nil {
			r.Error = fmt.Errorf("codex token refresh succeeded, but failed to save updated tokens: %w", err)
			r.Short = "!"
			return r
		}

		usage, status, err = fetchCodexUsage(ctx, auth.Tokens.AccessToken)
		if err != nil {
			r.Error = err
			r.Short = "?"
			return r
		}
	}

	if isCodexAuthStatus(status) {
		r.Error = fmt.Errorf("%s (HTTP %d)", codexAuthFailedError, status)
		r.Short = "!"
		return r
	}

	if status != http.StatusOK {
		r.Error = fmt.Errorf("HTTP %d", status)
		r.Short = "?"
		return r
	}

	r.Plan = usage.PlanType
	r.Identity = usage.Email

	if usage.RateLimit != nil {
		rl := usage.RateLimit

		if rl.PrimaryWindow != nil {
			pw := rl.PrimaryWindow
			w := RateWindow{
				Label:   "Session (5h)",
				UsedPct: pw.UsedPercent,
			}
			if pw.ResetAt > 0 {
				w.ResetAt = time.Unix(pw.ResetAt, 0)
				w.HasReset = true
			}
			r.Windows = append(r.Windows, w)
			r.Short = fmt.Sprintf("%.0f%%", pw.UsedPercent)
			r.Class = classFromPct(pw.UsedPercent)
		}

		if rl.SecondaryWindow != nil {
			sw := rl.SecondaryWindow
			w := RateWindow{
				Label:   "Weekly (7d)",
				UsedPct: sw.UsedPercent,
			}
			if sw.ResetAt > 0 {
				w.ResetAt = time.Unix(sw.ResetAt, 0)
				w.HasReset = true
			}
			r.Windows = append(r.Windows, w)
		}

		if rl.LimitReached {
			r.Class = "critical"
		}
	}

	return r
}

func fetchCodexUsage(ctx context.Context, accessToken string) (codexUsageResponse, int, error) {
	var usage codexUsageResponse

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, codexUsageURL, nil)
	if err != nil {
		return usage, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return usage, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return usage, resp.StatusCode, nil
	}

	if err := json.NewDecoder(resp.Body).Decode(&usage); err != nil {
		return usage, 0, err
	}

	return usage, http.StatusOK, nil
}

func refreshCodexAuth(ctx context.Context, auth *codexAuth) error {
	if auth.Tokens.RefreshToken == "" {
		return fmt.Errorf("no Codex refresh token found")
	}

	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", auth.Tokens.RefreshToken)
	form.Set("client_id", codexOAuthClientID)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, codexTokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("token refresh HTTP %d", resp.StatusCode)
	}

	var refreshed codexRefreshResponse
	if err := json.NewDecoder(resp.Body).Decode(&refreshed); err != nil {
		return err
	}

	if refreshed.AccessToken == "" {
		return fmt.Errorf("token refresh returned no access token")
	}

	auth.Tokens.AccessToken = refreshed.AccessToken
	if refreshed.RefreshToken != "" {
		auth.Tokens.RefreshToken = refreshed.RefreshToken
	}
	if refreshed.IDToken != "" {
		auth.Tokens.IDToken = refreshed.IDToken
	}

	return nil
}

func isCodexAuthStatus(status int) bool {
	return status == http.StatusUnauthorized || status == http.StatusForbidden
}

func loadCodexAuth() (*codexAuth, error) {
	path, err := codexAuthPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var auth codexAuth
	if err := json.Unmarshal(data, &auth); err != nil {
		return nil, err
	}

	if auth.Tokens.AccessToken == "" {
		return nil, fmt.Errorf("no Codex access token found")
	}

	return &auth, nil
}

func saveCodexAuth(auth *codexAuth) error {
	path, err := codexAuthPath()
	if err != nil {
		return err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	tokens, _ := raw["tokens"].(map[string]any)
	if tokens == nil {
		tokens = map[string]any{}
	}

	tokens["access_token"] = auth.Tokens.AccessToken
	tokens["refresh_token"] = auth.Tokens.RefreshToken
	if auth.Tokens.IDToken != "" {
		tokens["id_token"] = auth.Tokens.IDToken
	}
	raw["tokens"] = tokens
	raw["last_refresh"] = time.Now().UTC().Format(time.RFC3339)

	updated, err := json.Marshal(raw)
	if err != nil {
		return err
	}

	return os.WriteFile(path, updated, 0o600)
}

func codexAuthPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, ".codex", "auth.json"), nil
}
