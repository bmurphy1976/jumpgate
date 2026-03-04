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
	fmt.Fprintln(os.Stderr, "  jumpgate server [--db PATH] [--addr ADDR] [--no-auth] [--slow]")
	fmt.Fprintln(os.Stderr, "  jumpgate server --demo config.yaml [--addr ADDR] [--slow]")
	fmt.Fprintln(os.Stderr, "  jumpgate import [--force] [--db PATH] config.yaml")
	fmt.Fprintln(os.Stderr, "  jumpgate export [--force] [--db PATH] [output.yaml]")
}

func runServer(args []string) {
	fs := flag.NewFlagSet("server", flag.ExitOnError)
	dbPath := fs.String("db", envOr("DB_FILE", defaultDBPath), "path to SQLite database")
	addr := fs.String("addr", envOr("LISTEN_ADDR", ":8080"), "listen address")
	noAuth := fs.Bool("no-auth", false, "disable admin authorization check")
	demo := fs.String("demo", "", "path to YAML config for demo mode (in-memory, per-session)")
	slow := fs.Bool("slow", false, "add 2s delay to every request (latency testing)")
	fs.Parse(args)

	if *demo != "" && *dbPath != envOr("DB_FILE", defaultDBPath) {
		fmt.Fprintln(os.Stderr, "error: --demo and --db are mutually exclusive")
		os.Exit(1)
	}

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
		demoMode bool
	)

	if *demo != "" {
		cfg, err := config.Load(*demo)
		if err != nil {
			slog.Error("failed to load demo config", "error", err)
			os.Exit(1)
		}

		store = storage.NewSessionStore(cfg, wrap)
		resolver = handlers.SessionResolver(store)

		demoMode = true
		*noAuth = true
		slog.Info("server starting (demo mode)", "addr", *addr, "config", *demo)
	} else {
		ds, err := storage.NewSQLiteDB(*dbPath)
		if err != nil {
			slog.Error("failed to open database", "error", err)
			os.Exit(1)
		}
		defer ds.Close()

		resolver = handlers.StaticResolver(wrap(ds))
		
		slog.Info("server starting", "addr", *addr)
	}

	if err := handlers.NewServer(resolver, il, *noAuth, demoMode, store, *slow).Start(*addr); err != nil {
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
