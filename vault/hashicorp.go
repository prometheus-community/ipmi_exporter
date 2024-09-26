package vault

import (
	"bytes"
	"errors"
	"fmt"
	"os"

	"github.com/alecthomas/kingpin/v2"
	"github.com/hashicorp/vault/api"
)

// HashiCorpVaultClient implements the VaultClient interface for HashiCorp Vault.
type HashiCorpVaultClient struct {
	client *api.Client
}

// Adds the Hashicorp Flags
func addHashiCorpFlags() {
	vaultAddress = kingpin.Flag("ip", "IP address of the Vault").String()
	tokenFile = kingpin.Flag("token-file", "Path to the file containing the Vault token").String()
}

// NewHashiCorpVaultClient creates a new Vault client for HashiCorp Vault.
func NewHashiCorpVaultClient(vaultAddress, tokenFile string) (VaultClient, error) {

	if vaultAddress == "" || tokenFile == "" {
		return nil, errors.New("both --ip and --token-file are required when using hashicorp vault")
	}

	// Read the token from the specified file
	token, err := os.ReadFile(tokenFile)
	if err != nil {
		return nil, err
	}

	// Convert byte slices to strings and trim whitespace
	vaultToken := string(bytes.TrimSpace(token))

	config := api.DefaultConfig()
	config.Address = vaultAddress

	client, err := api.NewClient(config)
	if err != nil {
		return nil, err
	}

	client.SetToken(vaultToken)
	return &HashiCorpVaultClient{client: client}, nil
}

// GetCredentials retrieves credentials from HashiCorp Vault.
func (h *HashiCorpVaultClient) GetCredentials(target string) (string, string, error) {
	vaultPath := fmt.Sprintf("kv/%s", target)

	secret, err := h.client.Logical().Read(vaultPath)
	if err != nil {
		return "", "", err
	}

	if secret == nil || secret.Data == nil {
		return "", "", fmt.Errorf("no data found for target %s", target)
	}

	username, ok := secret.Data["username"].(string)
	if !ok {
		return "", "", fmt.Errorf("username not found for target %s", target)
	}

	password, ok := secret.Data["password"].(string)
	if !ok {
		return "", "", fmt.Errorf("password not found for target %s", target)
	}

	return username, password, nil
}
