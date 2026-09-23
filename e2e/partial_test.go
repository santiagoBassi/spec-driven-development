package e2e

// Partial evidence for VCs whose full check needs the test server (Iteration 2).
//
// These VCs are NOT closed by the tests below: they are listed as "implemented,
// VC pending" in sdd/gcsgrep-cobertura-vc.md. The Parcial suffix keeps them from
// being read as a pass.

import (
	"strings"
	"testing"
)

// VC-12 (FR-12): the part observable against real GCS. Missing: no listing
// request and no read of a.log.bak, which needs the test server.
func TestVC12Parcial(t *testing.T) {
	api := gs(t, "logs/app/api.log")
	r := asLectora(t, "timeout", api)
	requireExit(t, r, 0)

	lines := r.lines()
	if len(lines) != 2 {
		t.Errorf("got %d lines, want 2", len(lines))
	}
	for _, l := range lines {
		if !strings.HasPrefix(l, api+":") {
			t.Errorf("line from another object: %q", l)
		}
	}
}

// VC-15 (FR-15): the part observable against real GCS. Missing: an existing
// bucket without objects (fake/empty), which needs the test server.
func TestVC15Parcial(t *testing.T) {
	r := asLectora(t, "timeout", gs(t, "no-existe/"))
	requireExit(t, r, 1)
	requireStdout(t, r, "")
	requireStderr(t, r, "")
}

// VC-41 (BR-3): the part observable against real GCS. Missing: the default cap
// of 1000 and that no page after the one holding object 1001 is requested.
func TestVC41Parcial(t *testing.T) {
	edge := gs(t, "edge-cases/")

	r := asLectora(t, "--max", "5", "timeout", edge)
	requireExit(t, r, 2)
	requireStdout(t, r, "")
	requireStderr(t, r, "gcsgrep: more than 5 objects under "+edge+"; use --max <N> or --max unlimited\n")

	r = asLectora(t, "--max", "6", "timeout", edge)
	requireExit(t, r, 0)
}

// VC-42 (BR-4): the part observable without a test server. Missing:
// --max unlimited reading 2500 objects.
func TestVC42Parcial(t *testing.T) {
	logs := gs(t, "logs/")
	for _, v := range []string{"0", "-3", "abc", "1.5"} {
		t.Run(v, func(t *testing.T) {
			requireUsageError(t,
				exact(`gcsgrep: invalid value for --max: "`+v+`" (integer >= 1 or unlimited)`),
				"--max", v, "timeout", logs)
		})
	}
}
