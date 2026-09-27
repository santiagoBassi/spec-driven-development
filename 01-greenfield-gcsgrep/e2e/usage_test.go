package e2e

import (
	"fmt"
	"testing"
)

// TestVC05 cubre FR-5: una regex inválida con -E sale con 2 y
// "invalid pattern: <detalle del motor>". El detalle no está fijado por la
// spec, así que solo se exige el prefijo.
func TestVC05(t *testing.T) {
	loc := "gs://" + bucket + "/logs/"
	requireUsageErrorPrefix(t, []string{"-E", "(", loc}, "gcsgrep: invalid pattern: ")
}

// TestVC06 cubre FR-6: el patrón vacío se rechaza, con o sin -E y también
// después de --.
func TestVC06(t *testing.T) {
	loc := "gs://" + bucket + "/logs/"
	for _, args := range [][]string{
		{"", loc},
		{"-E", "", loc},
		{"--", "", loc},
	} {
		requireUsageError(t, args, "gcsgrep: empty pattern")
	}
}

// TestVC14 cubre FR-14: ubicaciones con formato inválido.
func TestVC14(t *testing.T) {
	locs := []string{"logs/", "s3://" + bucket + "/", "gs://", "gs:///logs/", "gs://" + bucket}
	for _, loc := range locs {
		requireUsageError(t, []string{"timeout", loc}, fmt.Sprintf("gcsgrep: invalid location: %q", loc))
	}
}

// TestVC22 cubre FR-22: un flag desconocido.
func TestVC22(t *testing.T) {
	requireUsageError(t, []string{"-1]", "gs://" + bucket + "/logs/db/postgres.log"}, `gcsgrep: unknown flag: "-1]"`)
	requireUsageError(t, []string{"-v", "timeout", "gs://" + bucket + "/logs/"}, `gcsgrep: unknown flag: "-v"`)
}

// TestVC23 cubre FR-23: la cantidad de posicionales tiene que ser
// exactamente dos.
func TestVC23(t *testing.T) {
	requireUsageError(t, nil, "gcsgrep: expected 2 arguments (pattern and location), got 0")
	requireUsageError(t, []string{"timeout"}, "gcsgrep: expected 2 arguments (pattern and location), got 1")
	requireUsageError(t, []string{"timeout", "gs://" + bucket + "/logs/", "extra"}, "gcsgrep: expected 2 arguments (pattern and location), got 3")
}
