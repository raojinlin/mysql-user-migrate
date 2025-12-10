package migrate

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"
)

// UserRecord holds source-side user information.
type UserRecord struct {
	User        string
	Host        string
	Plugin      string
	AuthString  string
	Grants      []string
	RawIdentity string
}

// UserResult captures the outcome per user on a target.
type UserResult struct {
	User   string `json:"user"`
	Host   string `json:"host"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

// TargetReport summarizes migration to a single target.
type TargetReport struct {
	Target     string       `json:"target"`
	Applied    int          `json:"applied"`
	Skipped    int          `json:"skipped"`
	Failed     int          `json:"failed"`
	Users      []UserResult `json:"users"`
	Error      string       `json:"error,omitempty"`
	DurationMS int64        `json:"duration_ms"`
	DryRun     bool         `json:"dry_run"`
	StartedAt  time.Time    `json:"started_at"`
	FinishedAt time.Time    `json:"finished_at"`
}

// Report aggregates all target reports.
type Report struct {
	Source      string         `json:"source"`
	DryRun      bool           `json:"dry_run"`
	StartedAt   time.Time      `json:"started_at"`
	FinishedAt  time.Time      `json:"finished_at"`
	Targets     []TargetReport `json:"targets"`
	TotalFailed int            `json:"total_failed"`
	TotalUsers  int            `json:"total_users"`
}

// WriteJSON writes the report to a file path.
func (r *Report) WriteJSON(path string) error {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal report: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write report: %w", err)
	}
	return nil
}

// Print renders a concise text summary.
func (r *Report) Print(w io.Writer) {
	fmt.Fprintf(w, "Migration report (dry-run=%v)\n", r.DryRun)
	fmt.Fprintf(w, "Source: %s\n", r.Source)
	fmt.Fprintf(w, "Targets: %d | Duration: %s\n", len(r.Targets), r.FinishedAt.Sub(r.StartedAt).Round(time.Millisecond))
	for _, t := range r.Targets {
		fmt.Fprintf(w, "- %s | applied=%d skipped=%d failed=%d | duration=%s\n", t.Target, t.Applied, t.Skipped, t.Failed, time.Duration(t.DurationMS)*time.Millisecond)
		if t.Error != "" {
			fmt.Fprintf(w, "  error: %s\n", t.Error)
			continue
		}
		for _, u := range t.Users {
			if u.Error != "" {
				fmt.Fprintf(w, "  %s@%s -> %s (%s)\n", u.User, u.Host, u.Status, u.Error)
			} else {
				fmt.Fprintf(w, "  %s@%s -> %s\n", u.User, u.Host, u.Status)
			}
		}
	}
}
