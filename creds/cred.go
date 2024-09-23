package creds

import (
	"fmt"
	"log"

	"github.com/hashicorp/vault/api"
)

func getVaultClient() (*api.Client, error) {
	config := api.DefaultConfig()
	config.Address = "http://192.168.122.79:8200" //"http://192.168.122.79:8200" // Ensure VAULT_ADDR is set

	client, err := api.NewClient(config)
	if err != nil {
		return nil, err
	}

	// Set the Vault token from environment variables
	client.SetToken("hvs.CAESIB0uncyw0pyNtjhOf2YMN8mGJfbhnTh6Y_doeqRDJwujGh4KHGh2cy5USFZXdWhRSnZNdmlUQ2VPcjlwcEswSDQ")
	return client, nil
}

func getCredentialsFromVault(client *api.Client, targetAddress string) (map[string]interface{}, error) {
	// Construct the Vault path based on the target's address
	vaultPath := fmt.Sprintf("kv/%s", targetAddress)

	// Access the path and get the secrets
	secret, err := client.Logical().Read(vaultPath)
	if err != nil {
		return nil, err
	}

	if secret == nil || secret.Data == nil {
		return nil, fmt.Errorf("no data found for target %s", targetAddress)
	}

	return secret.Data, nil
}

func GetCreds(target string) (username string, password string, err error) {

	client, err := getVaultClient()
	if err != nil {
		log.Fatalf("Error creating Vault client: %v", err)
	}

	creds, err := getCredentialsFromVault(client, target)
	if err != nil {
		log.Fatalf("Error retrieving credentials: %v", err)
	}

	return creds["username"].(string), creds["password"].(string), nil
}
