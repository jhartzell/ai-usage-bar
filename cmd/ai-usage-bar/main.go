package main

import (
	"context"
	"fmt"
	"os"

	"github.com/joshhartzell/ai-usage-bar/internal/detail"
	"github.com/joshhartzell/ai-usage-bar/internal/provider"
	"github.com/joshhartzell/ai-usage-bar/internal/waybar"
)

func main() {
	providers := []provider.Provider{
		provider.Claude{},
		provider.Codex{},
		provider.OpenRouter{},
	}

	ctx := context.Background()
	results := provider.FetchAll(ctx, providers)

	if len(os.Args) > 1 && os.Args[1] == "--detail" {
		detail.ShowYad(results)
		return
	}

	output := waybar.Format(results)
	fmt.Println(waybar.FormatJSON(output))
}
