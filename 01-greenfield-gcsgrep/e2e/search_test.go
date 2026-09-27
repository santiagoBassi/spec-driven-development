package e2e

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gcsgrep/internal/search"
)

func TestVC01(t *testing.T) {
	loc := "gs://" + bucket + "/logs/app/"
	r := runTool(t, runOpts{args: []string{"timeout", loc}, creds: lectoraCreds})
	requireExit(t, r, 0)
	want := []string{
		"gs://" + bucket + "/logs/app/api.log:2026-09-22 10:05:22 ERROR Request timeout after 3000ms uri=/api/v1/checkout",
		"gs://" + bucket + "/logs/app/api.log:2026-09-22 10:50:23 ERROR Gateway timeout=504 from upstream payment service",
		"gs://" + bucket + "/logs/app/worker.log:job_id=102 queue=default status=ERROR reason=timeout_exceeded retries=3",
	}
	assertLinesUnordered(t, stdoutLines(r.stdout), want)
}

func TestVC03(t *testing.T) {
	loc1 := "gs://" + bucket + "/logs/app/worker.log"
	r1 := runTool(t, runOpts{args: []string{"job_id=10.", loc1}, creds: lectoraCreds})
	requireExit(t, r1, 1)
	if len(r1.stdout) != 0 {
		t.Errorf("stdout = %q, want empty", r1.stdout)
	}

	loc2 := "gs://" + bucket + "/logs/"
	r2 := runTool(t, runOpts{args: []string{"(", loc2}, creds: lectoraCreds})
	requireExit(t, r2, 1)
}

func TestVC04(t *testing.T) {
	loc := "gs://" + bucket + "/logs/app/worker.log"
	r := runTool(t, runOpts{args: []string{"-E", `job_id=\d+ queue=\w+ status=ERROR`, loc}, creds: lectoraCreds})
	requireExit(t, r, 0)

	fixture := fixtureLines(t, "logs/app/worker.log")
	var want []string
	for _, l := range fixture {
		if strings.Contains(l, "job_id=102") || strings.Contains(l, "job_id=104") {
			want = append(want, "gs://"+bucket+"/logs/app/worker.log:"+l)
		}
	}
	if len(want) != 2 {
		t.Fatalf("fixture desviado: %d líneas con job_id=102/104 en worker.log, quería 2", len(want))
	}
	assertLinesUnordered(t, stdoutLines(r.stdout), want)

	r2 := runTool(t, runOpts{args: []string{"-E", "job_id=10.", loc}, creds: lectoraCreds})
	requireExit(t, r2, 0)
	if n := len(stdoutLines(r2.stdout)); n != len(fixture) {
		t.Errorf("job_id=10. como regex debe matchear las %d líneas de worker.log, matcheó %d", len(fixture), n)
	}
}

func TestVC07a(t *testing.T) {
	loc := "gs://" + bucket + "/logs/app/api.log"
	r := runTool(t, runOpts{args: []string{"-i", "timeout", loc}, creds: lectoraCreds})
	requireExit(t, r, 0)
	if n := len(stdoutLines(r.stdout)); n != 4 {
		t.Errorf("-i timeout: %d líneas, quería 4", n)
	}

	r2 := runTool(t, runOpts{args: []string{"timeout", loc}, creds: lectoraCreds})
	requireExit(t, r2, 0)
	if n := len(stdoutLines(r2.stdout)); n != 2 {
		t.Errorf("sin -i: %d líneas, quería 2", n)
	}

	accLoc := "gs://" + bucket + "/data/acentos.log"
	r3 := runTool(t, runOpts{args: []string{"-i", "ñandú árbol", accLoc}, creds: lectoraCreds, env: []string{"LANG=C"}})
	requireExit(t, r3, 0)
	want := "gs://" + bucket + "/data/acentos.log:ÑANDÚ ÁRBOL"
	if got := stdoutLines(r3.stdout); len(got) != 1 || got[0] != want {
		t.Errorf("got %v, want [%s]", got, want)
	}

	r4 := runTool(t, runOpts{args: []string{"ñandú árbol", accLoc}, creds: lectoraCreds})
	requireExit(t, r4, 1)
}

func TestVC08(t *testing.T) {
	loc := "gs://" + bucket + "/logs/app/api.log"
	r := runTool(t, runOpts{args: []string{"-n", "timeout", loc}, creds: lectoraCreds})
	requireExit(t, r, 0)
	want := []string{
		"gs://" + bucket + "/logs/app/api.log:4:2026-09-22 10:05:22 ERROR Request timeout after 3000ms uri=/api/v1/checkout",
		"gs://" + bucket + "/logs/app/api.log:14:2026-09-22 10:50:23 ERROR Gateway timeout=504 from upstream payment service",
	}
	got := stdoutLines(r.stdout)
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("línea %d: got %q, want %q (orden estrictamente creciente dentro del objeto)", i, got[i], want[i])
		}
	}
}

