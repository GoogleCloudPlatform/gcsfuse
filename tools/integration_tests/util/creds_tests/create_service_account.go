package creds_tests

import (
	"context"
	"fmt"
	"io"
	"log"

	iam "google.golang.org/api/iam/v1"
)

// createServiceAccount creates a service account.
func createServiceAccount(w io.Writer, projectID, name, displayName string) (*iam.ServiceAccount, error) {
	ctx := context.Background()
	service, err := iam.NewService(ctx)
	if err != nil {
		return nil, fmt.Errorf("iam.NewService: %w", err)
	}

	request := &iam.CreateServiceAccountRequest{
		AccountId: name,
		ServiceAccount: &iam.ServiceAccount{
			DisplayName: displayName,
		},
	}
	account, err := service.Projects.ServiceAccounts.Create("projects/"+projectID, request).Do()
	if err != nil {
		log.Printf("Projects.ServiceAccounts.Create: %w", err)
		return nil, fmt.Errorf("Projects.ServiceAccounts.Create: %w", err)
	}
	fmt.Fprintf(w, "Created service account: %v", account)
	return account, nil
}
