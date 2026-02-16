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

type Claude struct{}

func (Claude) Name() string { return "Claude" }

type claudeCredentials struct {
	ClaudeAiOauth struct {
		AccessToken      string `json:"accessToken"`
		RefreshToken     string `json:"refreshToken"`
		ExpiresAt        int64  `json:"expiresAt"`
		SubscriptionType string `json:"subscriptionType"`
	} `json:"claudeAiOauth"`
}

type claudeRefreshResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

type claudeUsageResponse struct {
	FiveHour   *claudeWindow `json:"five_hour"`
	SevenDay   *claudeWindow `json:"seven_day"`
	ExtraUsage *claudeExtra  `json:"extra_usage"`
}

type claudeWindow struct {
	Utilization float64 `json:"utilization"`
	ResetsAt    string  `json:"resets_at"`
}

type claudeExtra struct {
	IsEnabled    bool     `json:"is_enabled"`
	MonthlyLimit float64  `json:"monthly_limit"`
	UsedCredits  float64  `json:"used_credits"`
	Utilization  *float64 `json:"utilization"`
}

type claudeProfileResponse struct {
	Account struct {
		Email       string `json:"email"`
		DisplayName string `json:"display_name"`
	} `json:"account"`
}

const (
	claudeUsageURL        = "https://api.anthropic.com/api/oauth/usage"
	claudeProfileURL      = "https://api.anthropic.com/api/oauth/profile"
	claudeTokenURL        = "https://platform.claude.com/v1/oauth/token"
	claudeOAuthClientID   = "9d1c250a-e61b-44d9-88ed-5944d1962f5e"
	claudeUserAgent       = "claude-code/2.1.32"
	claudeAuthFailedError = "claude auth expired; run `claude login`"
)

func (c Claude) Fetch(ctx context.Context) Result {
	r := Result{Name: "Claude"}

	creds, err := loadClaudeCredentials()
	if err != nil {
		r.Error = err
		r.Short = "?"
		return r
	}

	r.Plan = creds.ClaudeAiOauth.SubscriptionType

	if claudeTokenExpired(creds) {
		if err := refreshClaudeAuth(ctx, creds); err == nil {
			_ = saveClaudeCredentials(creds)
		}
	}

	usage, status, err := fetchClaudeUsage(ctx, creds.ClaudeAiOauth.AccessToken)
	if err != nil {
		r.Error = err
		r.Short = "?"
		return r
	}

	if isClaudeAuthStatus(status) {
		if err := refreshClaudeAuth(ctx, creds); err != nil {
			r.Error = fmt.Errorf("%s (%v)", claudeAuthFailedError, err)
			r.Short = "!"
			return r
		}

		if err := saveClaudeCredentials(creds); err != nil {
			r.Error = fmt.Errorf("claude token refresh succeeded, but failed to save updated tokens: %w", err)
			r.Short = "!"
			return r
		}

		usage, status, err = fetchClaudeUsage(ctx, creds.ClaudeAiOauth.AccessToken)
		if err != nil {
			r.Error = err
			r.Short = "?"
			return r
		}
	}

	if isClaudeAuthStatus(status) {
		r.Error = fmt.Errorf("%s (HTTP %d)", claudeAuthFailedError, status)
		r.Short = "!"
		return r
	}

	if status != http.StatusOK {
		r.Error = fmt.Errorf("HTTP %d", status)
		r.Short = "?"
		return r
	}

	if usage.FiveHour != nil {
		w := RateWindow{
			Label:   "Session (5h)",
			UsedPct: usage.FiveHour.Utilization,
		}
		if t, err := time.Parse(time.RFC3339, usage.FiveHour.ResetsAt); err == nil {
			w.ResetAt = t
			w.HasReset = true
		}
		r.Windows = append(r.Windows, w)
		r.Short = fmt.Sprintf("%.0f%%", usage.FiveHour.Utilization)
		r.Class = classFromPct(usage.FiveHour.Utilization)
	}

	if usage.SevenDay != nil {
		w := RateWindow{
			Label:   "Weekly (7d)",
			UsedPct: usage.SevenDay.Utilization,
		}
		if t, err := time.Parse(time.RFC3339, usage.SevenDay.ResetsAt); err == nil {
			w.ResetAt = t
			w.HasReset = true
		}
		r.Windows = append(r.Windows, w)
	}

	if usage.ExtraUsage != nil && usage.ExtraUsage.IsEnabled {
		remaining := claudeCreditsToDollars(usage.ExtraUsage.MonthlyLimit - usage.ExtraUsage.UsedCredits)
		if remaining < 0 {
			remaining = 0
		}
		r.Credits = &remaining
	}

	r.Identity = fetchClaudeProfile(ctx, creds.ClaudeAiOauth.AccessToken)

	return r
}

