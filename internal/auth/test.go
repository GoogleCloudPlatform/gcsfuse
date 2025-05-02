package auth

import (
	"context"
	"fmt"
	"log"
	"os"

	"cloud.google.com/go/storage/control/apiv2"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
)

func main() {
	ctx := context.Background()

	// Path to your service account key file
	keyFile := "/tmp/sa.key.json"

	// Universe domain
	universeDomain := "apis-tpczero.goog" // Replace with actual universe domain if needed

	contents, err := os.ReadFile(keyFile)
	if err != nil {
		fmt.Println("ReadFile(%q): %w", keyFile, err)
	}

	jwtConfig, err := google.JWTConfigFromJSON(contents, "https://www.googleapis.com/auth/devstorage.full_control")
	if err != nil {
		fmt.Println("JWTConfigFromJSON: %w", err)
	}

	creds, err := google.CredentialsFromJSON(ctx, contents, "https://www.googleapis.com/auth/devstorage.full_control")
	if err != nil {
		fmt.Println(("CredentialsFromJSON(): %w", err)
	}

	domain, err := creds.GetUniverseDomain()
	if err != nil {
		fmt.Println(("GetUniverseDomain(): %w", err)
	}
	if err != nil {
		fmt.Println("JWTConfigFromJSON: %w", err
	}

	var ts oauth2.TokenSource
	if domain != universeDomainDefault {
		ts, err = google.JWTAccessTokenSourceWithScope(contents, "https://www.googleapis.com/auth/devstorage.full_control")
		if err != nil {
			return nil, fmt.Errorf("JWTAccessTokenSourceWithScope: %w", err)
		}
	} else {
		ts = jwtConfig.TokenSource(ctx)
	}
	// Create the Storage Control client with JWT and custom endpoint
	client, err := control.NewStorageControlClient(ctx,
		option.WithTokenSource(ts),
		option.WithUniverseDomain(universeDomain),
	)
	if err != nil {
		log.Fatalf("failed to create storage control client: %v", err)
	}
	defer client.Close()
}

// Helper to read key file as string
func readKeyFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("unable to read key file: %v", err)
	}
	return string(data)
}
