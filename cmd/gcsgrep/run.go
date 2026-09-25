package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"

	"gcsgrep/internal/cli"
	"gcsgrep/internal/gcs"
	"gcsgrep/internal/location"
	"gcsgrep/internal/output"
	"gcsgrep/internal/search"
)

// Exit codes, as in grep.
const (
	exitMatch   = 0 // at least one match
	exitNoMatch = 1 // no match at all
	exitError   = 2 // usage, credentials, listing, cap, or some object failed
)

// workers is how many objects are read at once. It is 1 in this iteration.
const workers = 1

// run executes one invocation and returns the exit code.
func run(args []string, stdout, stderr io.Writer) int {
	cfg, err := cli.Parse(args)
	if err != nil {
		output.Errorf(stderr, "%v", err)
		return exitError
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	creds, err := gcs.Credentials(ctx)
	if err != nil {
		output.Errorf(stderr, "%v", err)
		return exitError
	}
	client, err := gcs.NewClient(ctx, creds)
	if err != nil {
		output.Errorf(stderr, "cannot create GCS client: %v", err)
		return exitError
	}
	defer client.Close()

	// Phase 1: decide which objects to read. Nothing is read until this ends,
	// so the cap can abort the run before any read is paid for (BR-3).
	objects, err := resolve(ctx, client, cfg)
	if err != nil {
		output.Errorf(stderr, "%s", describe(err, cfg.Location))
		return exitError
	}

	// Phase 2: read them.
	printer := output.New(stdout, stderr, cfg.LineNumbers)
	res := readAll(ctx, cancel, client, cfg, printer, objects)
	switch {
	case res.fatal != nil:
		output.Errorf(stderr, "write error: %v", res.fatal)
		return exitError
	case res.failed:
		return exitError
	case res.matched:
		return exitMatch
	default:
		return exitNoMatch
	}
}

// resolve returns the names of the objects to search.
func resolve(ctx context.Context, client *gcs.Client, cfg *cli.Config) ([]string, error) {
	loc := cfg.Location
	if loc.Mode == location.Object {
		if err := client.Stat(ctx, loc.Bucket, loc.Path); err != nil {
			return nil, &phaseError{phase: "metadata", err: err}
		}
		return []string{loc.Path}, nil
	}
	names, err := client.List(ctx, loc.Bucket, loc.Path, cfg.Max)
	if err != nil {
		return nil, &phaseError{phase: "list", err: err}
	}
	return names, nil
}

// phaseError tags an error from the listing or metadata phase.
type phaseError struct {
	phase string // "list" or "metadata"
	err   error
}

func (e *phaseError) Error() string { return e.phase + " error: " + e.err.Error() }
func (e *phaseError) Unwrap() error { return e.err }

// describe turns an error of the listing/metadata phase into its stderr message
// (without the "gcsgrep: " prefix).
func describe(err error, loc location.Location) string {
	var tooMany *gcs.TooManyObjectsError
	switch {
	case errors.Is(err, gcs.ErrAccessDenied):
		return "access denied: " + loc.Raw
	case errors.As(err, &tooMany):
		return fmt.Sprintf("more than %d objects under %s; use --max <N> or --max unlimited", tooMany.Max, loc.Raw)
	}
	var pe *phaseError
	if errors.As(err, &pe) {
		return fmt.Sprintf("%s: %s error: %v", loc.Raw, pe.phase, pe.err)
	}
	return err.Error()
}

type result struct {
	matched bool  // some line matched
	failed  bool  // some object could not be read
	fatal   error // stdout can no longer be written
}

// readAll searches every object with a pool of workers. An object that fails is
// reported and the rest go on; the run then exits with 2 (FR-29a).
func readAll(ctx context.Context, cancel context.CancelFunc, client *gcs.Client, cfg *cli.Config, p *output.Printer, objects []string) result {
	var (
		matches atomic.Int64
		failed  atomic.Bool
		fatal   error
		once    sync.Once
	)

	jobs := make(chan string)
	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sc := search.NewScanner() // one per worker: it owns 1 MiB of buffer
			for name := range jobs {
				n, err := searchObject(ctx, client, sc, cfg, p, name)
				matches.Add(n)
				var we *writeError
				switch {
				case err == nil:
				case errors.As(err, &we):
					once.Do(func() { fatal = we.err })
					cancel()
				case ctx.Err() != nil:
					// Cancelled because of a fatal error elsewhere: not this object's fault.
				default:
					p.Warnf("%s: read error: %v", cfg.Location.URI(name), err)
					failed.Store(true)
				}
			}
		}()
	}

feed:
	for _, name := range objects {
		select {
		case jobs <- name:
		case <-ctx.Done():
			break feed
		}
	}
	close(jobs)
	wg.Wait()

	return result{matched: matches.Load() > 0, failed: failed.Load(), fatal: fatal}
}

// writeError marks a failure to write to stdout, which is not the object's fault.
type writeError struct{ err error }

func (e *writeError) Error() string { return e.err.Error() }
func (e *writeError) Unwrap() error { return e.err }

// searchObject streams one object through the scanner and prints its matches.
// It returns how many lines matched, even when it then fails.
func searchObject(ctx context.Context, client *gcs.Client, sc *search.Scanner, cfg *cli.Config, p *output.Printer, name string) (int64, error) {
	rc, err := client.Open(ctx, cfg.Location.Bucket, name)
	if err != nil {
		return 0, err
	}
	defer rc.Close()

	uri := cfg.Location.URI(name)
	var n int64
	err = sc.Scan(rc, cfg.Matcher, func(l search.Line) error {
		n++
		if err := p.Match(uri, l.No, l.Text, l.Truncated); err != nil {
			return &writeError{err}
		}
		return nil
	})
	return n, err
}
