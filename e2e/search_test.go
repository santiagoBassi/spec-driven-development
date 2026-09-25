package e2e

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// vc1Lines is the output of `gcsgrep timeout gs://$B/logs/app/`.
func vc1Lines(t testing.TB) []string {
	t.Helper()
	return []string{
		gs(t, "logs/app/api.log") + ":2026-09-22 10:05:22 ERROR Request timeout after 3000ms uri=/api/v1/checkout",
		gs(t, "logs/app/api.log") + ":2026-09-22 10:50:23 ERROR Gateway timeout=504 from upstream payment service",
		gs(t, "logs/app/worker.log") + ":job_id=102 queue=default status=ERROR reason=timeout_exceeded retries=3",
	}
}

// fixture returns the lines of a local fixture file; the bucket holds the same bytes.
func fixture(t testing.TB, rel string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(repoRoot, "test-fixtures", rel))
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}
	return data
}

// prefixed prefixes each of texts with "<uri>:".
func prefixed(uri string, texts ...string) []string {
	out := make([]string, len(texts))
	for i, s := range texts {
		out[i] = uri + ":" + s
	}
	return out
}

var workerLines = []string{
	"job_id=101 queue=default status=SUCCESS duration=120ms",
	"job_id=102 queue=default status=ERROR reason=timeout_exceeded retries=3",
	"job_id=103 queue=priority status=SUCCESS duration=45ms",
	"job_id=104 queue=batch status=ERROR reason=connection_reset retries=1",
	"job_id=105 queue=default status=TIMEOUT retries=5 action=deadletter",
	"job_id=106 queue=default status=SUCCESS duration=95ms",
}

// FR-1
func TestVC01(t *testing.T) {
	r := asLectora(t, "timeout", gs(t, "logs/app/"))
	requireExit(t, r, 0)
	requireLines(t, r, vc1Lines(t))
}

// FR-3
func TestVC03(t *testing.T) {
	t.Run("the pattern is a literal without -E", func(t *testing.T) {
		r := asLectora(t, "job_id=10.", gs(t, "logs/app/worker.log"))
		requireExit(t, r, 1)
		requireStdout(t, r, "")
	})
	t.Run("a pattern that is not a valid regex is not an error", func(t *testing.T) {
		r := asLectora(t, "(", gs(t, "logs/"))
		requireExit(t, r, 1)
		requireStdout(t, r, "")
	})
}

// FR-4
func TestVC04(t *testing.T) {
	worker := gs(t, "logs/app/worker.log")

	r := asLectora(t, "-E", `job_id=\d+ queue=\w+ status=ERROR`, worker)
	requireExit(t, r, 0)
	requireLines(t, r, prefixed(worker, workerLines[1], workerLines[3]))

	r = asLectora(t, "-E", "job_id=10.", worker)
	requireExit(t, r, 0)
	requireLines(t, r, prefixed(worker, workerLines...))
}

// FR-7
func TestVC07(t *testing.T) {
	api := gs(t, "logs/app/api.log")

	t.Run("-i matches every case", func(t *testing.T) {
		r := asLectora(t, "-i", "timeout", api)
		requireExit(t, r, 0)
		requireLines(t, r, prefixed(api,
			"2026-09-22 10:05:22 ERROR Request timeout after 3000ms uri=/api/v1/checkout",
			"2026-09-22 10:14:03 WARN Connection TIMEOUT to redis cache replica-0",
			"2026-09-22 10:30:15 ERROR Database Timeout connecting to replica-db",
			"2026-09-22 10:50:23 ERROR Gateway timeout=504 from upstream payment service",
		))
	})
	t.Run("without -i it is case-sensitive", func(t *testing.T) {
		r := asLectora(t, "timeout", api)
		requireExit(t, r, 0)
		if n := len(r.lines()); n != 2 {
			t.Errorf("got %d lines, want 2", n)
		}
	})
	t.Run("-i folds non-ASCII letters, with LANG=C", func(t *testing.T) {
		acentos := gs(t, "data/acentos.log")
		r := asLectoraEnv(t, []string{"LANG=C"}, "-i", "ñandú árbol", acentos)
		requireExit(t, r, 0)
		requireStdout(t, r, acentos+":ÑANDÚ ÁRBOL\n")

		r = asLectoraEnv(t, []string{"LANG=C"}, "ñandú árbol", acentos)
		requireExit(t, r, 1)
		requireStdout(t, r, "")
	})
}

