package traceability

// Headers - Type for request/response headers
type Headers map[string]string

// GwTransaction - Type for gateway transaction detail
type GwTransaction struct {
	DesHost         string  `json:"destHost"`
	DestPort        int     `json:"destPort"`
	ID              string  `json:"id"`
	Method          string  `json:"method"`
	RequestBytes    int     `json:"requestByte"`
	RequestHeaders  Headers `json:"requestHeaders"`
	ResponseBytes   int     `json:"responseByte"`
	ResponseHeaders Headers `json:"responseHeaders"`
	SourceHost      string  `json:"srcHost"`
	SourcePort      int     `json:"srcPort"`
	StatusCode      int     `json:"statusCode"`
	URI             string  `json:"uri"`
}

// GwTrafficLogEntry - Represents the structure of log entry the agent will receive
type GwTrafficLogEntry struct {
	APIName             string        `json:"apiName"`
	InboundTransaction  GwTransaction `json:"inbound"`
	OutboundTransaction GwTransaction `json:"outbound"`
	TraceID             string        `json:"traceId"`
}
