package cli

import "testing"

func TestParseValid(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want Config
	}{
		{
			name: "minimal",
			args: []string{"timeout", "gs://b/logs/"},
			want: Config{Max: defaultMax, Pattern: "timeout", Location: "gs://b/logs/"},
		},
		{
			name: "all flags",
			args: []string{"-E", "-i", "-n", "--max", "5", "pat", "gs://b/"},
			want: Config{Extended: true, IgnoreCase: true, LineNumber: true, Max: 5, Pattern: "pat", Location: "gs://b/"},
		},
		{
			name: "max unlimited",
			args: []string{"--max", "unlimited", "pat", "gs://b/"},
			want: Config{Max: Unlimited, Pattern: "pat", Location: "gs://b/"},
		},
		{
			name: "double dash makes following args positional",
			args: []string{"--", "-1]", "gs://b/o"},
			want: Config{Max: defaultMax, Pattern: "-1]", Location: "gs://b/o"},
		},
		{
			name: "max overflow is a valid unreachable cap",
			args: []string{"--max", "99999999999999999999", "pat", "gs://b/"},
			want: Config{Max: Unlimited, Pattern: "pat", Location: "gs://b/"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := Parse(c.args)
			if err != nil {
				t.Fatalf("Parse(%v): unexpected error: %v", c.args, err)
			}
			if got != c.want {
				t.Errorf("Parse(%v) = %+v, want %+v", c.args, got, c.want)
			}
		})
	}
}

// TestParseUsageErrors cubre los mensajes de uso que decide cli sin tocar
// GCS: FR-6, FR-22, FR-23 y BR-4 (FR-5 y FR-14 son de otros paquetes).
func TestParseUsageErrors(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want string
	}{
		{"empty pattern", []string{"", "gs://b/"}, "empty pattern"},
		{"empty pattern with -E", []string{"-E", "", "gs://b/"}, "empty pattern"},
		{"unknown flag", []string{"-1]", "gs://b/o"}, `unknown flag: "-1]"`},
		{"unknown long flag", []string{"-v", "timeout", "gs://b/"}, `unknown flag: "-v"`},
		{"no args", nil, "expected 2 arguments (pattern and location), got 0"},
		{"one arg", []string{"timeout"}, "expected 2 arguments (pattern and location), got 1"},
		{"three args", []string{"timeout", "gs://b/", "extra"}, "expected 2 arguments (pattern and location), got 3"},
		{"max zero", []string{"--max", "0", "p", "gs://b/"}, `invalid value for --max: "0" (integer >= 1 or unlimited)`},
		{"max negative", []string{"--max", "-3", "p", "gs://b/"}, `invalid value for --max: "-3" (integer >= 1 or unlimited)`},
		{"max not a number", []string{"--max", "abc", "p", "gs://b/"}, `invalid value for --max: "abc" (integer >= 1 or unlimited)`},
		{"max decimal", []string{"--max", "1.5", "p", "gs://b/"}, `invalid value for --max: "1.5" (integer >= 1 or unlimited)`},
		{"max plus sign", []string{"--max", "+5", "p", "gs://b/"}, `invalid value for --max: "+5" (integer >= 1 or unlimited)`},
		{"max leading space", []string{"--max", " 5", "p", "gs://b/"}, `invalid value for --max: " 5" (integer >= 1 or unlimited)`},
		{"max missing value", []string{"timeout", "gs://b/", "--max"}, "flag --max requires a value"},
		{"max takes next arg even if flag-like", []string{"--max", "-3", "p", "gs://b/"}, `invalid value for --max: "-3" (integer >= 1 or unlimited)`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := Parse(c.args)
			if err == nil {
				t.Fatalf("Parse(%v): expected error, got nil", c.args)
			}
			if err.Error() != c.want {
				t.Errorf("Parse(%v) error = %q, want %q", c.args, err.Error(), c.want)
			}
			if _, ok := err.(*ErrUsage); !ok {
				t.Errorf("Parse(%v) error type = %T, want *ErrUsage", c.args, err)
			}
		})
	}
}

// TestParseErrorOrder comprueba D-08: los flags se informan en el orden de
// argv, antes que la cantidad de posicionales o el patrón vacío.
func TestParseErrorOrder(t *testing.T) {
	_, err := Parse([]string{"-v", "--max", "0", "", "gs://b/", "extra"})
	if err == nil || err.Error() != `unknown flag: "-v"` {
		t.Fatalf("Parse: error = %v, want unknown flag -v first", err)
	}
}
