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

// AnalyticsEvent -
type AnalyticsEvent struct {
	ClientIP           string    `json:"Client IP"`
	APIID              string    `json:"API ID"`
	APIName            string    `json:"API Name"`
	APIVersionID       string    `json:"API Version ID"`
	APIVersionName     string    `json:"API Version Name"`
	ApplicationName    string    `json:"Application Name"`
	MessageID          string    `json:"Message ID"`
	Timestamp          time.Time `json:"Timestamp"`
	RequestSize        int       `json:"Request Size"`
	ResponseSize       int       `json:"Response Size"`
	RequestOutcome     string    `json:"Request Outcome"`
	Verb               string    `json:"Verb"`
	ResourcePath       string    `json:"Resource Path"`
	StatusCode         int       `json:"Status Code"`
	UserAgentName      string    `json:"User Agent Name"`
	UserAgentVersion   string    `json:"User Agent Version"`
	Browser            string    `json:"Browser"`
	OSFamily           string    `json:"OS Family"`
	OSVersion          string    `json:"OS Version"`
	OSMajorVersion     string    `json:"OS Major Version"`
	OSMinorVersion     string    `json:"OS Minor Version"`
	HardwarePlatform   string    `json:"Hardware Platform"`
	Timezone           string    `json:"Timezone"`
	City               string    `json:"City"`
	Country            string    `json:"Country"`
	Continent          string    `json:"Continent"`
	PostalCode         string    `json:"Postal Code"`
	ViolatedPolicyName string    `json:"Violated Policy Name"`
	ResponseTime       int       `json:"Response Time"`
}

// Headers - Type for request/response headers
type Headers map[string]string

// GwTransaction - Type for gateway transaction detail
type GwTransaction struct {
	ID              string  `json:"id"`
	SourceHost      string  `json:"srcHost"`
	SourcePort      int     `json:"srcPort"`
	DesHost         string  `json:"destHost"`
	DestPort        int     `json:"destPort"`
	URI             string  `json:"uri"`
	Method          string  `json:"method"`
	StatusCode      int     `json:"statusCode"`
	RequestHeaders  Headers `json:"requestHeaders"`
	ResponseHeaders Headers `json:"responseHeaders"`
	RequestBytes    int     `json:"requestByte"`
	ResponseBytes   int     `json:"responseByte"`
}

// GwTrafficLogEntry - Represents the structure of log entry the agent will receive
type GwTrafficLogEntry struct {
	TraceID             string        `json:"traceId"`
	APIName             string        `json:"apiName"`
	InboundTransaction  GwTransaction `json:"inbound"`
	OutboundTransaction GwTransaction `json:"outbound"`
}
