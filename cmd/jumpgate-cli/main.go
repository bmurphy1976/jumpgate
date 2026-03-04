package main

import (
	"dashboard/config"
	"flag"
	"fmt"
	"os"
)

func main() {
	fs := flag.NewFlagSet("jumpgate-cli", flag.ExitOnError)
	configPath := fs.String("config", "", "path to jumpgate-cli.yaml config file")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: jumpgate-cli [--config PATH] <command> <subcommand> [flags]")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Commands:")
		fmt.Fprintln(os.Stderr, "  category list|get|create|update|delete")
		fmt.Fprintln(os.Stderr, "  bookmark create|get|update|delete|move|search")
		fmt.Fprintln(os.Stderr, "  keyword  list|add|delete|set|clear")
		fmt.Fprintln(os.Stderr, "  icon     list")
	}

	// Parse global flags up to first non-flag arg
	args := os.Args[1:]
	var remaining []string
	for i, arg := range args {
		if arg == "--config" && i+1 < len(args) {
			*configPath = args[i+1]
			remaining = append(args[:i], args[i+2:]...)
			break
		}
		if len(arg) > 9 && arg[:9] == "--config=" {
			*configPath = arg[9:]
			remaining = append(args[:i], args[i+1:]...)
			break
		}
	}
	if remaining == nil {
		remaining = args
	}

	if len(remaining) < 2 {
		fs.Usage()
		os.Exit(1)
	}

	cfg := loadCLIConfig(*configPath)
	client := &apiClient{baseURL: cfg.URL, token: cfg.Token}

	resource := remaining[0]
	verb := remaining[1]
	cmdArgs := remaining[2:]

	var err error
	switch resource {
	case "category":
		err = runCategory(client, verb, cmdArgs)
	case "bookmark":
		err = runBookmark(client, verb, cmdArgs)
	case "keyword":
		err = runKeyword(client, verb, cmdArgs)
	case "icon":
		err = runIcon(client, verb, cmdArgs)
	default:
		fmt.Fprintf(os.Stderr, "unknown resource: %s\n", resource)
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func loadCLIConfig(path string) config.CLIConfig {
	cfg := config.CLIConfig{URL: "http://localhost:8080"}

	if path != "" {
		loaded, err := config.LoadCLIConfig(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		cfg = loaded
	} else {
		// Search standard config locations
		for _, p := range config.DefaultCLIConfigPaths() {
			if loaded, err := config.LoadCLIConfig(p); err == nil {
				cfg = loaded
				break
			}
		}
	}

	// Env var overrides
	if v := os.Getenv("JUMPGATE_API_URL"); v != "" {
		cfg.URL = v
	}
	if v := os.Getenv("JUMPGATE_API_TOKEN"); v != "" {
		cfg.Token = v
	}

	return cfg
}
