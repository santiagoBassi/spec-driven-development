package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"

	gcsstorage "cloud.google.com/go/storage"

	"gcsgrep/internal/cli"
	"gcsgrep/internal/gcs"
	"gcsgrep/internal/location"
	"gcsgrep/internal/output"
	"gcsgrep/internal/search"
)

// run arma las piezas y traduce el resultado a exit code. Todo lo que se
// puede decidir con argv (cli, search.Compile, location.Parse) se decide
// antes de tocar GCS.
func run(args []string, stdout, stderr io.Writer) int {
	cfg, err := cli.Parse(args)
	if err != nil {
		return fail(stderr, err.Error())
	}
	re, err := search.Compile(cfg.Pattern, cfg.Extended, cfg.IgnoreCase)
	if err != nil {
		return fail(stderr, err.Error())
	}
	loc, err := location.Parse(cfg.Location)
	if err != nil {
		return fail(stderr, err.Error())
	}

	ctx := context.Background()

	// FR-35: se verifican las ADC siempre, antes de cualquier operación
	// contra GCS, aunque esté definido STORAGE_EMULATOR_HOST.
	creds, err := gcs.Credentials(ctx)
	if err != nil {
		return fail(stderr, err.Error())
	}
	client, err := gcs.NewClient(ctx, creds)
	if err != nil {
		return fail(stderr, err.Error())
	}
	defer client.Close()

	// Listar y leer son fases separadas (BR-3): el listado termina antes de
	// la primera lectura.
	var objects []string
	if loc.Mode == location.ModeObject {
		if _, err := gcs.Stat(ctx, client, loc.Bucket, loc.Path); err != nil {
			return fail(stderr, metadataErrorMsg(loc, err))
		}
		objects = []string{loc.Path}
	} else {
		names, err := gcs.List(ctx, client, loc.Bucket, loc.ListPrefix(), cfg.Max)
		if err != nil {
			return fail(stderr, listErrorMsg(cfg.Max, loc, err))
		}
		objects = names
	}

	printer := output.NewPrinter(stdout)
	scanner := search.NewScanner()
	matched := false
	failed := false

	for _, obj := range objects {
		err := readObject(ctx, client, loc.Bucket, obj, scanner, re, func(line []byte) error {
			matched = true
			return printer.Print(line)
		}, cfg.LineNumber)
		if err != nil {
			failed = true
			_ = output.Warn(stderr, fmt.Sprintf("gs://%s/%s: read error: %s", loc.Bucket, obj, err.Error()))
		}
	}

	switch {
	case failed:
		return 2
	case matched:
		return 0
	default:
		return 1
	}
}

// readObject abre un objeto y busca en él, línea por línea. onMatch recibe
// el registro ya formateado, listo para imprimir.
func readObject(ctx context.Context, client *gcsstorage.Client, bucket, object string, scanner *search.Scanner, re *regexp.Regexp, onMatch func([]byte) error, withLineNo bool) error {
	rc, err := gcs.Open(ctx, client, bucket, object)
	if err != nil {
		return err
	}
	defer rc.Close()
	return scanner.Scan(rc, re, func(lineNo int, text []byte, truncated bool) error {
		return onMatch(formatLine(bucket, object, withLineNo, lineNo, text, truncated))
	})
}

// formatLine arma gs://<bucket>/<objeto>:[<línea>:]<texto>[...]\n en un
// único []byte, para que output.Printer lo escriba con un solo Write.
func formatLine(bucket, object string, withLineNo bool, lineNo int, text []byte, truncated bool) []byte {
	var b bytes.Buffer
	b.Grow(len(bucket) + len(object) + len(text) + 16)
	b.WriteString("gs://")
	b.WriteString(bucket)
	b.WriteByte('/')
	b.WriteString(object)
	b.WriteByte(':')
	if withLineNo {
		b.WriteString(strconv.Itoa(lineNo))
		b.WriteByte(':')
	}
	b.Write(text)
	if truncated {
		b.WriteString("...")
	}
	b.WriteByte('\n')
	return b.Bytes()
}

func fail(stderr io.Writer, msg string) int {
	_ = output.Warn(stderr, msg)
	return 2
}

// metadataErrorMsg traduce el error de consultar un objeto puntual (BR-2,
// FR-30b provisional: FR-13a/FR-13b llegan en la Iteración 3).
func metadataErrorMsg(loc location.Location, err error) string {
	if gcs.IsAccessDenied(err) {
		return fmt.Sprintf("access denied: %s", loc.Raw)
	}
	return fmt.Sprintf("%s: metadata error: %s", loc.Raw, err.Error())
}

// listErrorMsg traduce el error de listar un bucket o prefijo (BR-2, BR-3,
// FR-30a provisional: FR-16a/FR-16b llegan en la Iteración 3).
func listErrorMsg(max int64, loc location.Location, err error) string {
	if gcs.IsAccessDenied(err) {
		return fmt.Sprintf("access denied: %s", loc.Raw)
	}
	var tooMany *gcs.ErrTooMany
	if errors.As(err, &tooMany) {
		return fmt.Sprintf("more than %d objects under %s; use --max <N> or --max unlimited", tooMany.Max, loc.Raw)
	}
	return fmt.Sprintf("%s: list error: %s", loc.Raw, err.Error())
}
