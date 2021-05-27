package common

import (
	"fmt"
	"net/url"
	"strconv"
)

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

// FormatEndpointKey key to use to look up endpoints for a service
func FormatEndpointKey(assetID, productVersion, assetVersion string) string {
	return fmt.Sprintf("%s-%s-%s", assetID, productVersion, assetVersion)
}

// ParseEndpoint parses an endpoint
func ParseEndpoint(endpointURL string) (host, basePath, scheme string, port int32, err error) {
	endpoint, err := url.Parse(endpointURL)
	scheme = endpoint.Scheme
	if err != nil {
		return "", "", "", 0, err
	}

	basePath = ""
	if endpoint.Path == "" {
		basePath = "/"
	} else {
		basePath = endpoint.Path
	}

	var portInt int
	if scheme == "http" {
		portInt = 80
	} else {
		portInt = 443
	}

	strPort := endpoint.Port()
	if strPort != "" {
		portInt, err = strconv.Atoi(strPort)
		if err != nil {
			return "", "", "", 0, err
		}
	}

	port = int32(portInt)

	return endpoint.Hostname(), basePath, scheme, port, nil
}
