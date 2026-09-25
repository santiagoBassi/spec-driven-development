package location

import "testing"

func TestParseValid(t *testing.T) {
	tests := []struct {
		raw    string
		bucket string
		path   string
		mode   Mode
	}{
		{"gs://b/", "b", "", Prefix},
		{"gs://b/logs/", "b", "logs/", Prefix},
		{"gs://b/logs/app/", "b", "logs/app/", Prefix},
		{"gs://b/logs/app/api.log", "b", "logs/app/api.log", Object},
		{"gs://b/logs/app", "b", "logs/app", Object},
		{"gs://b/a b/c", "b", "a b/c", Object},
	}
	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			got, err := Parse(tt.raw)
			if err != nil {
				t.Fatalf("Parse(%q): %v", tt.raw, err)
			}
			if got.Raw != tt.raw || got.Bucket != tt.bucket || got.Path != tt.path || got.Mode != tt.mode {
				t.Errorf("Parse(%q) = %+v", tt.raw, got)
			}
		})
	}
}

func TestParseInvalid(t *testing.T) {
	for _, raw := range []string{"", "logs/", "s3://b/", "gs://", "gs:///logs/", "gs://b", "GS://b/", "/gs://b/"} {
		t.Run(raw, func(t *testing.T) {
			_, err := Parse(raw)
			if err == nil {
				t.Fatalf("Parse(%q) accepted an invalid location", raw)
			}
			if want := `invalid location: "` + raw + `"`; err.Error() != want {
				t.Errorf("error = %q, want %q", err.Error(), want)
			}
		})
	}
}

func TestURI(t *testing.T) {
	l, _ := Parse("gs://b/logs/")
	if got := l.URI("logs/a.log"); got != "gs://b/logs/a.log" {
		t.Errorf("URI = %q", got)
	}
}
