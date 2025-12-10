package migrate

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/raojinlin/mysql-user-migrate/internal/config"

	_ "github.com/go-sql-driver/mysql" // register MySQL driver
)

// Runner orchestrates migrations across targets.
type Runner struct {
	SourceDSN      string
	Targets        []config.Target
	Include        []string
	Exclude        []string
	DryRun         bool
	DropMissing    bool
	ForceOverwrite bool
	Concurrency    int
	Logger         *log.Logger
}

// Run executes the migration across all targets.
func (r *Runner) Run(ctx context.Context) (*Report, error) {
	if r.Concurrency <= 0 {
		r.Concurrency = 1
	}
	if r.Logger == nil {
		r.Logger = log.New(log.Writer(), "", log.LstdFlags)
	}

	srcDB, err := openDB(ctx, r.SourceDSN)
	if err != nil {
		return nil, fmt.Errorf("connect source: %w", err)
	}
	defer srcDB.Close()

	sourceUsers, err := r.loadSourceUsers(ctx, srcDB)
	if err != nil {
		return nil, fmt.Errorf("load source users: %w", err)
	}
	r.Logger.Printf("loaded %d users from source", len(sourceUsers))

	report := &Report{
		Source:    MaskDSN(r.SourceDSN),
		DryRun:    r.DryRun,
		StartedAt: time.Now(),
	}

	sem := make(chan struct{}, r.Concurrency)
	var wg sync.WaitGroup
	results := make(chan TargetReport, len(r.Targets))

	for _, target := range r.Targets {
		wg.Add(1)
		go func(t config.Target) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			results <- r.migrateTarget(ctx, sourceUsers, t)
		}(target)
	}

	wg.Wait()
	close(results)

	for res := range results {
		report.Targets = append(report.Targets, res)
		report.TotalFailed += res.Failed
		report.TotalUsers += len(res.Users)
	}
	report.FinishedAt = time.Now()
	return report, nil
}

func (r *Runner) migrateTarget(ctx context.Context, users []UserRecord, target config.Target) TargetReport {
	start := time.Now()
	targetName := target.Name
	if targetName == "" {
		targetName = MaskDSN(target.DSN)
	}
	result := TargetReport{
		Target:    targetName,
		DryRun:    r.DryRun,
		StartedAt: start,
	}

	db, err := openDB(ctx, target.DSN)
	if err != nil {
		result.Error = fmt.Sprintf("connect target: %v", err)
		result.Failed = len(users)
		result.FinishedAt = time.Now()
		result.DurationMS = result.FinishedAt.Sub(result.StartedAt).Milliseconds()
		return result
	}
	defer db.Close()

	for _, user := range users {
		userResult := r.applyUser(ctx, db, user)
		result.Users = append(result.Users, userResult)
		switch userResult.Status {
		case "applied", "planned":
			result.Applied++
		case "skipped":
			result.Skipped++
		case "error":
			result.Failed++
		default:
			result.Failed++
		}
	}

	result.FinishedAt = time.Now()
	result.DurationMS = result.FinishedAt.Sub(result.StartedAt).Milliseconds()
	return result
}

func (r *Runner) applyUser(ctx context.Context, db *sql.DB, user UserRecord) UserResult {
	identity := fmt.Sprintf("%s@%s", user.User, user.Host)
	out := UserResult{User: user.User, Host: user.Host}

	exists, err := userExists(ctx, db, user.User, user.Host)
	if err != nil {
		out.Status = "error"
		out.Error = fmt.Sprintf("check exists: %v", err)
		return out
	}

	if r.DryRun {
		if exists && !(r.DropMissing || r.ForceOverwrite) {
			out.Status = "planned"
		} else {
			out.Status = "planned"
		}
		return out
	}

	if exists && (r.DropMissing || r.ForceOverwrite) {
		if err := dropUser(ctx, db, user); err != nil {
			out.Status = "error"
			out.Error = fmt.Sprintf("drop %s: %v", identity, err)
			return out
		}
		exists = false
	}

	if !exists {
		if err := createUser(ctx, db, user); err != nil {
			out.Status = "error"
			out.Error = fmt.Sprintf("create %s: %v", identity, err)
			return out
		}
	}

	for _, grant := range user.Grants {
		if err := applyGrant(ctx, db, grant); err != nil {
			out.Status = "error"
			out.Error = fmt.Sprintf("grant %s: %v", identity, err)
			return out
		}
	}

	out.Status = "applied"
	return out
}

