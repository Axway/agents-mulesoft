package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Axway/agent-sdk/pkg/cmd/properties"
	corecfg "github.com/Axway/agent-sdk/pkg/config"
)

type props interface {
	AddStringProperty(name string, defaultVal string, description string)
	AddStringSliceProperty(name string, defaultVal []string, description string)
	AddBoolProperty(name string, defaultVal bool, description string)
	AddDurationProperty(name string, defaultVal time.Duration, description string, opts ...properties.DurationOpt)
	StringPropertyValue(name string) string
	StringSlicePropertyValue(name string) []string
	BoolPropertyValue(name string) bool
	DurationPropertyValue(name string) time.Duration
}

var config *AgentConfig

const (
	pathAnypointExchangeURL   = "mulesoft.anypointExchangeUrl"
	pathAnypointMonitoringURL = "mulesoft.anypointMonitoringUrl"
	pathEnvironment           = "mulesoft.environment"
	pathOrgName               = "mulesoft.orgName"
	pathDiscoveryTags         = "mulesoft.discoveryTags"
	pathDiscoveryIgnoreTags   = "mulesoft.discoveryIgnoreTags"
	pathAuthClientID          = "mulesoft.auth.clientID"
	pathAuthClientSecret      = "mulesoft.auth.clientSecret"
	pathAuthLifetime          = "mulesoft.auth.lifetime"
	pathSSLNextProtos         = "mulesoft.ssl.nextProtos"
	pathSSLInsecureSkipVerify = "mulesoft.ssl.insecureSkipVerify"
	pathSSLCipherSuites       = "mulesoft.ssl.cipherSuites"
	pathSSLMinVersion         = "mulesoft.ssl.minVersion"
	pathSSLMaxVersion         = "mulesoft.ssl.maxVersion"
	pathPollInterval          = "mulesoft.pollInterval"
	pathProxyURL              = "mulesoft.proxyUrl"
	pathCachePath             = "mulesoft.cachePath"
	pathDiscoverOriginalRaml  = "mulesoft.discoverOriginalRaml"
	pathUseMonitoringAPI      = "mulesoft.useMonitoringAPI"
)

const (
	anypointExchangeUrlErr = "invalid mulesoft configuration: anypointExchangeUrl is not configured"
	clientCredentialsErr   = "invalid mulesoft configuration: clientID and clientSecret are required. Using Username and password is deprecated"
	envErr                 = "invalid mulesoft configuration: environment is not configured"
	orgNameErr             = "invalid mulesoft configuration: OrgName is not configured"
	pollIntervalErr        = "invalid mulesoft configuration: pollInterval is invalid"
	cachePathErr           = "invalid mulesoft cache path: path does not exist: "
)

// SetConfig sets the global AgentConfig reference.
func SetConfig(newConfig *AgentConfig) {
	config = newConfig
}

// GetConfig gets the AgentConfig
func GetConfig() *AgentConfig {
	return config
}

// AgentConfig - represents the config for agent
type AgentConfig struct {
	CentralConfig  corecfg.CentralConfig `config:"central"`
	MulesoftConfig *MulesoftConfig       `config:"mulesoft"`
}

// MulesoftConfig - represents the config for the Mulesoft gateway
type MulesoftConfig struct {
	corecfg.IConfigValidator
	AnypointExchangeURL   string            `config:"anypointExchangeUrl"`
	AnypointMonitoringURL string            `config:"anypointMonitoringUrl"`
	CachePath             string            `config:"cachePath"`
	DiscoveryIgnoreTags   string            `config:"discoveryIgnoreTags"`
	DiscoveryTags         string            `config:"discoveryTags"`
	Environment           string            `config:"environment"`
	OrgName               string            `config:"orgname"`
	PollInterval          time.Duration     `config:"pollInterval"`
	ProxyURL              string            `config:"proxyUrl"`
	SessionLifetime       time.Duration     `config:"auth.lifetime"`
	TLS                   corecfg.TLSConfig `config:"ssl"`
	ClientID              string            `config:"auth.clientID"`
	ClientSecret          string            `config:"auth.clientSecret"`
	DiscoverOriginalRaml  bool              `config:"discoverOriginalRaml"`
	UseMonitoringAPI      bool              `config:"useMonitoringAPI"`
}

// ValidateCfg - Validates the gateway config
func (c *MulesoftConfig) ValidateCfg() (err error) {
	if c.AnypointExchangeURL == "" {
		return errors.New(anypointExchangeUrlErr)
	}

	if c.ClientID == "" || c.ClientSecret == "" {
		return errors.New(clientCredentialsErr)
	}

	if c.Environment == "" {
		return errors.New(envErr)
	}

	if c.OrgName == "" {
		return errors.New(orgNameErr)
	}

	if c.PollInterval == 0 {
		return errors.New(pollIntervalErr)
	}

	if _, err := os.Stat(c.CachePath); os.IsNotExist(err) {
		return fmt.Errorf(cachePathErr + c.CachePath)
	}
	c.CachePath = filepath.Clean(c.CachePath)
	return
}

