// Package e2e verifies gcsgrep against the VCs of the spec, one test per VC
// (TestVC01 ... ). Every test runs the compiled ./gcsgrep as a separate process.
//
// The tests read their environment from:
//
//	GCSGREP_E2E_BUCKET      the fixtures bucket ($B in the spec)
//	GCSGREP_E2E_LECTORA     ADC file of the "lectora" identity (objectViewer on $B)
//	GCSGREP_E2E_SIN_ACCESO  ADC file of the "sin-acceso" identity (no role on $B)
//
// A test that needs a variable that is not set fails: it never skips.
package e2e

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const (
	envBucket    = "GCSGREP_E2E_BUCKET"
	envLectora   = "GCSGREP_E2E_LECTORA"
	envSinAcceso = "GCSGREP_E2E_SIN_ACCESO"

	runTimeout = 2 * time.Minute
)

var (
	repoRoot string
	binary   string

	// State of $B before the suite, for VC-39.
	snapshotBefore string
	snapshotErr    error
)

func TestMain(m *testing.M) {
	os.Exit(testMain(m))
}

func testMain(m *testing.M) int {
	root, err := findRoot()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	repoRoot = root
	binary = filepath.Join(root, "gcsgrep")

	build := exec.Command("go", "build", "-o", binary, "./cmd/gcsgrep")
	build.Dir = root
	if out, err := build.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "go build ./cmd/gcsgrep: %v\n%s", err, out)
		return 1
	}

	snapshotBefore, snapshotErr = takeSnapshot()
	return m.Run()
}

func findRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("go.mod not found above the working directory")
		}
		dir = parent
	}
}

// mustEnv returns a required environment variable, or fails the test.
func mustEnv(t testing.TB, name string) string {
	t.Helper()
	v := os.Getenv(name)
	if v == "" {
		t.Fatalf("%s is not set (see the header of e2e/main_test.go)", name)
	}
	return v
}

// bucket returns $B.
func bucket(t testing.TB) string {
	t.Helper()
	return mustEnv(t, envBucket)
}

// gs returns the gs:// URI of a path inside $B.
func gs(t testing.TB, path string) string {
	t.Helper()
	return "gs://" + bucket(t) + "/" + path
}

// ---- running the binary ----

type result struct {
	stdout, stderr string
	code           int
}

func (r result) String() string {
	return fmt.Sprintf("exit=%d\nstdout=%q\nstderr=%q", r.code, clip(r.stdout), clip(r.stderr))
}

func clip(s string) string {
	if len(s) > 600 {
		return s[:600] + fmt.Sprintf("...(%d bytes)", len(s))
	}
	return s
}

// lines returns stdout split in lines, without the final newline.
func (r result) lines() []string {
	if r.stdout == "" {
		return nil
	}
	return strings.Split(strings.TrimSuffix(r.stdout, "\n"), "\n")
}

// baseEnv is the process environment without anything that could steer the
// binary toward credentials or an endpoint the test did not choose.
func baseEnv(extra ...string) []string {
	var env []string
	for _, kv := range os.Environ() {
		key, _, _ := strings.Cut(kv, "=")
		switch key {
		case "GOOGLE_APPLICATION_CREDENTIALS", "STORAGE_EMULATOR_HOST", "LANG", "LC_ALL", "LC_CTYPE":
			continue
		}
		env = append(env, kv)
	}
	// With duplicate keys the last one wins.
	return append(env, extra...)
}

func adcEnv(adcFile string, extra ...string) []string {
	return baseEnv(append([]string{"GOOGLE_APPLICATION_CREDENTIALS=" + adcFile}, extra...)...)
}

// execute runs ./gcsgrep with the given environment.
func execute(t testing.TB, env []string, args ...string) result {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), runTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Dir = repoRoot
	cmd.Env = env
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr

	code := 0
	if err := cmd.Run(); err != nil {
		var ee *exec.ExitError
		if !errors.As(err, &ee) {
			t.Fatalf("running gcsgrep %q: %v", args, err)
		}
		code = ee.ExitCode()
	}
	return result{stdout.String(), stderr.String(), code}
}

// asLectora runs gcsgrep with the "lectora" identity as ADC.
func asLectora(t testing.TB, args ...string) result {
	t.Helper()
	return execute(t, adcEnv(mustEnv(t, envLectora)), args...)
}

// asLectoraEnv is asLectora with extra environment variables.
func asLectoraEnv(t testing.TB, extra []string, args ...string) result {
	t.Helper()
	return execute(t, adcEnv(mustEnv(t, envLectora), extra...), args...)
}

func asSinAcceso(t testing.TB, args ...string) result {
	t.Helper()
	return execute(t, adcEnv(mustEnv(t, envSinAcceso)), args...)
}

// ---- assertions ----

func requireExit(t testing.TB, r result, want int) {
	t.Helper()
	if r.code != want {
		t.Fatalf("exit code = %d, want %d\n%s", r.code, want, r)
	}
}

func requireStdout(t testing.TB, r result, want string) {
	t.Helper()
	if r.stdout != want {
		t.Errorf("stdout is not exact\n got: %q\nwant: %q", clip(r.stdout), clip(want))
	}
}

