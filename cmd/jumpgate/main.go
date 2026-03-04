package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	"dashboard/config"
	"dashboard/handlers"
	"dashboard/icons"
	"dashboard/storage"
)

const defaultDBPath = "data/jumpgate.db"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}
	switch os.Args[1] {
	case "server":
		runServer(os.Args[2:])
	case "import":
		runImport(os.Args[2:])
	case "export":
		runExport(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "Usage:")
	fmt.Fprintln(os.Stderr, "  jumpgate server [--config PATH]")
	fmt.Fprintln(os.Stderr, "  jumpgate import [--force] [--db PATH] config.yaml")
	fmt.Fprintln(os.Stderr, "  jumpgate export [--force] [--db PATH] [output.yaml]")
}

func runServer(args []string) {
	fs := flag.NewFlagSet("server", flag.ExitOnError)
	configPath := fs.String("config", "", "path to jumpgate.yaml config file")
	fs.Parse(args)

	cfg := loadServerConfig(*configPath)

	il, err := icons.New()
	if err != nil {
		slog.Warn("failed to load icons", "error", err)
	}

	wrap := func(ds storage.Datasource) storage.Datasource {
		return storage.Annotated(ds)
	}

	var (
		resolver handlers.DSResolver
		store    *storage.SessionStore
	)

	if cfg.Demo.Enabled {
		bookmarksCfg, err := config.Load(cfg.Demo.Source)
		if err != nil {
			slog.Error("failed to load demo config", "error", err)
			os.Exit(1)
		}

		store = storage.NewSessionStore(bookmarksCfg, wrap)
		resolver = handlers.SessionResolver(store)

		f := false
		cfg.Auth = &f
		slog.Info("server starting (demo mode)", "addr", cfg.Addr, "source", cfg.Demo.Source)
	} else {
		ds, err := storage.NewSQLiteDB(cfg.DB)
		if err != nil {
			slog.Error("failed to open database", "error", err)
			os.Exit(1)
		}
		defer ds.Close()

		resolver = handlers.StaticResolver(wrap(ds))

		slog.Info("server starting", "addr", cfg.Addr)
	}

	apiEnabled := cfg.API.Tokens.HasTokens() || cfg.API.Swagger
	slog.Info("features",
		"auth", cfg.AuthEnabled(),
		"api", apiEnabled,
		"swagger", cfg.API.Swagger,
		"mcp", cfg.MCP.Enabled,
		"slow", cfg.Slow > 0,
	)

	srv := handlers.NewServer(cfg, resolver, il, store)

	if err := srv.Start(cfg.Addr); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// loadServerConfig loads config from the given path, or returns defaults.
func loadServerConfig(path string) config.ServerConfig {
	if path == "" {
		var cfg config.ServerConfig
		cfg.ApplyDefaults()
		return cfg
	}
	cfg, err := config.LoadServerConfig(path)
	if err != nil {
		slog.Error("failed to load config", "path", path, "error", err)
		os.Exit(1)
	}
	return cfg
}
