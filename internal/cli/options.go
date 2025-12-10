package cli

import (
	"flag"
	"fmt"
	"strconv"
	"strings"

	"github.com/raojinlin/mysql-user-migrate/internal/config"
)

// Options parses and holds CLI-provided configuration.
type Options struct {
	ConfigPath string
	Config     config.CLIConfig
}

// ParseOptions parses command-line flags into Options.
func ParseOptions(args []string) (Options, error) {
	var (
		configPath string
		sourceDSN  string
		targets    stringListFlag
		include    stringListFlag
		exclude    stringListFlag
		reportPath string

		dryRunFlag         boolFlag
		dropMissingFlag    boolFlag
		forceOverwriteFlag boolFlag
		verboseFlag        boolFlag
		concurrencyFlag    intFlag
	)

	fs := flag.NewFlagSet("mysql-user-migrate", flag.ContinueOnError)
	fs.StringVar(&configPath, "config", "", "Path to YAML/JSON config file")
	fs.StringVar(&sourceDSN, "source", "", "Source MySQL DSN (e.g., user:pass@tcp(host:3306)/)")
	fs.Var(&targets, "target", "Target MySQL DSN; repeatable (name=dsn supported)")
	fs.Var(&include, "include", "Comma-separated list of users or user@host to include")
	fs.Var(&exclude, "exclude", "Comma-separated list of users or user@host to exclude")
	fs.StringVar(&reportPath, "report", "", "Path to write report (JSON)")
	fs.Var(&dryRunFlag, "dry-run", "Plan only; do not apply changes")
	fs.Var(&dropMissingFlag, "drop-missing", "Drop/replace target users to match source (cleans extra grants)")
	fs.Var(&forceOverwriteFlag, "force-overwrite", "Force reset of existing users (drop and recreate)")
	fs.Var(&verboseFlag, "verbose", "Verbose logs")
	fs.Var(&concurrencyFlag, "concurrency", "Number of targets to migrate concurrently")

	if err := fs.Parse(args); err != nil {
		return Options{}, err
	}

	cfg := config.CLIConfig{
		Source:         sourceDSN,
		Targets:        parseTargets(targets.values),
		Include:        include.values,
		Exclude:        exclude.values,
		ReportPath:     reportPath,
		DryRun:         boolPtr(dryRunFlag),
		DropMissing:    boolPtr(dropMissingFlag),
		ForceOverwrite: boolPtr(forceOverwriteFlag),
		Verbose:        boolPtr(verboseFlag),
		Concurrency:    intPtr(concurrencyFlag),
	}

	return Options{
		ConfigPath: configPath,
		Config:     cfg,
	}, nil
}

type stringListFlag struct {
	values []string
}

func (s *stringListFlag) String() string {
	return strings.Join(s.values, ",")
}

func (s *stringListFlag) Set(v string) error {
	parts := strings.Split(v, ",")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		s.values = append(s.values, p)
	}
	return nil
}

type boolFlag struct {
	set   bool
	value bool
}

func (b *boolFlag) String() string {
	return strconv.FormatBool(b.value)
}

func (b *boolFlag) Set(v string) error {
	parsed, err := strconv.ParseBool(v)
	if err != nil {
		return err
	}
	b.set = true
	b.value = parsed
	return nil
}

type intFlag struct {
	set   bool
	value int
}

func (i *intFlag) String() string {
	return strconv.Itoa(i.value)
}

func (i *intFlag) Set(v string) error {
	parsed, err := strconv.Atoi(v)
	if err != nil {
		return err
	}
	i.set = true
	i.value = parsed
	return nil
}

func boolPtr(flag boolFlag) *bool {
	if !flag.set {
		return nil
	}
	return &flag.value
}

func intPtr(flag intFlag) *int {
	if !flag.set {
		return nil
	}
	return &flag.value
}

// parseTargets builds named targets. Accepts "name=dsn" or bare "dsn".
func parseTargets(values []string) []config.Target {
	targets := make([]config.Target, 0, len(values))
	for idx, raw := range values {
		name := fmt.Sprintf("target-%d", idx+1)
		dsn := raw
		if parts := strings.SplitN(raw, "=", 2); len(parts) == 2 {
			name, dsn = strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
		}
		targets = append(targets, config.Target{
			Name: name,
			DSN:  dsn,
		})
	}
	return targets
}
