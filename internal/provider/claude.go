package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type Claude struct{}

func (Claude) Name() string { return "Claude" }

type claudeCredentials struct {
	ClaudeAiOauth struct {
		AccessToken      string `json:"accessToken"`
		SubscriptionType string `json:"subscriptionType"`
	} `json:"claudeAiOauth"`
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

func (c Claude) Fetch(ctx context.Context) Result {
	r := Result{Name: "Claude"}

	creds, err := loadClaudeCredentials()
	if err != nil {
		r.Error = err
		r.Short = "?"
		return r
	}

	r.Plan = creds.ClaudeAiOauth.SubscriptionType

	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.anthropic.com/api/oauth/usage", nil)
	if err != nil {
		r.Error = err
		r.Short = "?"
		return r
	}
	req.Header.Set("Authorization", "Bearer "+creds.ClaudeAiOauth.AccessToken)
	req.Header.Set("anthropic-beta", "oauth-2025-04-20")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		r.Error = err
		r.Short = "?"
		return r
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		r.Error = fmt.Errorf("auth failed (HTTP %d)", resp.StatusCode)
		r.Short = "!"
		return r
	}

	if resp.StatusCode != 200 {
		r.Error = fmt.Errorf("HTTP %d", resp.StatusCode)
		r.Short = "?"
		return r
	}

	var usage claudeUsageResponse
	if err := json.NewDecoder(resp.Body).Decode(&usage); err != nil {
		r.Error = err
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
		remaining := usage.ExtraUsage.MonthlyLimit - usage.ExtraUsage.UsedCredits
		r.Credits = &remaining
	}

	r.Identity = fetchClaudeProfile(ctx, creds.ClaudeAiOauth.AccessToken)

	return r
}

func fetchClaudeProfile(ctx context.Context, token string) string {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.anthropic.com/api/oauth/profile", nil)
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
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(filepath.Join(home, ".claude", ".credentials.json"))
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
