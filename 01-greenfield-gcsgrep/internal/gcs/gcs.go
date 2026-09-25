// Package gcs is the only place that talks to Google Cloud Storage: ADC, listing
// with a cap, object metadata and streaming reads. It does not know what a match is.
//
// Everything here is read-only (BR-1): the token is requested with the
// read-only scope, and no write method of the client is ever called.
package gcs

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"

	"cloud.google.com/go/storage"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

// ErrNoCredentials means there are no Application Default Credentials (FR-35).
var ErrNoCredentials = errors.New("no Application Default Credentials found (run: gcloud auth application-default login)")

// ErrAccessDenied means GCS answered 403 to a listing or a metadata request
// (BR-2). The caller reports it with the location, which this package does not have.
var ErrAccessDenied = errors.New("access denied")

// TooManyObjectsError means a listing has more objects than the cap (BR-3).
type TooManyObjectsError struct{ Max int }

func (e *TooManyObjectsError) Error() string { return "too many objects" }

// Credentials looks up the Application Default Credentials, read-only scope.
// It is always called before creating a client: with STORAGE_EMULATOR_HOST set
// the client would not ask for credentials at all (FR-35).
func Credentials(ctx context.Context) (*google.Credentials, error) {
	creds, err := google.FindDefaultCredentials(ctx, storage.ScopeReadOnly)
	if err != nil {
		return nil, ErrNoCredentials
	}
	return creds, nil
}

// Client reads from GCS.
type Client struct {
	sc *storage.Client
}

// NewClient creates a client that never retries (FR-31) and reads objects
// through the JSON API.
func NewClient(ctx context.Context, creds *google.Credentials) (*Client, error) {
	opts := []option.ClientOption{storage.WithJSONReads()}
	// With an emulator the client is built without authentication, and it
	// refuses credentials on top of that. They were checked already.
	if os.Getenv("STORAGE_EMULATOR_HOST") == "" {
		opts = append(opts, option.WithCredentials(creds))
	}
	sc, err := storage.NewClient(ctx, opts...)
	if err != nil {
		return nil, err
	}
	sc.SetRetry(storage.WithPolicy(storage.RetryNever))
	return &Client{sc: sc}, nil
}

// Close releases the client.
func (c *Client) Close() error { return c.sc.Close() }

// List returns the names of the objects in bucket whose name starts with
// prefix, at any depth. If max > 0 and there are more than max objects it stops
// as soon as it sees the max+1-th, without asking for another page, and returns
// *TooManyObjectsError: the caller has not read anything yet. max == 0 means no cap.
func (c *Client) List(ctx context.Context, bucket, prefix string, max int) ([]string, error) {
	q := &storage.Query{Prefix: prefix}
	if err := q.SetAttrSelection([]string{"Name"}); err != nil {
		return nil, err
	}
	it := c.sc.Bucket(bucket).Objects(ctx, q)
	var names []string
	for {
		attrs, err := it.Next()
		if errors.Is(err, iterator.Done) {
			return names, nil
		}
		if err != nil {
			return nil, classify(err)
		}
		if max > 0 && len(names) == max {
			return nil, &TooManyObjectsError{Max: max}
		}
		names = append(names, attrs.Name)
	}
}

// Stat checks that an object can be reached, without listing anything.
func (c *Client) Stat(ctx context.Context, bucket, object string) error {
	_, err := c.sc.Bucket(bucket).Object(object).Attrs(ctx)
	return classify(err)
}

// Open streams an object's content. The caller closes it.
func (c *Client) Open(ctx context.Context, bucket, object string) (io.ReadCloser, error) {
	return c.sc.Bucket(bucket).Object(object).NewReader(ctx)
}

func classify(err error) error {
	var gerr *googleapi.Error
	if errors.As(err, &gerr) && gerr.Code == http.StatusForbidden {
		return ErrAccessDenied
	}
	return err
}
