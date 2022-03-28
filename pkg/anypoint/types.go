package anypoint

import (
	"time"
)

// CurrentUser -
type CurrentUser struct {
	User User `json:"user"`
}

// User -
type User struct {
	Email                 string                  `json:"email"`
	FirstName             string                  `json:"firstName"`
	ID                    string                  `json:"id"`
	IdentityType          string                  `json:"identityType"`
	LastName              string                  `json:"lastName"`
	Organization          Organization            `json:"organization"`
	MemberOfOrganizations []MemberOfOrganizations `json:"memberOfOrganizations"`
	Username              string                  `json:"username"`
}

// Organization -
type Organization struct {
	Domain string `json:"domain"`
	ID     string `json:"id"`
	Name   string `json:"name"`
}

// MemberOfOrganizations -
type MemberOfOrganizations struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Environment -
type Environment struct {
	ClientID        string `json:"clientId"`
	EnvironmentType string `json:"type"`
	ID              string `json:"id"`
	IsProduction    bool   `json:"isProduction"`
	Name            string `json:"name"`
	OrganizationID  string `json:"organizationId"`
}

// EnvironmentSearch -
type EnvironmentSearch struct {
	Data  []Environment `json:"data"`
	Total int           `json:"total"`
}

// AssetSearch -
type AssetSearch struct {
	Assets []Asset `json:"assets"`
	Total  int     `json:"total"`
}

// Asset -
type Asset struct {
	APIs                 []API  `json:"apis"`
	AssetID              string `json:"assetId"`
	Audit                Audit  `json:"audit"`
	AutodiscoveryAPIName string `json:"autodiscoveryApiName"`
	ExchangeAssetName    string `json:"exchangeAssetName"`
	GroupID              string `json:"groupId"`
	ID                   int64  `json:"id"`
	MasterOrganizationID string `json:"masterOrganizationId"`
	Name                 string `json:"name"`
	OrganizationID       string `json:"organizationId"`
	TotalAPIs            int    `json:"totalApis"`
}

// API -
type API struct {
	ActiveContractsCount      int      `json:"activeContractsCount"`
	AssetID                   string   `json:"assetId"`
	AssetVersion              string   `json:"assetVersion"`
	Audit                     Audit    `json:"audit"`
	AutodiscoveryInstanceName string   `json:"autodiscoveryInstanceName"`
	Deprecated                bool     `json:"deprecated"`
	Description               string   `json:"description"`
	EndpointURI               string   `json:"endpointUri"`
	EnvironmentID             string   `json:"environmentId"`
	GroupID                   string   `json:"groupId"`
	ID                        int64    `json:"id"`
	InstanceLabel             string   `json:"instanceLabel"`
	IsPublic                  bool     `json:"isPublic"`
	MasterOrganizationID      string   `json:"masterOrganizationId"`
	Order                     int      `json:"order"`
	OrganizationID            string   `json:"organizationId"`
	Pinned                    bool     `json:"pinned"`
	ProductVersion            string   `json:"productVersion"`
	Tags                      []string `json:"tags"`
}

// Policy -
type Policy struct {
	APIID                int64       `json:"apiId"`
	Audit                Audit       `json:"audit"`
	Configuration        interface{} `json:"configuration"`
	ID                   int64       `json:"id"`
	MasterOrganizationID string      `json:"masterOrganizationId"`
	Order                int         `json:"order"`
	OrganizationID       string      `json:"organizationId"`
	PointCutData         interface{} `json:"pointCutData"`
	PolicyID             int         `json:"policyId"`
	PolicyTemplateID     string      `json:"policyTemplateId"`
	Template             Template    `json:"template"`
	Type                 string      `json:"type"`
	Version              int64       `json:"version"`
}

type Template struct {
	AssetID      string `json:"assetId"`
	AssetVersion string `json:"assetVersion"`
	GroupID      string `json:"groupId"`
}

type Policies struct {
	Policies []Policy `json:"policies"`
}

// Audit -
type Audit struct {
	Created AuditEntry `json:"created"`
	Updated AuditEntry `json:"updated"`
}

// AuditEntry -
type AuditEntry struct {
	Date time.Time `json:"date"`
}

// ExchangeAsset - https://anypoint.mulesoft.com/exchange/portals/anypoint-platform/f1e97bc6-315a-4490-82a7-23abe036327a.anypoint-platform/exchange-experience-api/minor/2.0/console/type/%231186/
type ExchangeAsset struct {
	AssetID      string                `json:"assetId"`
	AssetType    string                `json:"type"`
	Categories   []ExchangeCategory    `json:"categories"`
	CreatedAt    time.Time             `json:"createdAt"`
	Description  string                `json:"description"`
	Files        []ExchangeFile        `json:"files"`
	GroupID      string                `json:"groupId"`
	Icon         string                `json:"icon"`
	ID           string                `json:"id"`
	Instances    []ExchangeAPIInstance `json:"instances"`
	Labels       []string              `json:"labels"`
	MinorVersion string                `json:"minorVersion"`
	ModifiedAt   time.Time             `json:"modifiedAt"`
	Name         string                `json:"name"`
	Public       bool                  `json:"isPublic"`
	Snapshot     bool                  `json:"isSnapshot"`
	Status       string                `json:"status"`
	Version      string                `json:"version"`
	VersionGroup string                `json:"versionGroup"`
}

// ExchangeCategory -
type ExchangeCategory struct {
	DisplayName string   `json:"displayName"`
	Key         string   `json:"key"`
	Value       []string `json:"value"`
}

