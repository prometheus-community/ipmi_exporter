package vault

import (
	"fmt"
	"log"

	"github.com/alecthomas/kingpin/v2"
)

var (
	vaultAddress *string
	tokenFile    *string
)

func AddFlags() *string {
	vaultType := kingpin.Flag("vault.type", "Specify the type of vault (default: none).").Enum(
		"hashiCorp",
		"aws",
		"azure",
		"google",
		"cyberArk",
		"lastPass",
		"bitwarden",
		"keePass",
		"thycotic")
	addHashiCorpFlags()
	// Add for other vaults (e.g., AWS, GCP)

	return vaultType
}

// VaultClient defines the interface for interacting with a vault to get credentials.
type VaultClient interface {
	GetCredentials(target string) (username string, password string, err error)
}

// NewVaultClient is a factory function that returns the appropriate VaultClient.
func NewVaultClient(VaultType string) (VaultClient, error) {
	switch VaultType {
	case "hashiCorp":
		return NewHashiCorpVaultClient(*vaultAddress, *tokenFile)
	// Add cases for other vaults (e.g., AWS, GCP)
	default:
		return nil, fmt.Errorf("unsupported vault type: %s", VaultType)
	}
}

// LogError is a utility function for logging errors.
func LogError(err error) {
	if err != nil {
		log.Printf("Error: %v", err)
	}
}
