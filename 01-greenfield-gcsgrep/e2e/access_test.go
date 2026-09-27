package e2e

import (
	"testing"
	"time"
)

// TestVC15b cubre FR-15b: un prefijo sin objetos en un bucket que existe no
// es error: exit 1, stdout y stderr vacíos.
func TestVC15b(t *testing.T) {
	loc := "gs://" + bucket + "/no-existe/"
	r := runTool(t, runOpts{args: []string{"timeout", loc}, creds: lectoraCreds})
	requireExit(t, r, 1)
	if len(r.stdout) != 0 {
		t.Errorf("stdout = %q, want empty", r.stdout)
	}
	if len(r.stderr) != 0 {
		t.Errorf("stderr = %q, want empty", r.stderr)
	}
}

// TestVC35 cubre FR-35: sin ADC, la corrida sale con 2 y el mensaje fijo,
// antes de intentar nada contra GCS (el listener del chequeo P no recibe
// ninguna conexión), aunque STORAGE_EMULATOR_HOST esté definido.
func TestVC35(t *testing.T) {
	cl := newConnListener(t)
	loc := "gs://" + bucket + "/logs/"
	r := runTool(t, runOpts{args: []string{"timeout", loc}, env: []string{"STORAGE_EMULATOR_HOST=" + cl.addr()}})
	requireExit(t, r, 2)
	if len(r.stdout) != 0 {
		t.Errorf("stdout = %q, want empty", r.stdout)
	}
	want := "gcsgrep: no Application Default Credentials found (run: gcloud auth application-default login)\n"
	if string(r.stderr) != want {
		t.Errorf("stderr = %q, want %q", r.stderr, want)
	}
	time.Sleep(150 * time.Millisecond)
	if n := cl.conns(); n != 0 {
		t.Errorf("el listener registró %d conexiones, quería 0", n)
	}
}

// TestVC40 cubre BR-2: sin-acceso no puede leer $B (ni prefijo ni objeto
// puntual), y lectora sigue pudiendo.
func TestVC40(t *testing.T) {
	prefixLoc := "gs://" + bucket + "/logs/app/"
	r := runTool(t, runOpts{args: []string{"timeout", prefixLoc}, creds: sinAccesoCreds})
	requireExit(t, r, 2)
	wantPrefix := "gcsgrep: access denied: " + prefixLoc + "\n"
	if string(r.stderr) != wantPrefix {
		t.Errorf("stderr = %q, want %q", r.stderr, wantPrefix)
	}

	objLoc := "gs://" + bucket + "/logs/app/api.log"
	r2 := runTool(t, runOpts{args: []string{"timeout", objLoc}, creds: sinAccesoCreds})
	requireExit(t, r2, 2)
	wantObj := "gcsgrep: access denied: " + objLoc + "\n"
	if string(r2.stderr) != wantObj {
		t.Errorf("stderr = %q, want %q", r2.stderr, wantObj)
	}

	r3 := runTool(t, runOpts{args: []string{"timeout", prefixLoc}, creds: lectoraCreds})
	requireExit(t, r3, 0)
	want := []string{
		"gs://" + bucket + "/logs/app/api.log:2026-09-22 10:05:22 ERROR Request timeout after 3000ms uri=/api/v1/checkout",
		"gs://" + bucket + "/logs/app/api.log:2026-09-22 10:50:23 ERROR Gateway timeout=504 from upstream payment service",
		"gs://" + bucket + "/logs/app/worker.log:job_id=102 queue=default status=ERROR reason=timeout_exceeded retries=3",
	}
	assertLinesUnordered(t, stdoutLines(r3.stdout), want)
}
