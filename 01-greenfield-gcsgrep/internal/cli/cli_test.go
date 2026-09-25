package cli

import (
	"errors"
	"math"
	"strings"
	"testing"

	"gcsgrep/internal/location"
)

func parseErr(t *testing.T, args ...string) string {
	t.Helper()
	cfg, err := Parse(args)
	if err == nil {
		t.Fatalf("Parse(%q) accepted the invocation: %+v", args, cfg)
	}
	var ue *UsageError
	if !errors.As(err, &ue) {
		t.Fatalf("Parse(%q) error is not a UsageError: %v", args, err)
	}
	return err.Error()
}

func TestParseDefaults(t *testing.T) {
	cfg, err := Parse([]string{"timeout", "gs://b/logs/"})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.LineNumbers || cfg.Max != 1000 {
		t.Errorf("defaults: %+v", cfg)
	}
	if cfg.Location.Bucket != "b" || cfg.Location.Path != "logs/" || cfg.Location.Mode != location.Prefix {
		t.Errorf("location: %+v", cfg.Location)
	}
	if !cfg.Matcher.MatchString("a timeout b") || cfg.Matcher.MatchString("TIMEOUT") {
		t.Error("default matcher is not a case-sensitive literal")
	}
}

func TestParseFlags(t *testing.T) {
	cfg, err := Parse([]string{"-E", "-i", "-n", "--max", "5", `job_id=\d+`, "gs://b/x"})
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.LineNumbers || cfg.Max != 5 {
		t.Errorf("cfg = %+v", cfg)
	}
	if !cfg.Matcher.MatchString("JOB_ID=42") {
		t.Error("-E -i did not give a case-insensitive regex")
	}
}

func TestParseFlagsAfterPositionals(t *testing.T) {
	cfg, err := Parse([]string{"timeout", "-n", "gs://b/x", "-i"})
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.LineNumbers || !cfg.Matcher.MatchString("TIMEOUT") {
		t.Errorf("flags after positionals were not honored: %+v", cfg)
	}
}

func TestParseDoubleDash(t *testing.T) {
	cfg, err := Parse([]string{"--", "-1]", "gs://b/logs/db/postgres.log"})
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Matcher.MatchString("x -1] y") {
		t.Error("pattern after -- was not taken literally")
	}

	// Everything after "--" is positional, even a location that starts with "-".
	if got := parseErr(t, "--", "a", "b", "-n"); got != "expected 2 arguments (pattern and location), got 3" {
		t.Errorf("got %q", got)
	}
	// The pattern can be a flag-looking string and the flags still work before --.
	if _, err := Parse([]string{"-n", "--", "-E", "gs://b/x"}); err != nil {
		t.Errorf("-n -- -E gs://b/x: %v", err)
	}
}

func TestParseUsageErrors(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"unknown flag", []string{"-1]", "gs://b/x"}, `unknown flag: "-1]"`},
		{"unsupported flag -v", []string{"-v", "timeout", "gs://b/"}, `unknown flag: "-v"`},
		{"lone dash", []string{"-", "gs://b/x"}, `unknown flag: "-"`},
		{"-c is not implemented yet", []string{"-c", "timeout", "gs://b/"}, `unknown flag: "-c"`},
		{"-l is not implemented yet", []string{"-l", "timeout", "gs://b/"}, `unknown flag: "-l"`},
		{"--concurrency is not implemented yet", []string{"--concurrency", "2", "timeout", "gs://b/"}, `unknown flag: "--concurrency"`},
		{"no arguments", nil, "expected 2 arguments (pattern and location), got 0"},
		{"one argument", []string{"timeout"}, "expected 2 arguments (pattern and location), got 1"},
		{"three arguments", []string{"timeout", "gs://b/", "extra"}, "expected 2 arguments (pattern and location), got 3"},
		{"empty pattern", []string{"", "gs://b/"}, "empty pattern"},
		{"empty pattern with -E", []string{"-E", "", "gs://b/"}, "empty pattern"},
		{"empty pattern after --", []string{"--", "", "gs://b/"}, "empty pattern"},
		{"--max without value", []string{"timeout", "gs://b/", "--max"}, "flag --max requires a value"},
		{"location without scheme", []string{"timeout", "logs/"}, `invalid location: "logs/"`},
		{"location without slash", []string{"timeout", "gs://b"}, `invalid location: "gs://b"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseErr(t, tt.args...); got != tt.want {
				t.Errorf("error = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseInvalidRegex(t *testing.T) {
	got := parseErr(t, "-E", "(", "gs://b/logs/")
	if !strings.HasPrefix(got, "invalid pattern: ") {
		t.Errorf("error = %q", got)
	}
	// Without -E the same pattern is a literal and is fine.
	if _, err := Parse([]string{"(", "gs://b/logs/"}); err != nil {
		t.Errorf("literal ( rejected: %v", err)
	}
}

func TestParseMax(t *testing.T) {
	valid := map[string]int{
		"1":         1,
		"5":         5,
		"1000":      1000,
		"007":       7,
		"unlimited": 0,
		// Larger than any int: still an integer >= 1.
		"99999999999999999999999": math.MaxInt,
	}
	for v, want := range valid {
		cfg, err := Parse([]string{"--max", v, "timeout", "gs://b/"})
		if err != nil {
			t.Errorf("--max %s: %v", v, err)
			continue
		}
		if cfg.Max != want {
			t.Errorf("--max %s: Max = %d, want %d", v, cfg.Max, want)
		}
	}

	for _, v := range []string{"0", "00", "-3", "abc", "1.5", "", "+5", "5x", "Unlimited", " 5", "--"} {
		want := `invalid value for --max: "` + v + `" (integer >= 1 or unlimited)`
		if got := parseErr(t, "--max", v, "timeout", "gs://b/"); got != want {
			t.Errorf("--max %q: error = %q, want %q", v, got, want)
		}
	}
}

func TestParseReportsOneError(t *testing.T) {
	for _, args := range [][]string{
		{"-E", "(", "s3://x/"},
		{"-c", "-l", "--max", "0", "", "gs://b"},
		{"-x", "--max", "0", "", "s3://x/", "extra"},
	} {
		got := parseErr(t, args...)
		if strings.Contains(got, "\n") || got == "" {
			t.Errorf("Parse(%q) error = %q, want exactly one line", args, got)
		}
	}
}
