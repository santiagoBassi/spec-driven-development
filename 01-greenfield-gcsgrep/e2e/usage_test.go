package e2e

import (
	"strconv"
	"testing"
)

// FR-5
func TestVC05(t *testing.T) {
	requireUsageError(t, prefix("gcsgrep: invalid pattern: "), "-E", "(", gs(t, "logs/"))
}

// FR-6
func TestVC06(t *testing.T) {
	logs := gs(t, "logs/")
	for name, args := range map[string][]string{
		"empty literal": {"", logs},
		"empty regex":   {"-E", "", logs},
		"after --":      {"--", "", logs},
	} {
		t.Run(name, func(t *testing.T) {
			requireUsageError(t, exact("gcsgrep: empty pattern"), args...)
		})
	}
}

// FR-14
func TestVC14(t *testing.T) {
	b := bucket(t)
	for _, loc := range []string{"logs/", "s3://" + b + "/", "gs://", "gs:///logs/", "gs://" + b} {
		t.Run(loc, func(t *testing.T) {
			requireUsageError(t, exact(`gcsgrep: invalid location: "`+loc+`"`), "timeout", loc)
		})
	}
}

// FR-22
func TestVC22(t *testing.T) {
	t.Run("-1]", func(t *testing.T) {
		requireUsageError(t, exact(`gcsgrep: unknown flag: "-1]"`), "-1]", gs(t, "logs/db/postgres.log"))
	})
	t.Run("-v", func(t *testing.T) {
		requireUsageError(t, exact(`gcsgrep: unknown flag: "-v"`), "-v", "timeout", gs(t, "logs/"))
	})
}

// FR-23
func TestVC23(t *testing.T) {
	logs := gs(t, "logs/")
	for _, tc := range []struct {
		args []string
		n    int
	}{
		{nil, 0},
		{[]string{"timeout"}, 1},
		{[]string{"timeout", logs, "extra"}, 3},
	} {
		t.Run(strconv.Itoa(tc.n)+" arguments", func(t *testing.T) {
			requireUsageError(t,
				exact("gcsgrep: expected 2 arguments (pattern and location), got "+strconv.Itoa(tc.n)),
				tc.args...)
		})
	}
}
