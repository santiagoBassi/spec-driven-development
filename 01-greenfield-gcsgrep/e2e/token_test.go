package e2e

import (
	"context"
	"os"

	"golang.org/x/oauth2/google"
)

const readOnlyScope = "https://www.googleapis.com/auth/devstorage.read_only"

// accessToken returns an OAuth token, read-only scope, for an ADC file.
func accessToken(adcFile string) (string, error) {
	data, err := os.ReadFile(adcFile)
	if err != nil {
		return "", err
	}
	creds, err := google.CredentialsFromJSON(context.Background(), data, readOnlyScope)
	if err != nil {
		return "", err
	}
	tok, err := creds.TokenSource.Token()
	if err != nil {
		return "", err
	}
	return tok.AccessToken, nil
}
