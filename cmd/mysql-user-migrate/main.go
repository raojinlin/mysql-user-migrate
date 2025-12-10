package main

import (
	"context"
	"io"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/raojinlin/mysql-user-migrate/internal/cli"
	"github.com/raojinlin/mysql-user-migrate/internal/config"
	"github.com/raojinlin/mysql-user-migrate/internal/migrate"
)

func main() {
	opts, err := cli.ParseOptions(os.Args[1:])
	if err != nil {
		log.Fatalf("parse flags: %v", err)
	}

	fileCfg, err := config.Load(opts.ConfigPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	merged := config.Merge(fileCfg, opts.Config)
	applyEnvDefaults(&merged)

	if err := merged.Validate(); err != nil {
		log.Fatalf("config: %v", err)
	}

	logger := log.New(io.Discard, "", log.LstdFlags)
	if merged.Verbose {
		logger = log.New(os.Stdout, "[mysql-user-migrate] ", log.LstdFlags)
	}

	runner := migrate.Runner{
		SourceDSN:      merged.Source,
		Targets:        merged.Targets,
		Include:        merged.Include,
		Exclude:        merged.Exclude,
		DryRun:         merged.DryRun,
		DropMissing:    merged.DropMissing,
		ForceOverwrite: merged.ForceOverwrite,
		Concurrency:    merged.Concurrency,
		Logger:         logger,
	}

	ctx := context.Background()
	report, err := runner.Run(ctx)
	if err != nil {
		log.Fatalf("migrate: %v", err)
	}

	report.Print(os.Stdout)
	if merged.ReportPath != "" {
		if err := report.WriteJSON(merged.ReportPath); err != nil {
			log.Printf("write report: %v", err)
		}
	}
}

func applyEnvDefaults(cfg *config.RuntimeConfig) {
	if cfg.Source == "" {
		if v := os.Getenv("SOURCE_DSN"); v != "" {
			cfg.Source = v
		}
	}
	if len(cfg.Targets) == 0 {
		if v := os.Getenv("TARGET_DSN"); v != "" {
			cfg.Targets = append(cfg.Targets, config.Target{Name: "target-1", DSN: v})
		} else if v := os.Getenv("TARGET_DSN_LIST"); v != "" {
			values := strings.Split(v, ",")
			for idx, raw := range values {
				raw = strings.TrimSpace(raw)
				if raw == "" {
					continue
				}
				cfg.Targets = append(cfg.Targets, config.Target{
					Name: targetName(idx, raw),
					DSN:  raw,
				})
			}
		}
	}
	if cfg.Concurrency <= 0 {
		cfg.Concurrency = 1
	}
}

func targetName(idx int, dsn string) string {
	host := dsn
	if at := strings.Index(dsn, "@"); at >= 0 && at+1 < len(dsn) {
		host = dsn[at+1:]
	}
	if host == "" {
		return "target-" + strconv.Itoa(idx+1)
	}
	return host
}
