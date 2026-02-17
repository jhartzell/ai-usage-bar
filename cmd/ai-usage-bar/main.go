package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jhartzell/ai-usage-bar/internal/cache"
	"github.com/jhartzell/ai-usage-bar/internal/detail"
	"github.com/jhartzell/ai-usage-bar/internal/provider"
	"github.com/jhartzell/ai-usage-bar/internal/recovery"
	"github.com/jhartzell/ai-usage-bar/internal/waybar"
)

func main() {
	if handled := handleCommand(os.Args[1:]); handled {
		return
	}

	results := cache.Load()
	if results == nil {
		providers := []provider.Provider{
			provider.Claude{},
			provider.Codex{},
			provider.OpenRouter{},
		}
		ctx := context.Background()
		results = provider.FetchAll(ctx, providers)
		cache.Save(results)
	}

	if len(os.Args) > 1 && os.Args[1] == "--detail" {
		detail.ShowYad(results)
		return
	}

	output := waybar.Format(results)
	fmt.Println(waybar.FormatJSON(output))
}

func handleCommand(args []string) bool {
	if len(args) == 0 {
		return false
	}

	switch args[0] {
	case "--detail":
		return false
	case "--recover-auth":
		if err := recovery.RunAuthRecovery(context.Background(), os.Stdin, os.Stdout, os.Stderr); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return true
	case "--clear-cache":
		if err := cache.Clear(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("Cache cleared.")
		return true
	case "-h", "--help":
		printUsage()
		return true
	default:
		fmt.Fprintf(os.Stderr, "unknown flag: %s\n\n", args[0])
		printUsage()
		os.Exit(2)
		return true
	}
}

func printUsage() {
	fmt.Println("Usage: ai-usage-bar [--detail|--recover-auth|--clear-cache]")
	fmt.Println()
	fmt.Println("  --detail         Open popup with provider details")
	fmt.Println("  --recover-auth   Run provider login flows and clear cache")
	fmt.Println("  --clear-cache    Remove cached usage data")
}
