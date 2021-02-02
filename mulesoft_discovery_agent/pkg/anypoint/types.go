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
