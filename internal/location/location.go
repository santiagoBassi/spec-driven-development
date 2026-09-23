// Package location interprets a gs://bucket/[path] argument (FR-10 to FR-14).
package location

import (
	"strings"
)

// Mode says what a location refers to.
type Mode int

const (
	// Prefix is a whole bucket (empty Path) or everything under Path, which
	// ends in "/". The search is recursive.
	Prefix Mode = iota + 1
	// Object is a single object named exactly Path.
	Object
)

// Location is a parsed gs:// argument.
type Location struct {
	// Raw is the argument as the user typed it; messages quote it verbatim.
	Raw    string
	Bucket string
	Path   string
	Mode   Mode
}

// InvalidError reports a location that is not gs://<bucket>/[path] (FR-14).
type InvalidError struct{ Raw string }

func (e *InvalidError) Error() string {
	return `invalid location: "` + e.Raw + `"`
}

// Parse accepts gs://<bucket>/, gs://<bucket>/<prefix>/ and gs://<bucket>/<object>.
// The scheme, a bucket name and the "/" that follows it are all mandatory, so
// "gs://bucket" is rejected: the trailing "/" is what decides the mode.
func Parse(raw string) (Location, error) {
	rest, ok := strings.CutPrefix(raw, "gs://")
	if !ok {
		return Location{}, &InvalidError{Raw: raw}
	}
	bucket, path, found := strings.Cut(rest, "/")
	if !found || bucket == "" {
		return Location{}, &InvalidError{Raw: raw}
	}
	mode := Object
	if path == "" || strings.HasSuffix(path, "/") {
		mode = Prefix
	}
	return Location{Raw: raw, Bucket: bucket, Path: path, Mode: mode}, nil
}

// URI returns the gs:// URI of an object in this location's bucket.
func (l Location) URI(object string) string {
	return "gs://" + l.Bucket + "/" + object
}