// ExchangeAPIInstance - https://anypoint.mulesoft.com/exchange/portals/anypoint-platform/f1e97bc6-315a-4490-82a7-23abe036327a.anypoint-platform/exchange-experience-api/minor/2.0/console/method/%231972/
type ExchangeAPIInstance struct {
	AssetID           string    `json:"assetId"`
	AssetName         string    `json:"assetName"`
	CreatedBy         string    `json:"createdBy"`
	CreatedDate       time.Time `json:"createdDate"`
	EndpointURI       string    `json:"endpointUri"`
	EnvironmentID     string    `json:"environmentId"`
	Fullname          string    `json:"fullname"`
	GroupID           string    `json:"groupId"`
	ID                string    `json:"id"`
	InstanceType      string    `json:"type"`
	IsPublic          bool      `json:"isPublic"`
	Name              string    `json:"name"`
	ProductAPIVersion string    `json:"productApiVersion"`
	ProviderID        string    `json:"providerId"`
	UpdatedDate       time.Time `json:"updatedDate"`
	Version           string    `json:"version"`
}

// ExchangeFile -
type ExchangeFile struct {
	Classifier   string    `json:"classifier"`
	CreatedDate  time.Time `json:"createdDate"`
	DownloadURL  string    `json:"downloadURL"`
	ExternalLink string    `json:"externalLink"`
	Generated    bool      `json:"isGenerated"`
	MainFile     string    `json:"mainFile"`
	MD5          string    `json:"md5"`
	Packaging    string    `json:"packaging"`
	SHA1         string    `json:"sha1"`
}

// AnalyticsEvent -
type AnalyticsEvent struct {
	APIID              string    `json:"API ID"`
	APIName            string    `json:"API Name"`
	APIVersionID       string    `json:"API Version ID"`
	APIVersionName     string    `json:"API Version Name"`
	ApplicationName    string    `json:"Application Name"`
	Application        string    `json:"Application"`
	Browser            string    `json:"Browser"`
	City               string    `json:"City"`
	ClientIP           string    `json:"Client IP"`
	Continent          string    `json:"Continent"`
	Country            string    `json:"Country"`
	HardwarePlatform   string    `json:"Hardware Platform"`
	MessageID          string    `json:"Message ID"`
	OSFamily           string    `json:"OS Family"`
	OSMajorVersion     string    `json:"OS Major Version"`
	OSMinorVersion     string    `json:"OS Minor Version"`
	OSVersion          string    `json:"OS Version"`
	PostalCode         string    `json:"Postal Code"`
	RequestOutcome     string    `json:"Request Outcome"`
	RequestSize        int       `json:"Request Size"`
	ResourcePath       string    `json:"Resource Path"`
	ResponseSize       int       `json:"Response Size"`
	ResponseTime       int       `json:"Response Time"`
	StatusCode         int       `json:"Status Code"`
	Timestamp          time.Time `json:"Timestamp"`
	Timezone           string    `json:"Timezone"`
	UserAgentName      string    `json:"User Agent Name"`
	UserAgentVersion   string    `json:"User Agent Version"`
	Verb               string    `json:"Verb"`
	ViolatedPolicyName string    `json:"Violated Policy Name"`
}

const (
	AccessCode       = "accessCode"
	APIKey           = "apiKey"
	AppName          = "appName"
	Authorization    = "authorization"
	ClientID         = "client-id-enforcement"
	ClientIDProp     = "client_id"
	ClientSecretProp = "client_secret"
	CredOrigin       = "credentialsOriginHasHttpBasicAuthenticationHeader"
	DescClienCred    = "Provided as: client_id:<INSERT_VALID_CLIENTID_HERE> \n\n client_secret:<INSERT_VALID_SECRET_HERE>\n\n"
	DescOauth2       = "This API supports OAuth 2.0 for authenticating all API requests"
	Description      = "description"
	ExternalOauth    = "external-oauth2-access-token-enforcement"
	Header           = "header"
	Oauth2           = "oauth2"
	Scopes           = "scopes"
	SLAAuth          = "sla-based"
	TierLabel        = "SLA Tier"
	TokenURL         = "tokenUrl"
)

type Application struct {
	APIEndpoints bool   `json:"apiEndpoints,omitempty"`
	ClientID     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
	Description  string `json:"description"`
	ID           int64  `json:"id"`
	Name         string `json:"name"`
}

type AppRequestBody struct {
	Description string `json:"description"`
	Name        string `json:"name"`
}

type Contract struct {
	AcceptedTerms   bool   `json:"acceptedTerms"`
	APIID           string `json:"apiId"`
	AssetID         string `json:"assetId"`
	EnvironmentID   string `json:"environmentId"`
	GroupID         string `json:"groupId"`
	OrganizationID  string `json:"organizationId"`
	RequestedTierID int64  `json:"requestedTierId,omitempty"`
	Version         string `json:"version"`
	VersionGroup    string `json:"versionGroup"`
}

type ContractResponse struct {
	API           API    `json:"api"`
	ApplicationID int    `json:"applicationId"`
	ClientID      string `json:"clientId"`
	ClientSecret  string `json:"clientSecret"`
	ID            int    `json:"id"`
	Status        string `json:"status"`
}

type Tiers struct {
	Total int       `json:"total"`
	Tiers []SLATier `json:"tiers"`
}

type SLATier struct {
	Description interface{} `json:"description"`
	ID          int         `json:"id"`
	Limits      []Limits    `json:"limits"`
	Name        string      `json:"name"`
	Status      string      `json:"status"`
}

type Limits struct {
	MaximumRequests          interface{} `json:"maximumRequests,omitempty"`
	TimePeriodInMilliseconds int         `json:"timePeriodInMilliseconds,omitempty"`
	Visible                  bool        `json:"visible,omitempty"`
}
