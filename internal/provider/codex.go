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

type Codex struct{}

func (Codex) Name() string { return "Codex" }

type codexAuth struct {
	Tokens struct {
		AccessToken string `json:"access_token"`
	} `json:"tokens"`
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

func (c Codex) Fetch(ctx context.Context) Result {
	r := Result{Name: "Codex"}

	auth, err := loadCodexAuth()
	if err != nil {
		r.Error = err
		r.Short = "?"
		return r
	}

	req, err := http.NewRequestWithContext(ctx, "GET", "https://chatgpt.com/backend-api/wham/usage", nil)
	if err != nil {
		r.Error = err
		r.Short = "?"
		return r
	}
	req.Header.Set("Authorization", "Bearer "+auth.Tokens.AccessToken)

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

	var usage codexUsageResponse
	if err := json.NewDecoder(resp.Body).Decode(&usage); err != nil {
		r.Error = err
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

func loadCodexAuth() (*codexAuth, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(filepath.Join(home, ".codex", "auth.json"))
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
