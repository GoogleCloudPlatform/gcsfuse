package main

import (
	"os"
)

// Create a fake creds.json file for testing
func CreateFakeCreds(creds string) (err error) {
	content := "{\n  \"type\": \"service_account\"," +
		"\n  \"project_id\":  \"project_id\"," +
		"\n  \"private_key_id\":  \"private_key_id\"," +
		"\n  \"private_key\":  \"private_key\"," +
		"\n  \"client_email\":  \"client_email\"," +
		"\n  \"client_id\":  \"client_id\"," +
		"\n  \"auth_uri\":  \"auth_uri\"," +
		"\n  \"token_uri\":  \"token_uri\"," +
		"\n  \"auth_provider_x509_cert_url\":  \"auth_provider_x509_cert_url\"," +
		"\n  \"client_x509_cert_url\":  \"client_x509_cert_url\"\n}"

	f, err := os.Create(creds)
	if err != nil {
		return
	}

	_, err = f.WriteString(content)

	return
}

// Remove the file after successful testing
func RemoveFakeCreds(creds string) (err error) {
	err = os.Remove(creds)
	return
}
