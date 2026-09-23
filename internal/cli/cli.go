// Package cli parses argv and validates an invocation. It never touches GCS:
// everything that argv alone can decide is decided here, before authenticating.
package cli

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"gcsgrep/internal/location"
	"gcsgrep/internal/search"
)

// DefaultMax is the object cap when --max is not given (BR-3).
const DefaultMax = 1000

// Config is a valid invocation.
type Config struct {
	LineNumbers bool // -n
	// Max is the object cap; 0 means unlimited (--max unlimited).
	Max      int
	Matcher  *regexp.Regexp
	Location location.Location
}

// UsageError is an invocation error: exit code 2, and no operation against GCS.
// Its message does not include the "gcsgrep: " prefix.
type UsageError struct{ Msg string }

func (e *UsageError) Error() string { return e.Msg }

func usagef(format string, args ...any) error {
	return &UsageError{Msg: fmt.Sprintf(format, args...)}
}

// Parse validates args (without the program name) and reports the first usage
// error found. Which one is reported when there are several is not specified
// (FR-27).
//
// Every argument before "--" that starts with "-" is a flag. Flags this
// iteration does not know yet, including -c, -l and --concurrency, are unknown.
func Parse(args []string) (*Config, error) {
	cfg := &Config{Max: DefaultMax}
	var (
		regex      bool
		ignoreCase bool
		pos        []string
	)

parse:
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--":
			pos = append(pos, args[i+1:]...)
			break parse
		case !strings.HasPrefix(a, "-"):
			pos = append(pos, a)
			continue
		}

		switch a {
		case "-E":
			regex = true
		case "-i":
			ignoreCase = true
		case "-n":
			cfg.LineNumbers = true
		case "--max":
			// The value is always the next argument, whatever it looks like.
			if i+1 == len(args) {
				return nil, usagef("flag --max requires a value")
			}
			i++
			n, err := parseMax(args[i])
			if err != nil {
				return nil, err
			}
			cfg.Max = n
		default:
			return nil, usagef(`unknown flag: "%s"`, a)
		}
	}

	if len(pos) != 2 {
		return nil, usagef("expected 2 arguments (pattern and location), got %d", len(pos))
	}
	pattern, rawLocation := pos[0], pos[1]

	if pattern == "" {
		return nil, usagef("empty pattern")
	}
	re, err := search.Compile(pattern, regex, ignoreCase)
	if err != nil {
		return nil, usagef("invalid pattern: %v", err)
	}
	cfg.Matcher = re

	loc, err := location.Parse(rawLocation)
	if err != nil {
		return nil, &UsageError{Msg: err.Error()}
	}
	cfg.Location = loc
	return cfg, nil
}

// parseMax accepts an integer >= 1, or "unlimited" (0).
func parseMax(v string) (int, error) {
	invalid := usagef(`invalid value for --max: "%s" (integer >= 1 or unlimited)`, v)
	if v == "unlimited" {
		return 0, nil
	}
	if v == "" || strings.Trim(v, "0123456789") != "" {
		return 0, invalid
	}
	n, err := strconv.Atoi(v)
	// A number too big for an int is still an integer >= 1, and a cap that
	// large can never be reached: Atoi returns the largest int.
	if err != nil && !errors.Is(err, strconv.ErrRange) {
		return 0, invalid
	}
	if n < 1 {
		return 0, invalid
	}
	return n, nil
}
