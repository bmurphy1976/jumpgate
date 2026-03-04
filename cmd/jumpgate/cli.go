package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	"dashboard/config"
	"dashboard/storage"

	"gopkg.in/yaml.v3"
)

func runImport(args []string) {
	fs := flag.NewFlagSet("import", flag.ExitOnError)
	dbPath := fs.String("db", envOr("DB_FILE", defaultDBPath), "path to SQLite database")
	force := fs.Bool("force", false, "overwrite existing database")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: jumpgate import [--force] [--db PATH] config.yaml")
		fs.PrintDefaults()
	}
	fs.Parse(args)
	if fs.NArg() < 1 {
		fs.Usage()
		os.Exit(1)
	}
	if err := doImport(*dbPath, fs.Arg(0), *force); err != nil {
		slog.Error("import failed", "error", err)
		os.Exit(1)
	}
}

func runExport(args []string) {
	fs := flag.NewFlagSet("export", flag.ExitOnError)
	dbPath := fs.String("db", envOr("DB_FILE", defaultDBPath), "path to SQLite database")
	force := fs.Bool("force", false, "overwrite existing output file")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: jumpgate export [--force] [--db PATH] [output.yaml]")
		fs.PrintDefaults()
	}
	fs.Parse(args)
	outFile := ""
	if fs.NArg() > 0 {
		outFile = fs.Arg(0)
	}
	if err := doExport(*dbPath, outFile, *force); err != nil {
		slog.Error("export failed", "error", err)
		os.Exit(1)
	}
}

func doImport(dbPath, configPath string, force bool) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if _, err := os.Stat(dbPath); err == nil && !force {
		return fmt.Errorf("database already exists: %s (use --force to overwrite)", dbPath)
	}

	ds, err := storage.NewSQLiteDB(dbPath)
	if err != nil {
		return err
	}
	defer ds.Close()

	if err := ds.ImportConfig(cfg); err != nil {
		return err
	}

	bookmarkCount := 0
	for _, cat := range cfg.Categories {
		bookmarkCount += len(cat.Links)
	}
	fmt.Printf("Imported %d categories, %d bookmarks from %s\n", len(cfg.Categories), bookmarkCount, configPath)
	return nil
}

func doExport(dbPath, outFile string, force bool) error {
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return fmt.Errorf("database not found: %s", dbPath)
	}

	ds, err := storage.NewSQLiteDB(dbPath)
	if err != nil {
		return err
	}
	defer ds.Close()

	cfg, err := ds.ExportConfig()
	if err != nil {
		return err
	}

	out, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal yaml: %w", err)
	}

	if outFile == "" {
		_, err = os.Stdout.Write(out)
		return err
	}
	if _, err := os.Stat(outFile); err == nil && !force {
		return fmt.Errorf("file already exists: %s (use --force to overwrite)", outFile)
	}
	if err := os.WriteFile(outFile, out, 0644); err != nil {
		return fmt.Errorf("write %s: %w", outFile, err)
	}
	fmt.Printf("Exported to %s\n", outFile)
	return nil
}