// FR-8
func TestVC08(t *testing.T) {
	api := gs(t, "logs/app/api.log")
	r := asLectora(t, "-n", "timeout", api)
	requireExit(t, r, 0)
	requireStdout(t, r, api+":4:2026-09-22 10:05:22 ERROR Request timeout after 3000ms uri=/api/v1/checkout\n"+
		api+":14:2026-09-22 10:50:23 ERROR Gateway timeout=504 from upstream payment service\n")
}

// FR-9a
func TestVC09a(t *testing.T) {
	crlf := gs(t, "data/windows_crlf.txt")

	r := asLectora(t, "-n", "timeout", crlf)
	requireExit(t, r, 0)
	requireStdout(t, r, crlf+":2:Linea 2 con timeout y terminacion Windows\n")
	if strings.ContainsRune(r.stdout, '\r') {
		t.Errorf("stdout contains a 0x0d byte: %q", r.stdout)
	}

	r = asLectora(t, "-E", "Windows$", crlf)
	requireExit(t, r, 0)
}

// FR-9b
func TestVC09b(t *testing.T) {
	obj := gs(t, "data/sin_salto_final.txt")
	r := asLectora(t, "-n", "-E", "salto final$", obj)
	requireExit(t, r, 0)
	// The exact comparison includes the trailing "\n": the last byte of stdout is 0a.
	requireStdout(t, r, obj+":2:Ultima linea sin salto final\n")
}

// FR-15b
func TestVC15b(t *testing.T) {
	r := asLectora(t, "timeout", gs(t, "no-existe/"))
	requireExit(t, r, 1)
	requireStdout(t, r, "")
	requireStderr(t, r, "")
}

// FR-21
func TestVC21(t *testing.T) {
	postgres := gs(t, "logs/db/postgres.log")
	want := strings.Split(strings.TrimSuffix(string(fixture(t, "logs/db/postgres.log")), "\n"), "\n")
	if len(want) != 5 {
		t.Fatalf("fixture has %d lines, the VC expects 5", len(want))
	}

	r := asLectora(t, "--", "-1]", postgres)
	requireExit(t, r, 0)
	requireLines(t, r, prefixed(postgres, want...))
}

// BR-8
func TestVC46(t *testing.T) {
	long := gs(t, "edge-cases/long_line_exceeds_1mb.log")
	data := fixture(t, "edge-cases/long_line_exceeds_1mb.log")
	line1 := data[:bytes.IndexByte(data, '\n')]

	const limit = 1 << 20
	if len(line1) <= limit {
		t.Fatalf("fixture's first line has %d bytes, want more than %d", len(line1), limit)
	}
	// "timeout=late" starts at byte 1 099 838 of the line, counting from 1.
	if i := bytes.Index(line1, []byte("timeout=late")); i != 1099837 {
		t.Fatalf("timeout=late at offset %d, the VC expects 1099837", i)
	}
	want := long + ":" + string(line1[:limit]) + "...\n"

	for _, tc := range []struct {
		name string
		args []string
	}{
		{"match before the limit", []string{"timeout=early", long}},
		{"match after the limit", []string{"timeout=late", long}},
		{"regex across the limit", []string{"-E", "timeout=early.*timeout=late", long}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := asLectora(t, tc.args...)
			requireExit(t, r, 0)
			requireStdout(t, r, want)
		})
	}
}
