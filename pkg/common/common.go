package common

import "fmt"

const (
	AttrAPIID          = "API ID"
	AttrAssetID        = "Asset ID"
	AttrAssetVersion   = "Asset Version"
	AttrChecksum       = "checksum"
	AttrProductVersion = "Product Version"
)

// FormatAPICacheKey ensure consistent naming of the cache key for an API.
func FormatAPICacheKey(apiID, stageName string) string {
	return fmt.Sprintf("%s-%s", apiID, stageName)
}

// FormatRevisionName format for an API Service Revision
func FormatRevisionName(assetVersion, apiID string) string {
	return fmt.Sprintf("%s-%s", assetVersion, apiID)
}