func fetchClaudeUsage(ctx context.Context, accessToken string) (claudeUsageResponse, int, error) {
	var usage claudeUsageResponse

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, claudeUsageURL, nil)
	if err != nil {
		return usage, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("anthropic-beta", "oauth-2025-04-20")

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

func refreshClaudeAuth(ctx context.Context, creds *claudeCredentials) error {
	if creds.ClaudeAiOauth.RefreshToken == "" {
		return fmt.Errorf("no Claude refresh token found")
	}

	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", creds.ClaudeAiOauth.RefreshToken)
	form.Set("client_id", claudeOAuthClientID)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, claudeTokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", claudeUserAgent)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("token refresh HTTP %d", resp.StatusCode)
	}

	var refreshed claudeRefreshResponse
	if err := json.NewDecoder(resp.Body).Decode(&refreshed); err != nil {
		return err
	}

	if refreshed.AccessToken == "" {
		return fmt.Errorf("token refresh returned no access token")
	}

	creds.ClaudeAiOauth.AccessToken = refreshed.AccessToken
	if refreshed.RefreshToken != "" {
		creds.ClaudeAiOauth.RefreshToken = refreshed.RefreshToken
	}
	if refreshed.ExpiresIn > 0 {
		creds.ClaudeAiOauth.ExpiresAt = time.Now().Add(time.Duration(refreshed.ExpiresIn) * time.Second).UnixMilli()
	}

	return nil
}

func claudeTokenExpired(creds *claudeCredentials) bool {
	if creds.ClaudeAiOauth.ExpiresAt <= 0 {
		return false
	}

	return time.Now().UnixMilli() >= (creds.ClaudeAiOauth.ExpiresAt - 30_000)
}

func isClaudeAuthStatus(status int) bool {
	return status == http.StatusUnauthorized || status == http.StatusForbidden
}

func fetchClaudeProfile(ctx context.Context, token string) string {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, claudeProfileURL, nil)
	if err != nil {
		return ""
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("anthropic-beta", "oauth-2025-04-20")

	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != 200 {
		return ""
	}
	defer resp.Body.Close()

	var profile claudeProfileResponse
	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		return ""
	}
	return profile.Account.Email
}

func loadClaudeCredentials() (*claudeCredentials, error) {
	path, err := claudeCredentialsPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var creds claudeCredentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, err
	}

	if creds.ClaudeAiOauth.AccessToken == "" {
		return nil, fmt.Errorf("no Claude OAuth access token found")
	}

	return &creds, nil
}

func saveClaudeCredentials(creds *claudeCredentials) error {
	path, err := claudeCredentialsPath()
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

	oauth, _ := raw["claudeAiOauth"].(map[string]any)
	if oauth == nil {
		oauth = map[string]any{}
	}

	oauth["accessToken"] = creds.ClaudeAiOauth.AccessToken
	oauth["refreshToken"] = creds.ClaudeAiOauth.RefreshToken
	if creds.ClaudeAiOauth.ExpiresAt > 0 {
		oauth["expiresAt"] = creds.ClaudeAiOauth.ExpiresAt
	}
	raw["claudeAiOauth"] = oauth

	updated, err := json.Marshal(raw)
	if err != nil {
		return err
	}

	return os.WriteFile(path, updated, 0o600)
}

func claudeCredentialsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, ".claude", ".credentials.json"), nil
}

func claudeCreditsToDollars(v float64) float64 {
	return v / 100
}
