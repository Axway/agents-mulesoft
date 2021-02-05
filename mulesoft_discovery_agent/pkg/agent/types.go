package agent

import anypoint "github.com/Axway/agents-mulesoft/mulesoft_discovery_agent/pkg/anypoint"

// ServiceDetail is the information for the ex
type ServiceDetail struct {
	ID                string
	Title             string
	APIName           string
	URL               string
	Stage             string
	Description       string
	AuthPolicy        string
	APISpec           []byte
	Documentation     []byte
	Tags              []string
	Image             string
	ImageContentType  string
	ResourceType      string
	SubscriptionName  string
	APIUpdateSeverity string
	State             string
	Status            string
	ServiceAttributes map[string]string
	Instances         []anypoint.ExchangeAPIInstance
}

var specPreference = map[string]int{
	"oas":      0,
	"fat-oas":  1,
	"wsdl":     2,
	"raml":     3,
	"fat-raml": 4,
}

// BySpecType implements sort.Interface for []ExchangeFile based on
// prefered specification support.
type BySpecType []anypoint.ExchangeFile

func (a BySpecType) Len() int      { return len(a) }
func (a BySpecType) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a BySpecType) Less(i, j int) bool {
	var ok bool
	iVal := 0
	jVal := 0
	if iVal, ok = specPreference[a[i].Classifier]; !ok {
		iVal = 10000
	}
	if jVal, ok = specPreference[a[j].Classifier]; !ok {
		jVal = 10000
	}
	return iVal < jVal
}
