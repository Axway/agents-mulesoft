package gateway

// CHANGE_HERE - Change the structures below to represent the log entry the agent is going to receive

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

func GenerateSample() *GwTrafficLogEntry {
	return &GwTrafficLogEntry{
		TraceID:             "123459",
		APIName:             "SampleAPI2",
		InboundTransaction:  GwTransaction{
			ID:              "12233444",
			SourceHost:      "fidoshouse.com",
			SourcePort:      443,
			DesHost:         "petstore.com",
			DestPort:        443,
			URI:             "https://www.petstore.com/food",
			Method:          "GET",
			StatusCode:      200,
			RequestHeaders:  nil,
			ResponseHeaders: nil,
			RequestBytes:    100,
			ResponseBytes:   1000,
		},
		OutboundTransaction: GwTransaction{
			ID:              "12233555",
			SourceHost:      "petstore.com",
			SourcePort:      443,
			DesHost:         "fidoshouse.com",
			DestPort:        0,
			URI:             "https://www.petstore.com/food",
			Method:          "GET",
			StatusCode:      200,
			RequestHeaders:  nil,
			ResponseHeaders: nil,
			RequestBytes:    100,
			ResponseBytes:   1000,
		},
	}
}