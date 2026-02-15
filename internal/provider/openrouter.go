package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
)

type OpenRouter struct{}

func (OpenRouter) Name() string { return "OpenRouter" }

type openRouterKeyResponse struct {
	Data struct {
		Label          string   `json:"label"`
		Limit          *float64 `json:"limit"`
		LimitRemaining *float64 `json:"limit_remaining"`
		Usage          float64  `json:"usage"`
		UsageDaily     float64  `json:"usage_daily"`
		UsageWeekly    float64  `json:"usage_weekly"`
		UsageMonthly   float64  `json:"usage_monthly"`
		IsFreeTier     bool     `json:"is_free_tier"`
	} `json:"data"`
}

func (o OpenRouter) Fetch(ctx context.Context) Result {
	r := Result{Name: "OpenRouter"}

	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		r.Error = fmt.Errorf("OPENROUTER_API_KEY not set")
		r.Short = "?"
		return r
	}

	req, err := http.NewRequestWithContext(ctx, "GET", "https://openrouter.ai/api/v1/key", nil)
	if err != nil {
		r.Error = err
		r.Short = "?"
		return r
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

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

	var keyResp openRouterKeyResponse
	if err := json.NewDecoder(resp.Body).Decode(&keyResp); err != nil {
		r.Error = err
		r.Short = "?"
		return r
	}

	d := keyResp.Data
	if d.Label != "" && !strings.HasPrefix(d.Label, "sk-") {
		r.Identity = d.Label
	}

	if d.LimitRemaining != nil {
		credits := *d.LimitRemaining
		r.Credits = &credits
		r.Short = fmt.Sprintf("$%.2f", credits)
		if d.Limit != nil && *d.Limit > 0 {
			usedPct := (d.Usage / *d.Limit) * 100
			r.Class = classFromPct(usedPct)
			r.Windows = append(r.Windows, RateWindow{
				Label:   "Budget",
				UsedPct: usedPct,
			})
		}
	} else {
		r.Short = fmt.Sprintf("$%.2f", d.UsageMonthly)
	}

	r.Spend = []SpendEntry{
		{Label: "Today", Amount: d.UsageDaily},
		{Label: "This week", Amount: d.UsageWeekly},
		{Label: "This month", Amount: d.UsageMonthly},
		{Label: "All time", Amount: d.Usage},
	}

	if d.IsFreeTier {
		r.Plan = "free"
	}

	return r
}