func TestVC09a(t *testing.T) {
	loc := "gs://" + bucket + "/data/windows_crlf.txt"
	r := runTool(t, runOpts{args: []string{"-n", "timeout", loc}, creds: lectoraCreds})
	requireExit(t, r, 0)
	want := "gs://" + bucket + "/data/windows_crlf.txt:2:Linea 2 con timeout y terminacion Windows"
	got := stdoutLines(r.stdout)
	if len(got) != 1 || got[0] != want {
		t.Fatalf("got %v, want [%s]", got, want)
	}
	if bytes.ContainsRune(r.stdout, '\r') {
		t.Errorf("stdout no debe contener ningún byte 0x0d")
	}

	r2 := runTool(t, runOpts{args: []string{"-E", "Windows$", loc}, creds: lectoraCreds})
	requireExit(t, r2, 0)
}

func TestVC09b(t *testing.T) {
	loc := "gs://" + bucket + "/data/sin_salto_final.txt"
	r := runTool(t, runOpts{args: []string{"-n", "-E", "salto final$", loc}, creds: lectoraCreds})
	requireExit(t, r, 0)
	want := "gs://" + bucket + "/data/sin_salto_final.txt:2:Ultima linea sin salto final"
	got := stdoutLines(r.stdout)
	if len(got) != 1 || got[0] != want {
		t.Fatalf("got %v, want [%s]", got, want)
	}
	if len(r.stdout) == 0 || r.stdout[len(r.stdout)-1] != '\n' {
		t.Errorf("el último byte de stdout debe ser 0x0a, stdout = %q", r.stdout)
	}
}

func TestVC21(t *testing.T) {
	loc := "gs://" + bucket + "/logs/db/postgres.log"
	r := runTool(t, runOpts{args: []string{"--", "-1]", loc}, creds: lectoraCreds})
	requireExit(t, r, 0)

	fixture := fixtureLines(t, "logs/db/postgres.log")
	var want []string
	for _, l := range fixture {
		if strings.Contains(l, "-1]") {
			want = append(want, "gs://"+bucket+"/logs/db/postgres.log:"+l)
		}
	}
	if len(want) != 5 {
		t.Fatalf("fixture desviado: %d líneas con \"-1]\" en postgres.log, quería 5", len(want))
	}
	assertLinesUnordered(t, stdoutLines(r.stdout), want)
}

// TestVC46 cubre BR-8: una línea de más de 1 MiB se busca completa, pero se
// imprime truncada a su primer MiB + "...", incluso con un match posterior
// al límite o una regex que lo cruza.
func TestVC46(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(fixturesDir(), "edge-cases", "long_line_exceeds_1mb.log"))
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}
	nl := bytes.IndexByte(data, '\n')
	if nl < 0 {
		t.Fatalf("fixture sin salto de línea")
	}
	line1 := data[:nl]

	const lateOffset1Based = 1099838
	if lateOffset1Based > len(line1) || !bytes.HasPrefix(line1[lateOffset1Based-1:], []byte("timeout=late")) {
		t.Fatalf("fixture desviado: no hay \"timeout=late\" en el byte %d de la línea 1", lateOffset1Based)
	}
	if len(line1) <= search.MaxLine {
		t.Fatalf("fixture desviado: la línea 1 mide %d bytes, no supera 1 MiB", len(line1))
	}

	loc := "gs://" + bucket + "/edge-cases/long_line_exceeds_1mb.log"
	prefix := "gs://" + bucket + "/edge-cases/long_line_exceeds_1mb.log:"
	want := append([]byte(prefix), line1[:search.MaxLine]...)
	want = append(want, []byte("...\n")...)

	for _, pattern := range []string{"timeout=early", "timeout=late"} {
		r := runTool(t, runOpts{args: []string{pattern, loc}, creds: lectoraCreds})
		requireExit(t, r, 0)
		if !bytes.Equal(r.stdout, want) {
			t.Errorf("pattern %q: stdout no coincide (len got=%d want=%d)", pattern, len(r.stdout), len(want))
		}
	}

	r := runTool(t, runOpts{args: []string{"-E", "timeout=early.*timeout=late", loc}, creds: lectoraCreds})
	requireExit(t, r, 0)
	if !bytes.Equal(r.stdout, want) {
		t.Errorf("regex cruzando el límite: stdout no coincide (len got=%d want=%d)", len(r.stdout), len(want))
	}
}