func (r *Runner) loadSourceUsers(ctx context.Context, db *sql.DB) ([]UserRecord, error) {
	rows, err := db.QueryContext(ctx, `SELECT user, host, plugin, authentication_string FROM mysql.user`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []UserRecord
	for rows.Next() {
		var user, host, plugin, auth string
		if err := rows.Scan(&user, &host, &plugin, &auth); err != nil {
			return nil, err
		}

		if !ShouldInclude(user, host, r.Include, r.Exclude) {
			continue
		}

		grants, err := fetchGrants(ctx, db, user, host)
		if err != nil {
			return nil, fmt.Errorf("grants for %s@%s: %w", user, host, err)
		}

		users = append(users, UserRecord{
			User:        user,
			Host:        host,
			Plugin:      plugin,
			AuthString:  auth,
			Grants:      grants,
			RawIdentity: fmt.Sprintf("%s@%s", user, host),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return nil, errors.New("no users matched include/exclude filters")
	}
	return users, nil
}

func fetchGrants(ctx context.Context, db *sql.DB, user, host string) ([]string, error) {
	// MySQL does not permit parameter placeholders in SHOW GRANTS.
	stmt := fmt.Sprintf("SHOW GRANTS FOR '%s'@'%s'", escape(user), escape(host))
	rows, err := db.QueryContext(ctx, stmt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var grants []string
	for rows.Next() {
		var grant string
		if err := rows.Scan(&grant); err != nil {
			return nil, err
		}
		grants = append(grants, grant)
	}
	return grants, rows.Err()
}

func userExists(ctx context.Context, db *sql.DB, user, host string) (bool, error) {
	var count int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM mysql.user WHERE user=? AND host=?", user, host).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

func dropUser(ctx context.Context, db *sql.DB, user UserRecord) error {
	stmt := fmt.Sprintf("DROP USER IF EXISTS '%s'@'%s'", escape(user.User), escape(user.Host))
	_, err := db.ExecContext(ctx, stmt)
	return err
}

func createUser(ctx context.Context, db *sql.DB, user UserRecord) error {
	stmt := fmt.Sprintf("CREATE USER IF NOT EXISTS '%s'@'%s'", escape(user.User), escape(user.Host))
	if user.Plugin != "" && user.AuthString != "" {
		stmt = fmt.Sprintf("%s IDENTIFIED WITH '%s' AS '%s'", stmt, escape(user.Plugin), escape(user.AuthString))
	} else if user.AuthString != "" {
		stmt = fmt.Sprintf("%s IDENTIFIED BY PASSWORD '%s'", stmt, escape(user.AuthString))
	}
	_, err := db.ExecContext(ctx, stmt)
	return err
}

func applyGrant(ctx context.Context, db *sql.DB, grant string) error {
	_, err := db.ExecContext(ctx, grant)
	return err
}

func escape(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `'`, `''`)
	return value
}

func ShouldInclude(user, host string, include, exclude []string) bool {
	u := strings.ToLower(user)
	h := strings.ToLower(host)
	for _, ex := range exclude {
		if MatchIdentity(u, h, ex) {
			return false
		}
	}
	if len(include) == 0 {
		return true
	}
	for _, inc := range include {
		if MatchIdentity(u, h, inc) {
			return true
		}
	}
	return false
}

func openDB(ctx context.Context, dsn string) (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func MatchIdentity(user, host, pattern string) bool {
	pattern = strings.TrimSpace(strings.ToLower(pattern))
	if pattern == "" {
		return false
	}
	userPat := pattern
	hostPat := ""
	if strings.Contains(pattern, "@") {
		parts := strings.SplitN(pattern, "@", 2)
		userPat = parts[0]
		hostPat = parts[1]
	}
	if !matchGlob(user, userPat) {
		return false
	}
	if hostPat == "" {
		return true
	}
	return matchGlob(host, hostPat)
}

func matchGlob(value, pattern string) bool {
	pattern = normalizePattern(pattern)
	if pattern == "" {
		return false
	}
	if pattern == "*" {
		return true
	}
	ok, err := path.Match(pattern, value)
	if err != nil {
		return pattern == value
	}
	return ok
}

func normalizePattern(p string) string {
	p = strings.ReplaceAll(p, "%", "*")
	for strings.Contains(p, "**") {
		p = strings.ReplaceAll(p, "**", "*")
	}
	return p
}

// maskDSN redacts password component in DSN for safe logging/reporting.
func MaskDSN(dsn string) string {
	cfg, err := mysql.ParseDSN(dsn)
	if err == nil {
		cfg.Passwd = "****"
		return cfg.FormatDSN()
	}

	// Fallback: naive masking user:pass@...
	at := strings.Index(dsn, "@")
	if at == -1 {
		return dsn
	}
	before := dsn[:at]
	if colon := strings.Index(before, ":"); colon != -1 {
		before = before[:colon] + ":****"
	}
	return before + dsn[at:]
}
