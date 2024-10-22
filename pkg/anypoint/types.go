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
	ActiveContractsCount      int       `json:"activeContractsCount"`
	AssetID                   string    `json:"assetId"`
	AssetVersion              string    `json:"assetVersion"`
	Audit                     Audit     `json:"audit"`
	AutodiscoveryInstanceName string    `json:"autodiscoveryInstanceName"`
	Deprecated                bool      `json:"deprecated"`
	Description               string    `json:"description"`
	EndpointURI               string    `json:"endpointUri"`
	Endpoint                  *Endpoint `json:"endpoint,omitempty"`
	EnvironmentID             string    `json:"environmentId"`
	GroupID                   string    `json:"groupId"`
	ID                        int       `json:"id"`
	InstanceLabel             string    `json:"instanceLabel"`
	IsPublic                  bool      `json:"isPublic"`
	MasterOrganizationID      string    `json:"masterOrganizationId"`
	Order                     int       `json:"order"`
	OrganizationID            string    `json:"organizationId"`
	Pinned                    bool      `json:"pinned"`
	ProductVersion            string    `json:"productVersion"`
	Tags                      []string  `json:"tags"`
}

type Endpoint struct {
	ProxyURI string `json:"proxyUri"`
}

// Policy -
type Policy struct {
	Configuration     interface{} `json:"configuration,omitempty"`
	ConfigurationData interface{} `json:"configurationData,omitempty"`
	PolicyTemplateID  string      `json:"policyTemplateId,omitempty"`
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

// APIMonitoringMetric -
type APIMonitoringMetric struct {
	Time   time.Time
	Events []APISummaryMetricEvent
}

type DataFile struct {
	ID   string    `json:"id"`
	Time time.Time `json:"time"`
	Size int       `json:"size"`
}

type DataFileResources struct {
	Resources []DataFile `json:"resources"`
}

type APISummaryMetricEvent struct {
	APIName            string `json:"api_name"`
	APIVersion         string `json:"api_version"`
	APIVersionID       string `json:"api_version_id"`
	ClientID           string `json:"client_id"`
	Method             string `json:"method"`
	StatusCode         string `json:"status_code"`
	ResponseSizeCount  int    `json:"response_size.count"`
	ResponseSizeMax    int    `json:"response_size.max"`
	ResponseSizeMin    int    `json:"response_size.min"`
	ResponseSizeSos    int    `json:"response_size.sos"`
	ResponseSizeSum    int    `json:"response_size.sum"`
	ResponseTimeCount  int    `json:"response_time.count"`
	ResponseTimeMax    int    `json:"response_time.max"`
	ResponseTimeMin    int    `json:"response_time.min"`
	ResponseTimeSos    int    `json:"response_time.sos"`
	ResponseTimeSum    int    `json:"response_time.sum"`
	RequestSizeCount   int    `json:"request_size.count"`
	RequestSizeMax     int    `json:"request_size.max"`
	RequestSizeMin     int    `json:"request_size.min"`
	RequestSizeSos     int    `json:"request_size.sos"`
	RequestSizeSum     int    `json:"request_size.sum"`
	RequestDisposition string `json:"request_disposition"`
}

type MetricData struct {
	Format   string                 `json:"format"`
	Time     int64                  `json:"time"`
	Type     string                 `json:"type"`
	Metadata map[string]interface{} `json:"metadata"`
	Commons  map[string]interface{} `json:"commons"`
	Events   []APISummaryMetricEvent
}

type Application struct {
	APIEndpoints bool   `json:"apiEndpoints,omitempty"`
	ClientID     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
	Description  string `json:"description"`
	ID           int    `json:"id"`
	Name         string `json:"name"`
}

type AppRequestBody struct {
	Description string `json:"description"`
	Name        string `json:"name"`
}

type Contract struct {
	AcceptedTerms   bool   `json:"acceptedTerms"`
	ApiID           string `json:"apiId"`
	AssetID         string `json:"assetId"`
	EnvironmentID   string `json:"environmentId"`
	GroupID         string `json:"groupId"`
	ID              int    `json:"id"`
	OrganizationID  string `json:"organizationId"`
	RequestedTierID int    `json:"requestedTierId,omitempty"`
	Status          string `json:"status"`
	Version         string `json:"version"`
	VersionGroup    string `json:"versionGroup"`
}

type Tiers struct {
	Total int       `json:"total"`
	Tiers []SLATier `json:"tiers"`
}

type SLATier struct {
	Audit                *Audit   `json:"audit,omitempty"`
	MasterOrganizationID *string  `json:"masterOrganizationId,omitempty"`
	OrganizationID       *string  `json:"organizationId,omitempty"`
	Description          *string  `json:"description,omitempty"`
	ID                   *int     `json:"id,omitempty"`
	Limits               []Limits `json:"limits"`
	Name                 string   `json:"name"`
	Status               string   `json:"status"`
	AutoApprove          bool     `json:"autoApprove"`
	ApplicationCount     *int     `json:"applicationCount,omitempty"`
}

type Limits struct {
	MaximumRequests          interface{} `json:"maximumRequests"`
	TimePeriodInMilliseconds int         `json:"timePeriodInMilliseconds"`
	Visible                  bool        `json:"visible"`
}
