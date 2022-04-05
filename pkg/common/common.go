package common

import "fmt"

const (
	AttrAPIID          = "API ID"
	AttrAssetID        = "Asset ID"
	AttrAssetVersion   = "Asset Version"
	AttrChecksum       = "checksum"
	AttrProductVersion = "Product Version"
	ClientID           = "client-id"
	ClientSecret       = "client-secret"
	ClientIDLabel      = "Client ID"
	ClientSecretLabel  = "Client Secret"
	SlaTier            = "sla-tier"
)

// FormatAPICacheKey ensure consistent naming of the cache key for an API.
func FormatAPICacheKey(apiID, stageName string) string {
	return fmt.Sprintf("%s-%s", apiID, stageName)
}

type PolicyDetail struct {
	Policy     string
	IsSLABased bool
	APIId      string
}
