package detail

import (
	"bufio"
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"html"
	"html/template"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/jhartzell/ai-usage-bar/internal/provider"
)

//go:embed popup.html.tmpl
var popupTemplate string

var popupHTMLTemplate = template.Must(template.New("popup").Parse(popupTemplate))

type popupData struct {
	Providers        []providerView
	ShowRecoverAuth  bool
	RecoverAuthLabel string
}

type providerView struct {
	Class        string
	Name         string
	Plan         string
	Identity     string
	Error        string
	Windows      []windowView
	Spend        []spendView
	ShowCredits  bool
	CreditsLabel string
	CreditsValue float64
	NoData       bool
}

type windowView struct {
	Label   string
	UsedPct float64
	Color   string
	Reset   string
}

type spendView struct {
	Label  string
	Amount float64
}

func ShowYad(results []provider.Result) {
	action, err := showYadOnce(results)
	if err != nil {
		return
	}

	switch action {
	case "recover-auth":
		_ = startRecoveryInTerminal(context.Background())
	}
}

func showYadOnce(results []provider.Result) (string, error) {
	htmlDoc := renderHTML(results)
	width, height := popupSize(results)

	cmd := exec.Command("yad",
		"--html",
		"--title=AI Usage",
		fmt.Sprintf("--width=%d", width),
		fmt.Sprintf("--height=%d", height),
		"--borders=0",
		"--print-uri",
		"--css=window,dialog,box,frame,scrolledwindow,viewport,grid { border: 0; box-shadow: none; background: #303446; } scrollbar, scrollbar slider { min-width: 0; min-height: 0; opacity: 0; }",
		"--user-style=html,body{margin:0;padding:0;background:#303446;overflow:hidden;}::-webkit-scrollbar{width:0;height:0;}",
		"--hscroll-policy=never",
		"--vscroll-policy=never",
		"--center",
		"--no-buttons",
		"--undecorated",
		"--skip-taskbar",
		"--class=ai-usage-detail",
	)
	cmd.Stdin = strings.NewReader(htmlDoc)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return "", err
	}

	actionCh := make(chan string, 1)
	go func() {
		s := bufio.NewScanner(stdout)
		for s.Scan() {
			uri := strings.TrimSpace(s.Text())
			switch uri {
			case "ai-usage-bar://recover-auth":
				select {
				case actionCh <- "recover-auth":
				default:
				}
				_ = cmd.Process.Signal(os.Interrupt)
				return
			}
		}
	}()

	_ = cmd.Wait()

	select {
	case action := <-actionCh:
		return action, nil
	default:
		return "", nil
	}
}

func renderHTML(results []provider.Result) string {
	data := popupData{Providers: make([]providerView, 0, len(results))}
	for _, r := range results {
		data.Providers = append(data.Providers, toProviderView(r))
	}

	data.ShowRecoverAuth = shouldShowRecoverAuth(results)
	data.RecoverAuthLabel = "Recover auth"

	var buf bytes.Buffer
	if err := popupHTMLTemplate.Execute(&buf, data); err != nil {
		return fmt.Sprintf("<html><body style=\"background:#303446;color:#e78284;font-family:monospace;padding:12px;\">failed to render detail popup: %s</body></html>", html.EscapeString(err.Error()))
	}

	return buf.String()
}

func shouldShowRecoverAuth(results []provider.Result) bool {
	for _, r := range results {
		if r.Short == "!" {
			return true
		}
		if r.Error != nil {
			err := strings.ToLower(r.Error.Error())
			if strings.Contains(err, "auth") || strings.Contains(err, "expired") || strings.Contains(err, "token") {
				return true
			}
		}
	}

	return false
}

func startRecoveryInTerminal(ctx context.Context) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}

	if err := spawnInTerminal(ctx, exe, []string{"--recover-auth"}); err == nil {
		return nil
	}

	// Fall back to running directly (may still open browser-based flows).
	return exec.CommandContext(ctx, exe, "--recover-auth").Run()
}

