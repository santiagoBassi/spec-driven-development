// Package e2e tiene un test por VC de la Iteración 1: cada uno compila e
// invoca ./gcsgrep como proceso aparte, contra el bucket de fixtures $B con
// las identidades lectora y sin-acceso.
//
// Variables de entorno obligatorias (si falta alguna, TestMain falla, no
// saltea nada):
//
//	GCSGREP_BUCKET      nombre de $B (sin gs://), p. ej. sdd-fardenghi-itba
//	GCSGREP_LECTORA     ruta a la clave JSON de la service account lectora
//	GCSGREP_SIN_ACCESO  ruta a la clave JSON de la service account sin-acceso
//
// Además necesita gcloud en el PATH, para la foto de VC-39.
package e2e

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

var (
	binPath        string
	bucket         string
	lectoraCreds   string
	sinAccesoCreds string
	vc39Before     string
)

func TestMain(m *testing.M) {
	os.Exit(mainRun(m))
}

func mainRun(m *testing.M) int {
	bucket = os.Getenv("GCSGREP_BUCKET")
	lectoraCreds = os.Getenv("GCSGREP_LECTORA")
	sinAccesoCreds = os.Getenv("GCSGREP_SIN_ACCESO")
	if bucket == "" || lectoraCreds == "" || sinAccesoCreds == "" {
		fmt.Fprintln(os.Stderr, "e2e: GCSGREP_BUCKET, GCSGREP_LECTORA y GCSGREP_SIN_ACCESO son obligatorias")
		return 1
	}

	dir, err := os.MkdirTemp("", "gcsgrep-e2e-bin-")
	if err != nil {
		fmt.Fprintln(os.Stderr, "e2e:", err)
		return 1
	}
	defer os.RemoveAll(dir)
	binPath = filepath.Join(dir, "gcsgrep")

	build := exec.Command("go", "build", "-o", binPath, "../cmd/gcsgrep")
	build.Env = append(os.Environ(), "GOTOOLCHAIN=local")
	build.Stdout = os.Stdout
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "e2e: building gcsgrep:", err)
		return 1
	}

	// VC-39 (BR-1): foto de $B antes de correr la suite. zz_vc39_test.go la
	// compara con una foto nueva al final.
	before, err := snapshotBucket()
	if err != nil {
		fmt.Fprintln(os.Stderr, "e2e: VC-39 snapshot:", err)
		return 1
	}
	vc39Before = before

	return m.Run()
}

// snapshotBucket lista $B con lectora, tal como pide VC-39.
func snapshotBucket() (string, error) {
	cfgDir, err := os.MkdirTemp("", "gcsgrep-e2e-cfg-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(cfgDir)

	cmd := exec.Command("gcloud", "storage", "objects", "list", "gs://"+bucket+"/**",
		"--format=value(name,generation,metageneration)")
	cmd.Env = append(os.Environ(),
		"CLOUDSDK_CONFIG="+cfgDir,
		"CLOUDSDK_AUTH_CREDENTIAL_FILE_OVERRIDE="+lectoraCreds,
	)
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("gcloud storage objects list: %w (stderr: %s)", err, ee.Stderr)
		}
		return "", fmt.Errorf("gcloud storage objects list: %w", err)
	}
	lines := strings.Split(strings.TrimRight(string(out), "\n"), "\n")
	sort.Strings(lines)
	return strings.Join(lines, "\n"), nil
}

// runOpts configura una invocación de ./gcsgrep con un entorno armado desde
// cero: sin LANG ni STORAGE_EMULATOR_HOST salvo que env los agregue.
type runOpts struct {
	args  []string
	creds string   // ruta a la clave JSON para GOOGLE_APPLICATION_CREDENTIALS; "" = sin ADC
	env   []string // variables extra, p. ej. "STORAGE_EMULATOR_HOST=...", "LANG=C"
}

type result struct {
	stdout   []byte
	stderr   []byte
	exitCode int
}

func runTool(t *testing.T, o runOpts) result {
	t.Helper()
	cmd := exec.Command(binPath, o.args...)
	env := []string{"PATH=" + os.Getenv("PATH")}
	if o.creds != "" {
		env = append(env, "HOME="+t.TempDir(), "GOOGLE_APPLICATION_CREDENTIALS="+o.creds)
	} else {
		// Entorno sin ADC: HOME y CLOUDSDK_CONFIG frescos, sin
		// GOOGLE_APPLICATION_CREDENTIALS.
		env = append(env, "HOME="+t.TempDir(), "CLOUDSDK_CONFIG="+t.TempDir())
	}
	env = append(env, o.env...)
	cmd.Env = env

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	exitCode := 0
	if err != nil {
		ee, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatalf("running gcsgrep %v: %v", o.args, err)
		}
		exitCode = ee.ExitCode()
	}
	return result{stdout: stdout.Bytes(), stderr: stderr.Bytes(), exitCode: exitCode}
}

