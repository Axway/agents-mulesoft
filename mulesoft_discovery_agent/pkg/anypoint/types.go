package anypoint

import "time"

// CurrentUser -
type CurrentUser struct {
	User User `json:"user"`
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

// Environment -
type Environment struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	OrganizationID  string `json:"organizationId"`
	IsProduction    bool   `json:"isProduction"`
	EnvironmentType string `json:"type"`
	ClientID        string `json:"clientId"`
}

// EnvironmentSearch -
type EnvironmentSearch struct {
	Data  []Environment `json:"data"`
	Total int           `json:"total"`
}

// AssetSearch -
type AssetSearch struct {
	Total  int     `json:"total"`
	Assets []Asset `json:"assets"`
}

// Asset -
type Asset struct {
	Audit                Audit  `json:"audit"`
	MasterOrganizationID string `json:"masterOrganizationId"`
	OrganizationID       string `json:"organizationId"`
	ID                   int64  `json:"id"`
	Name                 string `json:"name"`
	ExchangeAssetName    string `json:"exchangeAssetName"`
	GroupID              string `json:"groupId"`
	AssetID              string `json:"assetId"`
	APIs                 []API  `json:"apis"`
	TotalAPIs            int    `json:"totalApis"`
	AutodiscoveryAPIName string `json:"autodiscoveryApiName"`
}

// API -
type API struct {
	Audit                     Audit    `json:"audit"`
	MasterOrganizationID      string   `json:"masterOrganizationId"`
	OrganizationID            string   `json:"organizationId"`
	ID                        int64    `json:"id"`
	InstanceLabel             string   `json:"instanceLabel"`
	GroupID                   string   `json:"groupId"`
	AssetID                   string   `json:"assetId"`
	AssetVersion              string   `json:"assetVersion"`
	ProductVersion            string   `json:"productVersion"`
	Description               string   `json:"description"`
	Tags                      []string `json:"tags"`
	Order                     int      `json:"order"`
	Deprecated                bool     `json:"deprecated"`
	EndpointURI               string   `json:"endpointUri"`
	EnvironmentID             string   `json:"environmentId"`
	IsPublic                  bool     `json:"isPublic"`
	Pinned                    bool     `json:"pinned"`
	ActiveContractsCount      int      `json:"activeContractsCount"`
	AutodiscoveryInstanceName string   `json:"autodiscoveryInstanceName"`
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

// var assetTypeMap = map[string]string{
// 	"template":     "mule-application-template",
// 	"example":      "mule-plugin",
// 	"connector":    "mule-plugin",
// 	"extension":    "mule-plugin",
// 	"custom":       "custom",
// 	"api-fragment": "raml-fragment",
// 	"soap-api":     "wsdl",
// 	"rest-api":     "oas",
// }

// // Asset - https://anypoint.mulesoft.com/exchange/portals/anypoint-platform/f1e97bc6-315a-4490-82a7-23abe036327a.anypoint-platform/exchange-experience-api/minor/2.0/console/type/%231186/
// type Asset struct {
// 	ID           string     `json:"id"`
// 	Name         string     `json:"name"`
// 	GroupID      string     `json:"groupId"`
// 	AssetID      string     `json:"assetId"`
// 	Version      string     `json:"version"`
// 	MinorVersion string     `json:"minorVersion"`
// 	VersionGroup string     `json:"versionGroup"`
// 	Description  string     `json:"description"`
// 	Public       bool       `json:"isPublic"`
// 	AssetType    string     `json:"type"`
// 	Snapshopt    bool       `json:"isSnapshot"`
// 	Status       string     `json:"status"`
// 	CreatedAt    time.Time  `json:"createdAt"`
// 	ModifiedAt   time.Time  `json:"modifiedAt"`
// 	Labels       []string   `json:"labels"`
// 	Categories   []Category `json:"categories"`
// 	Icon         string     `json:"icon"`
// 	Files        []File     `json:"files"`
// }

// // AssetDetails -
// type AssetDetails struct {
// 	Asset
// 	Instances []APIInstance `json:"instances"`
// }

// // Category -
// type Category struct {
// 	Value       []string `json:"value"`
// 	DisplayName string   `json:"displayName"`
// 	Key         string   `json:"key"`
// }

// // APIInstance - https://anypoint.mulesoft.com/exchange/portals/anypoint-platform/f1e97bc6-315a-4490-82a7-23abe036327a.anypoint-platform/exchange-experience-api/minor/2.0/console/method/%231972/
// type APIInstance struct {
// 	ID                string    `json:"id"`
// 	Name              string    `json:"name"`
// 	GroupID           string    `json:"groupId"`
// 	AssetID           string    `json:"assetId"`
// 	ProductAPIVersion string    `json:"productApiVersion"`
// 	EnvironmentID     string    `json:"environmentId"`
// 	EndpointURI       string    `json:"endpointUri"`
// 	Public            bool      `json:"isPublic"`
// 	InstanceType      string    `json:"type"`
// 	CreatedBy         string    `json:"createdBy"`
// 	CreatedDate       time.Time `json:"createdDate"`
// 	UpdatedDate       time.Time `json:"updatedDate"`
// 	ProviderID        string    `json:"providerId"`
// }

// // File -
// type File struct {
// 	Classifier   string    `json:"classifier"`
// 	Packaging    string    `json:"packaging"`
// 	DownloadURL  string    `json:"downloadURL"`
// 	ExternalLink string    `json:"externalLink"`
// 	MD5          string    `json:"md5"`
// 	SHA1         string    `json:"sha1"`
// 	CreatedDate  time.Time `json:"createdDate"`
// 	MainFile     string    `json:"mainFile"`
// 	Generated    bool      `json:"isGenerated"`
// }