func spawnInTerminal(ctx context.Context, exe string, args []string) error {
	terminals := []struct {
		name string
		args func(cmdline string) []string
	}{
		{name: "foot", args: func(cmdline string) []string { return []string{"-e", "sh", "-lc", cmdline} }},
		{name: "alacritty", args: func(cmdline string) []string { return []string{"-e", "sh", "-lc", cmdline} }},
		{name: "kitty", args: func(cmdline string) []string { return []string{"sh", "-lc", cmdline} }},
		{name: "wezterm", args: func(cmdline string) []string { return []string{"start", "--", "sh", "-lc", cmdline} }},
		{name: "gnome-terminal", args: func(cmdline string) []string { return []string{"--", "sh", "-lc", cmdline} }},
		{name: "konsole", args: func(cmdline string) []string { return []string{"-e", "sh", "-lc", cmdline} }},
		{name: "xterm", args: func(cmdline string) []string { return []string{"-e", "sh", "-lc", cmdline} }},
	}

	cmdline := shellCmdline(exe, args)
	for _, t := range terminals {
		if _, err := exec.LookPath(t.name); err != nil {
			continue
		}
		cmd := exec.CommandContext(ctx, t.name, t.args(cmdline)...)
		return cmd.Start()
	}

	return fmt.Errorf("no terminal emulator found")
}

func shellCmdline(exe string, args []string) string {
	parts := make([]string, 0, 1+len(args)+4)
	parts = append(parts, shellQuote(exe))
	for _, a := range args {
		parts = append(parts, shellQuote(a))
	}

	// Keep the terminal open so users can read output.
	parts = append(parts, ";", "echo", "\"\"", ";", "echo", shellQuote("Done. Press Enter to close."), ";", "read")
	return strings.Join(parts, " ")
}

func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

func toProviderView(r provider.Result) providerView {
	v := providerView{
		Class:    providerClass(r.Name),
		Name:     r.Name,
		Plan:     r.Plan,
		Identity: r.Identity,
		Windows:  make([]windowView, 0, len(r.Windows)),
		Spend:    make([]spendView, 0, len(r.Spend)),
	}

	if r.Error != nil {
		v.Error = r.Error.Error()
		return v
	}

	for _, w := range r.Windows {
		usedPct := clampPct(w.UsedPct)
		resetStr := ""
		if w.HasReset {
			d := time.Until(w.ResetAt)
			if d > 0 {
				resetStr = fmt.Sprintf("resets in %s", formatDuration(d))
			}
		}

		v.Windows = append(v.Windows, windowView{
			Label:   w.Label,
			UsedPct: usedPct,
			Color:   colorForPct(usedPct),
			Reset:   resetStr,
		})
	}

	for _, s := range r.Spend {
		v.Spend = append(v.Spend, spendView{
			Label:  s.Label,
			Amount: s.Amount,
		})
	}

	if r.Credits != nil {
		v.ShowCredits = true
		v.CreditsValue = *r.Credits
		if r.Name == "Claude" {
			v.CreditsLabel = "Extra usage remaining"
		} else {
			v.CreditsLabel = "Credits"
		}
	}

	v.NoData = len(v.Windows) == 0 && len(v.Spend) == 0 && !v.ShowCredits
	return v
}

func providerClass(name string) string {
	return strings.ToLower(strings.ReplaceAll(name, " ", ""))
}

func clampPct(pct float64) float64 {
	if pct < 0 {
		return 0
	}
	if pct > 100 {
		return 100
	}

	return pct
}

func colorForPct(pct float64) string {
	switch {
	case pct >= 90:
		return "#e78284"
	case pct >= 75:
		return "#e5c890"
	default:
		return "#a6d189"
	}
}

func formatDuration(d time.Duration) string {
	if d <= 0 {
		return "now"
	}
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	mins := int(d.Minutes()) % 60

	switch {
	case days > 0:
		return fmt.Sprintf("%dd %dh", days, hours)
	case hours > 0:
		return fmt.Sprintf("%dh %dm", hours, mins)
	default:
		return fmt.Sprintf("%dm", mins)
	}
}

func popupSize(results []provider.Result) (int, int) {
	const width = 560

	height := 92
	for _, r := range results {
		rows := 1
		if r.Error == nil {
			rows = len(r.Windows) + len(r.Spend)
			if r.Credits != nil {
				rows++
			}
			if rows == 0 {
				rows = 1
			}
		}

		height += 62 + rows*22
	}

	if height < 300 {
		height = 300
	}
	if height > 760 {
		height = 760
	}

	return width, height
}
