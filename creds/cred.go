package vaultlib

import (
	"fmt"
	"log"

	"github.com/hashicorp/vault/api"
)

// VaultClient defines the interface for interacting with a vault to get credentials.
type VaultClient interface {
	GetCredentials(target string) (username string, password string, err error)
}

// HashiCorpVaultClient implements the VaultClient interface for HashiCorp Vault.
type HashiCorpVaultClient struct {
	client *api.Client
}

// NewHashiCorpVaultClient creates a new Vault client for HashiCorp Vault.
func NewHashiCorpVaultClient(address, token string) (VaultClient, error) {
	config := api.DefaultConfig()
	config.Address = address

	client, err := api.NewClient(config)
	if err != nil {
		return nil, err
	}

	client.SetToken(token)
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

// NewVaultClient is a factory function that returns the appropriate VaultClient.
func NewVaultClient(vaultType, address, token string) (VaultClient, error) {
	switch vaultType {
	case "hashicorp":
		return NewHashiCorpVaultClient(address, token)
	// Add cases for other vaults (e.g., AWS, GCP)
	default:
		return nil, fmt.Errorf("unsupported vault type: %s", vaultType)
	}
}

// LogError is a utility function for logging errors.
func LogError(err error) {
	if err != nil {
		log.Printf("Error: %v", err)
	}
}
