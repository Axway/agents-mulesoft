package common

import "fmt"

const (
	AccessCode          = "accessCode"
	APIKey              = "apiKey"
	AppID               = "appID"
	AppName             = "appName"
	AttrAPIID           = "API ID"
	AttrAssetID         = "Asset ID"
	AttrAssetVersion    = "Asset Version"
	AttrChecksum        = "checksum"
	AttrProductVersion  = "Product Version"
	Authorization       = "authorization"
	ClientID            = "client-id"
	ClientIDEnforcement = "client-id-enforcement"
	ClientIDLabel       = "Client ID"
	ClientSecret        = "client-secret"
	ClientSecretLabel   = "Client Secret"
	ContractID          = "contractID"
	CredOrigin          = "credentialsOriginHasHttpBasicAuthenticationHeader"
	DescClientCred      = "Provided as: client_id:<INSERT_VALID_CLIENTID_HERE> \n\n client_secret:<INSERT_VALID_SECRET_HERE>\n\n"
	DescOauth2          = "This API supports OAuth 2.0 for authenticating all API requests"
	Description         = "description"
	ExternalOauth       = "external-oauth2-access-token-enforcement"
	Header              = "header"
	Oauth2              = "oauth2"
	Scopes              = "scopes"
	SLABased            = "sla-based"
	SlaTier             = "sla-tier"
	TierLabel           = "SLA Tier"
	TokenURL            = "tokenUrl"
)

// FormatAPICacheKey ensure consistent naming of the cache key for an API.
func FormatAPICacheKey(apiID, stageName string) string {
	return fmt.Sprintf("%s-%s", apiID, stageName)
}

// PolicyDetail represents a policy
type PolicyDetail struct {
	Policy     string
	IsSLABased bool
	APIId      string
}
