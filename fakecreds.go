package main

import (
	"log"
	"os"
)

// Create a fake creds.json file for testing
func CreateFakeCreds() {
	c := "{\n  \"type\": \"service_account\",\n  \"project_id\":  \"test\",\n  \"private_key_id\":  \"test\",\n  \"private_key\":  \"test\",\n  \"client_email\":  \"test\",\n  \"client_id\":  \"test\",\n  \"auth_uri\":  \"test\",\n  \"token_uri\":  \"test\",\n  \"auth_provider_x509_cert_url\":  \"test\",\n  \"client_x509_cert_url\":  \"test\"\n}"
	f, err := os.Create("creds.json")
	if err != nil {
		log.Fatal(err)
	}
	_, err = f.WriteString(c)
	if err != nil {
		log.Fatal(err)
	}
}

// Remove the file after successful testing
func RemoveFakeCreds() {
	e := os.Remove("creds.json")
	if e != nil {
		log.Fatal(e)
	}
}
