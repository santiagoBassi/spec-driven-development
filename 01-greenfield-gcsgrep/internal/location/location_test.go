package location

import "testing"

func TestParseValid(t *testing.T) {
	cases := []struct {
		raw    string
		bucket string
		path   string
		mode   Mode
	}{
		{"gs://b/", "b", "", ModeBucket},
		{"gs://b/logs/", "b", "logs", ModePrefix},
		{"gs://b/logs/app/", "b", "logs/app", ModePrefix},
		{"gs://b/logs/app/api.log", "b", "logs/app/api.log", ModeObject},
	}
	for _, c := range cases {
		loc, err := Parse(c.raw)
		if err != nil {
			t.Fatalf("Parse(%q): unexpected error: %v", c.raw, err)
		}
		if loc.Bucket != c.bucket || loc.Path != c.path || loc.Mode != c.mode {
			t.Errorf("Parse(%q) = %+v, want bucket=%q path=%q mode=%v", c.raw, loc, c.bucket, c.path, c.mode)
		}
		if loc.Raw != c.raw {
			t.Errorf("Parse(%q).Raw = %q, want %q", c.raw, loc.Raw, c.raw)
		}
	}
}

// TestVC15aParcial es la evidencia de la Iteración 1 para VC-15a (FR-15a):
// un bucket se interpreta con el mismo modo y el mismo prefijo vacío que un
// prefijo sin objetos (VC-15b, cubierto en e2e), así que un bucket vacío se
// trata igual. Falta observarlo contra un bucket real sin objetos, que
// necesita el servidor de prueba (Iteración 2).
func TestVC15aParcial(t *testing.T) {
	loc, err := Parse("gs://empty/")
	if err != nil {
		t.Fatalf("Parse: unexpected error: %v", err)
	}
	if loc.Mode != ModeBucket || loc.Path != "" || loc.ListPrefix() != "" {
		t.Errorf("Parse(gs://empty/) = %+v, want ModeBucket con prefijo vacío", loc)
	}
}

// TestParseInvalid ejercita FR-14 / VC-14.
func TestParseInvalid(t *testing.T) {
	invalid := []string{"logs/", "s3://$B/", "gs://", "gs:///logs/", "gs://$B"}
	for _, raw := range invalid {
		_, err := Parse(raw)
		if err == nil {
			t.Fatalf("Parse(%q): expected error, got nil", raw)
		}
		want := `invalid location: "` + raw + `"`
		if err.Error() != want {
			t.Errorf("Parse(%q) error = %q, want %q", raw, err.Error(), want)
		}
	}
}