func requireStderr(t testing.TB, r result, want string) {
	t.Helper()
	if r.stderr != want {
		t.Errorf("stderr is not exact\n got: %q\nwant: %q", clip(r.stderr), clip(want))
	}
}

// requireLines compares the stdout lines with want, in any order: lines from
// different objects may interleave (FR-28).
func requireLines(t testing.TB, r result, want []string) {
	t.Helper()
	got := append([]string(nil), r.lines()...)
	w := append([]string(nil), want...)
	sort.Strings(got)
	sort.Strings(w)
	if strings.Join(got, "\n") != strings.Join(w, "\n") {
		t.Errorf("stdout lines differ (any order)\n got %d lines:\n%s\nwant %d lines:\n%s",
			len(got), clip(strings.Join(got, "\n")), len(w), clip(strings.Join(w, "\n")))
	}
}

// ---- check P: usage errors are detected before anything touches the network ----

// stderrWant is what a usage error must print: the whole line, or its start.
type stderrWant struct{ exact, prefix string }

func exact(msg string) stderrWant  { return stderrWant{exact: msg} }
func prefix(msg string) stderrWant { return stderrWant{prefix: msg} }

func (w stderrWant) check(t testing.TB, r result) {
	t.Helper()
	if w.exact != "" {
		if r.stderr != w.exact+"\n" {
			t.Errorf("stderr is not exact\n got: %q\nwant: %q", clip(r.stderr), w.exact+"\n")
		}
		return
	}
	first, _, _ := strings.Cut(r.stderr, "\n")
	if !strings.HasPrefix(first, w.prefix) {
		t.Errorf("first stderr line %q does not start with %q", first, w.prefix)
	}
}

// listener accepts TCP connections on a local port, counts them and never
// answers. Pointing STORAGE_EMULATOR_HOST at it shows whether the tool tried to
// reach GCS.
type listener struct {
	ln    net.Listener
	count atomic.Int64
	mu    sync.Mutex
	conns []net.Conn
}

func newListener(t testing.TB) *listener {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	l := &listener{ln: ln}
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			l.count.Add(1)
			l.mu.Lock()
			l.conns = append(l.conns, c)
			l.mu.Unlock()
		}
	}()
	t.Cleanup(func() {
		ln.Close()
		l.mu.Lock()
		defer l.mu.Unlock()
		for _, c := range l.conns {
			c.Close()
		}
	})
	return l
}

func (l *listener) addr() string { return l.ln.Addr().String() }

// requireNoConnections fails if anything connected. The process has already
// exited when this runs; the wait lets the accept loop see a connection that is
// still sitting in the kernel's queue.
func (l *listener) requireNoConnections(t testing.TB) {
	t.Helper()
	time.Sleep(250 * time.Millisecond)
	if n := l.count.Load(); n != 0 {
		t.Errorf("the tool opened %d connection(s) to the emulator endpoint; want 0", n)
	}
}

// noADCEnv is the "environment without ADC" of the spec, with the emulator
// endpoint pointing at the listener.
func noADCEnv(t testing.TB, emulator string) []string {
	t.Helper()
	return baseEnv(
		"HOME="+t.TempDir(),
		"CLOUDSDK_CONFIG="+t.TempDir(),
		"STORAGE_EMULATOR_HOST="+emulator,
	)
}

// requireUsageError checks an invocation that must fail as a usage error:
// exit 2, empty stdout and the wanted stderr, both with regular credentials and
// under check P (no ADC, endpoint = listener, and not a single connection).
func requireUsageError(t testing.TB, want stderrWant, args ...string) {
	t.Helper()

	r := asLectora(t, args...)
	requireExit(t, r, 2)
	requireStdout(t, r, "")
	want.check(t, r)

	ln := newListener(t)
	r = execute(t, noADCEnv(t, ln.addr()), args...)
	if r.code != 2 || r.stdout != "" {
		t.Errorf("check P: want exit 2 and empty stdout\n%s", r)
	}
	want.check(t, r) // the usage message, not the missing-credentials one
	ln.requireNoConnections(t)
}

// ---- state of $B, for VC-39 ----

// takeSnapshot lists name, generation and metageneration of every object in $B
// as the "lectora" identity, with the gcloud command of the spec. It returns an
// error when the environment is not set up, so that TestVC39 fails with it.
func takeSnapshot() (string, error) {
	b, lectora := os.Getenv(envBucket), os.Getenv(envLectora)
	if b == "" || lectora == "" {
		return "", fmt.Errorf("%s and %s must be set to snapshot the bucket", envBucket, envLectora)
	}
	token, err := accessToken(lectora)
	if err != nil {
		return "", fmt.Errorf("token for %s: %w", lectora, err)
	}
	dir, err := os.MkdirTemp("", "gcsgrep-e2e-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(dir)
	tokenFile := filepath.Join(dir, "token")
	if err := os.WriteFile(tokenFile, []byte(token), 0o600); err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), runTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "gcloud", "storage", "objects", "list", "gs://"+b+"/**",
		"--format=value(name,generation,metageneration)", "--access-token-file="+tokenFile)
	cmd.Env = append(os.Environ(), "CLOUDSDK_CORE_DISABLE_PROMPTS=1")
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("gcloud storage objects list: %v\n%s", err, stderr.String())
	}
	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	sort.Strings(lines)
	return strings.Join(lines, "\n"), nil
}
