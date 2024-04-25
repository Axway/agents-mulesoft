package common

import "fmt"

const (
	AccessCode    = "accessCode"
	APIKey        = "apiKey"
	AppID         = "appID"
	AppName       = "appName"
	Authorization = "authorization"

	AttrAPIID          = "API ID"
	AttrAssetID        = "Asset ID"
	AttrAssetVersion   = "Asset Version"
	AttrChecksum       = "checksum"
	AttrProductVersion = "Product Version"

	ClientID            = "client-id"
	ClientIDEnforcement = "client-id-enforcement"
	ClientIDLabel       = "Client ID"
	ClientSecret        = "client-secret"
	ClientSecretLabel   = "Client Secret"
	ContractID          = "contractID"
	CredOrigin          = "credentialsOriginHasHttpBasicAuthenticationHeader"

	ClientCredDesc = "This API supports client credentials for authenticating all API requests"
	Description    = "description"
	ExternalOauth  = "external-oauth2-access-token-enforcement"
	Header         = "header"

	Oauth2Desc     = "This API supports OAuth 2.0 for authenticating all API requests"
	Oauth2OASType  = "oauth2"
	Oauth2RAMLType = "OAuth 2.0"
	Oauth2Name     = "o_auth_2"

	BasicAuthDesc     = "This API supports Basic Authentication for authenticating all API requests"
	BasicAuthName     = "basicAuth"
	BasicAuthScheme   = "basic"
	BasicAuthOASType  = "http"
	BasicAuthRAMLType = "Basic Authentication"

	Scopes    = "scopes"
	SLABased  = "sla-based"
	SlaTier   = "sla-tier"
	SLAActive = "ACTIVE"

	TierLabel = "SLA Tier"
	TokenURL  = "tokenUrl"

	AxwayAgentSLATierName        = "Axway Agent Tier"
	AxwayAgentSLATierDescription = "SLA Tier created for Axway Agent Provisioning purposes"
)

const (
	RateLimitingSLABasedPolicy    = "348742"
	BasicAuthSimplePolicy         = "341474"
	OAuth2MuleOauthProviderPolicy = "386072"
	HeaderRemovalPolicy           = "341469"
	MessageLoggingPolicy          = "345754"
	JWTValidationPolicy           = "385922"
	RateLimitingPolicy            = "348741"
	BasicAuthLDAPPolicy           = "345188"
	ClientIDEnforcementPolicy     = "341473"
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