// AddConfigProperties - Adds the command properties needed for Mulesoft
func AddConfigProperties(rootProps props, isTA bool) {
	rootProps.AddStringProperty(pathAnypointExchangeURL, "https://anypoint.mulesoft.com", "Mulesoft Anypoint Exchange URL.")
	rootProps.AddStringProperty(pathAnypointMonitoringURL, "https://monitoring.anypoint.mulesoft.com", "Mulesoft Anypoint Monitoring URL.")
	rootProps.AddStringProperty(pathEnvironment, "", "Mulesoft Anypoint environment.")
	rootProps.AddStringProperty(pathOrgName, "", "Mulesoft Anypoint Business Group.")
	rootProps.AddStringProperty(pathAuthClientID, "", "Mulesoft client id.")
	rootProps.AddStringProperty(pathAuthClientSecret, "", "Mulesoft client secret.")
	rootProps.AddDurationProperty(pathAuthLifetime, 60*time.Minute, "Mulesoft session lifetime.")
	rootProps.AddStringProperty(pathDiscoveryTags, "", "APIs containing any of these tags are selected for discovery.")
	rootProps.AddStringProperty(pathDiscoveryIgnoreTags, "", "APIs containing any of these tags are ignored. Takes precedence over "+pathDiscoveryIgnoreTags+".")
	rootProps.AddStringProperty(pathCachePath, "/data", "Mulesoft Cache Path")

	if isTA {
		rootProps.AddDurationProperty(pathPollInterval, 5*time.Minute, "The interval at which Mulesoft Traceability is checked for updates.", properties.WithLowerLimit(30*time.Second))
	} else {
		rootProps.AddDurationProperty(pathPollInterval, time.Minute, "The interval at which Mulesoft Discovery is checked for updates.", properties.WithLowerLimit(30*time.Second))
	}

	rootProps.AddStringProperty(pathProxyURL, "", "Proxy URL")

	// ssl properties and command flags
	rootProps.AddStringSliceProperty(pathSSLNextProtos, []string{}, "List of supported application level protocols, comma separated.")
	rootProps.AddBoolProperty(pathSSLInsecureSkipVerify, false, "Controls whether a client verifies the server's certificate chain and host name.")
	rootProps.AddStringSliceProperty(pathSSLCipherSuites, corecfg.TLSDefaultCipherSuitesStringSlice(), "List of supported cipher suites, comma separated.")
	rootProps.AddStringProperty(pathSSLMinVersion, corecfg.TLSDefaultMinVersionString(), "Minimum acceptable SSL/TLS protocol version.")
	rootProps.AddStringProperty(pathSSLMaxVersion, "0", "Maximum acceptable SSL/TLS protocol version.")
	rootProps.AddBoolProperty(pathDiscoverOriginalRaml, false, "If RAML API specs are discovered as RAML and not converted to OAS")
	rootProps.AddBoolProperty(pathUseMonitoringAPI, true, "Flag to setup traceability agent to use Anypoint Monitoring Archive API")
}

// NewMulesoftConfig - parse the props and create an Mulesoft Configuration structure
func NewMulesoftConfig(rootProps props) *MulesoftConfig {
	return &MulesoftConfig{
		AnypointExchangeURL:   rootProps.StringPropertyValue(pathAnypointExchangeURL),
		AnypointMonitoringURL: rootProps.StringPropertyValue(pathAnypointMonitoringURL),
		CachePath:             rootProps.StringPropertyValue(pathCachePath),
		DiscoveryIgnoreTags:   rootProps.StringPropertyValue(pathDiscoveryIgnoreTags),
		DiscoveryTags:         rootProps.StringPropertyValue(pathDiscoveryTags),
		Environment:           rootProps.StringPropertyValue(pathEnvironment),
		OrgName:               rootProps.StringPropertyValue(pathOrgName),
		PollInterval:          rootProps.DurationPropertyValue(pathPollInterval),
		ProxyURL:              rootProps.StringPropertyValue(pathProxyURL),
		SessionLifetime:       rootProps.DurationPropertyValue(pathAuthLifetime),
		ClientID:              rootProps.StringPropertyValue(pathAuthClientID),
		ClientSecret:          rootProps.StringPropertyValue(pathAuthClientSecret),
		TLS: &corecfg.TLSConfiguration{
			NextProtos:         rootProps.StringSlicePropertyValue(pathSSLNextProtos),
			InsecureSkipVerify: rootProps.BoolPropertyValue(pathSSLInsecureSkipVerify),
			CipherSuites:       corecfg.NewCipherArray(rootProps.StringSlicePropertyValue(pathSSLCipherSuites)),
			MinVersion:         corecfg.TLSVersionAsValue(rootProps.StringPropertyValue(pathSSLMinVersion)),
			MaxVersion:         corecfg.TLSVersionAsValue(rootProps.StringPropertyValue(pathSSLMaxVersion)),
		},
		DiscoverOriginalRaml: rootProps.BoolPropertyValue(pathDiscoverOriginalRaml),
		UseMonitoringAPI:     rootProps.BoolPropertyValue(pathUseMonitoringAPI),
	}
}