func requireExit(t *testing.T, r result, want int) {
	t.Helper()
	if r.exitCode != want {
		t.Fatalf("exit = %d, want %d (stdout=%q stderr=%q)", r.exitCode, want, r.stdout, r.stderr)
	}
}

// connListener acepta conexiones TCP y cuenta cuántas veces se conectaron;
// es el listener del chequeo P.
type connListener struct {
	ln    net.Listener
	count int32
}

func newConnListener(t *testing.T) *connListener {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	cl := &connListener{ln: ln}
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			atomic.AddInt32(&cl.count, 1)
			conn.Close()
		}
	}()
	t.Cleanup(func() { ln.Close() })
	return cl
}

func (c *connListener) addr() string { return c.ln.Addr().String() }
func (c *connListener) conns() int   { return int(atomic.LoadInt32(&c.count)) }

// requireUsageError corre args con lectora y exige el mensaje exacto y
// exit 2; después repite el chequeo P descripto en la spec ("Entorno de
// verificación"): mismo comando sin ADC con STORAGE_EMULATOR_HOST apuntando
// a un listener, y exige el mismo mensaje (no el de credenciales) y cero
// conexiones.
func requireUsageError(t *testing.T, args []string, wantStderr string) {
	t.Helper()
	checkUsage(t, args, func(s string) bool { return s == wantStderr+"\n" }, wantStderr+"\n")
}

// requireUsageErrorPrefix es como requireUsageError, pero solo exige que
// stderr empiece con wantPrefix (para mensajes con el detalle del motor de
// regex, que no está fijado por la spec).
func requireUsageErrorPrefix(t *testing.T, args []string, wantPrefix string) {
	t.Helper()
	checkUsage(t, args, func(s string) bool { return strings.HasPrefix(s, wantPrefix) }, wantPrefix+"...")
}

func checkUsage(t *testing.T, args []string, ok func(string) bool, describe string) {
	t.Helper()
	r := runTool(t, runOpts{args: args, creds: lectoraCreds})
	if r.exitCode != 2 {
		t.Fatalf("exit = %d, want 2 (stdout=%q stderr=%q)", r.exitCode, r.stdout, r.stderr)
	}
	if len(r.stdout) != 0 {
		t.Errorf("stdout = %q, want empty", r.stdout)
	}
	if !ok(string(r.stderr)) {
		t.Errorf("stderr = %q, want %s", r.stderr, describe)
	}

	cl := newConnListener(t)
	r2 := runTool(t, runOpts{args: args, env: []string{"STORAGE_EMULATOR_HOST=" + cl.addr()}})
	if r2.exitCode != 2 {
		t.Errorf("chequeo P: exit = %d, want 2 (stdout=%q stderr=%q)", r2.exitCode, r2.stdout, r2.stderr)
	}
	if len(r2.stdout) != 0 {
		t.Errorf("chequeo P: stdout = %q, want empty", r2.stdout)
	}
	if !ok(string(r2.stderr)) {
		t.Errorf("chequeo P: stderr = %q, want %s (no debe ser el de credenciales ausentes)", r2.stderr, describe)
	}
	time.Sleep(150 * time.Millisecond)
	if n := cl.conns(); n != 0 {
		t.Errorf("chequeo P: el listener registró %d conexiones, quería 0", n)
	}
}

func stdoutLines(stdout []byte) []string {
	s := strings.TrimSuffix(string(stdout), "\n")
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

func assertLinesUnordered(t *testing.T, got, want []string) {
	t.Helper()
	gotSorted := append([]string(nil), got...)
	wantSorted := append([]string(nil), want...)
	sort.Strings(gotSorted)
	sort.Strings(wantSorted)
	if len(gotSorted) != len(wantSorted) {
		t.Fatalf("stdout tiene %d líneas, quería %d.\ngot:  %v\nwant: %v", len(gotSorted), len(wantSorted), got, want)
	}
	for i := range gotSorted {
		if gotSorted[i] != wantSorted[i] {
			t.Errorf("stdout no coincide.\ngot:  %v\nwant: %v", got, want)
			return
		}
	}
}

func fixturesDir() string {
	return filepath.Join("..", "test-fixtures")
}

// fixtureLines lee un fixture y lo separa en líneas (sin el terminador),
// para armar la salida esperada sin repetir contenido a mano.
func fixtureLines(t *testing.T, rel string) []string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(fixturesDir(), rel))
	if err != nil {
		t.Fatalf("reading fixture %s: %v", rel, err)
	}
	s := strings.TrimSuffix(string(data), "\n")
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}
