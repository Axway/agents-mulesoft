package anypoint

import "time"

var assetTypeMap = map[string]string{
	"template":     "mule-application-template",
	"example":      "mule-plugin",
	"connector":    "mule-plugin",
	"extension":    "mule-plugin",
	"custom":       "custom",
	"api-fragment": "raml-fragment",
	"soap-api":     "wsdl",
	"rest-api":     "oas",
}

// Asset - https://anypoint.mulesoft.com/exchange/portals/anypoint-platform/f1e97bc6-315a-4490-82a7-23abe036327a.anypoint-platform/exchange-experience-api/minor/2.0/console/type/%231186/
type Asset struct {
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	GroupID      string     `json:"groupId"`
	AssetID      string     `json:"assetId"`
	Version      string     `json:"version"`
	MinorVersion string     `json:"minorVersion"`
	VersionGroup string     `json:"versionGroup"`
	Description  string     `json:"description"`
	Public       bool       `json:"isPublic"`
	AssetType    string     `json:"type"`
	Snapshopt    bool       `json:"isSnapshot"`
	Status       string     `json:"status"`
	CreatedAt    time.Time  `json:"createdAt"`
	ModifiedAt   time.Time  `json:"modifiedAt"`
	Labels       []string   `json:"labels"`
	Categories   []Category `json:"categories"`
	Icon         string     `json:"icon"`
	Files        []File     `json:"files"`
}

// AssetDetails -
type AssetDetails struct {
	Asset
	Instances []APIInstance `json:"instances"`
}

// Category -
type Category struct {
	Value       []string `json:"value"`
	DisplayName string   `json:"displayName"`
	Key         string   `json:"key"`
}

// User -
type User struct {
	IdentityType string       `json:"identityType"`
	ID           string       `json:"id"`
	Username     string       `json:"username"`
	FirstName    string       `json:"firstName"`
	LastName     string       `json:"lastName"`
	Email        string       `json:"email"`
	Organization Organization `json:"organization"`
}

// Organization -
type Organization struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Domain string `json:"domain"`
}

// APIInstance - https://anypoint.mulesoft.com/exchange/portals/anypoint-platform/f1e97bc6-315a-4490-82a7-23abe036327a.anypoint-platform/exchange-experience-api/minor/2.0/console/method/%231972/
type APIInstance struct {
	ID                string    `json:"id"`
	Name              string    `json:"name"`
	GroupID           string    `json:"groupId"`
	AssetID           string    `json:"assetId"`
	ProductAPIVersion string    `json:"productApiVersion"`
	EnvironmentID     string    `json:"environmentId"`
	EndpointURI       string    `json:"endpointUri"`
	Public            bool      `json:"isPublic"`
	InstanceType      string    `json:"type"`
	CreatedBy         string    `json:"createdBy"`
	CreatedDate       time.Time `json:"createdDate"`
	UpdatedDate       time.Time `json:"updatedDate"`
	ProviderID        string    `json:"providerId"`
}

// File -
type File struct {
	Classifier   string    `json:"classifier"`
	Packaging    string    `json:"packaging"`
	DownloadURL  string    `json:"downloadURL"`
	ExternalLink string    `json:"externalLink"`
	MD5          string    `json:"md5"`
	SHA1         string    `json:"sha1"`
	CreatedDate  time.Time `json:"createdDate"`
	MainFile     string    `json:"mainFile"`
	Generated    bool      `json:"isGenerated"`
}

// Environment -
type Environment struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	OrganizationID string `json:"organizationId"`
	IsProduction   bool   `json:"isProduction"`
	Type           string `json:"type"`
	ClientID       string `json:"clientId"`
}
// AnalyticsEvent -
type AnalyticsEvent struct {
	ClientIP           string      `json:"Client IP"`
	APIID              string      `json:"API ID"`
	APIName            string      `json:"API Name"`
	APIVersionID       string      `json:"API Version ID"`
	APIVersionName     string      `json:"API Version Name"`
	ApplicationName    string      `json:"Application Name"`
	MessageID          string      `json:"Message ID"`
	Timestamp          time.Time   `json:"Timestamp"`
	RequestSize        interface{} `json:"Request Size"`
	ResponseSize       interface{} `json:"Response Size"`
	RequestOutcome     string      `json:"Request Outcome"`
	Verb               string      `json:"Verb"`
	ResourcePath       string      `json:"Resource Path"`
	StatusCode         int         `json:"Status Code"`
	UserAgentName      interface{} `json:"User Agent Name"`
	UserAgentVersion   interface{} `json:"User Agent Version"`
	Browser            interface{} `json:"Browser"`
	OSFamily           interface{} `json:"OS Family"`
	OSVersion          interface{} `json:"OS Version"`
	OSMajorVersion     interface{} `json:"OS Major Version"`
	OSMinorVersion     interface{} `json:"OS Minor Version"`
	HardwarePlatform   interface{} `json:"Hardware Platform"`
	Timezone           string      `json:"Timezone"`
	City               string      `json:"City"`
	Country            string      `json:"Country"`
	Continent          string      `json:"Continent"`
	PostalCode         string      `json:"Postal Code"`
	ViolatedPolicyName interface{} `json:"Violated Policy Name"`
	ResponseTime       int         `json:"Response Time"`
}