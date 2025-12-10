package migrate

import "testing"

func TestMatchIdentity(t *testing.T) {
	tests := []struct {
		name    string
		user    string
		host    string
		pattern string
		want    bool
	}{
		{"wildcard user", "mysql.sys", "localhost", "mysql.*", true},
		{"user and host glob match", "app", "10.0.5.3", "app@10.0.%", true},
		{"host glob mismatch", "app", "10.1.1.1", "app@10.0.%", false},
		{"user only pattern", "root", "anywhere", "root", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchIdentity(tt.user, tt.host, tt.pattern)
			if got != tt.want {
				t.Fatalf("matchIdentity(%s, %s, %s) = %v, want %v", tt.user, tt.host, tt.pattern, got, tt.want)
			}
		})
	}
}

func TestShouldInclude(t *testing.T) {
	tests := []struct {
		name    string
		user    string
		host    string
		include []string
		exclude []string
		want    bool
	}{
		{"no filters", "app", "10.0.0.1", nil, nil, true},
		{"include matches", "app", "10.0.1.2", []string{"app@10.0.%"},
			nil, true},
		{"include does not match host", "app", "10.1.1.1", []string{"app@10.0.%"},
			nil, false},
		{"exclude wildcard", "mysql.sys", "localhost", nil, []string{"mysql.*"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldInclude(tt.user, tt.host, tt.include, tt.exclude)
			if got != tt.want {
				t.Fatalf("shouldInclude(%s, %s) = %v, want %v", tt.user, tt.host, got, tt.want)
			}
		})
	}
}

func TestMaskDSN(t *testing.T) {
	tests := []struct {
		name string
		dsn  string
		want string
	}{
		{"standard DSN", "user:secret@tcp(localhost:3306)/", "user:****@tcp(localhost:3306)/"},
		{"fallback masking", "user:secret@localhost", "user:****@localhost"},
		{"no at symbol", "localhost", "localhost"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := maskDSN(tt.dsn); got != tt.want {
				t.Fatalf("maskDSN(%s) = %s, want %s", tt.dsn, got, tt.want)
			}
		})
	}
}
