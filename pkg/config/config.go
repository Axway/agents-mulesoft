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

var config *AgentConfig

const (
	pathAnypointExchangeURL   = "mulesoft.anypointExchangeUrl"
	pathAuthLifetime          = "mulesoft.auth.lifetime"
	pathAuthPassword          = "mulesoft.auth.password"
	pathAuthUsername          = "mulesoft.auth.username"
	pathCachePath             = "mulesoft.cachePath"
	pathDiscoveryIgnoreTags   = "mulesoft.discoveryIgnoreTags"
	pathDiscoveryTags         = "mulesoft.discoveryTags"
	pathEnvironment           = "mulesoft.environment"
	pathOrgName               = "mulesoft.orgName"
	pathPollInterval          = "mulesoft.pollInterval"
	pathProxyURL              = "mulesoft.proxyUrl"
	pathSSLCipherSuites       = "mulesoft.ssl.cipherSuites"
	pathSSLInsecureSkipVerify = "mulesoft.ssl.insecureSkipVerify"
	pathSSLMaxVersion         = "mulesoft.ssl.maxVersion"
	pathSSLMinVersion         = "mulesoft.ssl.minVersion"
	pathSSLNextProtos         = "mulesoft.ssl.nextProtos"
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
	corecfg.IResourceConfigCallback
	AnypointExchangeURL string            `config:"anypointExchangeUrl"`
	CachePath           string            `config:"cachePath"`
	DiscoveryIgnoreTags string            `config:"discoveryIgnoreTags"`
	DiscoveryTags       string            `config:"discoveryTags"`
	Environment         string            `config:"environment"`
	OrgName             string            `config:"orgname"`
	Password            string            `config:"auth.password"`
	PollInterval        time.Duration     `config:"pollInterval"`
	ProxyURL            string            `config:"proxyUrl"`
	SessionLifetime     time.Duration     `config:"auth.lifetime"`
	TLS                 corecfg.TLSConfig `config:"ssl"`
	Username            string            `config:"auth.username"`
}

// ValidateCfg - Validates the gateway config
func (c *MulesoftConfig) ValidateCfg() (err error) {
	if c.AnypointExchangeURL == "" {
		return errors.New("Invalid mulesoft configuration: anypointExchangeUrl is not configured")
	}

	if c.Username == "" {
		return errors.New("Invalid mulesoft configuration: username is not configured")
	}

	if c.Password == "" {
		return errors.New("Invalid mulesoft configuration: password is not configured")
	}

	if c.Environment == "" {
		return errors.New("Invalid mulesoft configuration: environment is not configured")
	}

	if c.OrgName == "" {
		return errors.New("Invalid mulesoft configuration: OrgName is not configured")
	}

	if c.PollInterval == 0 {
		return errors.New("Invalid mulesoft configuration: pollInterval is invalid")
	}

	if _, err := os.Stat(c.CachePath); os.IsNotExist(err) {
		return fmt.Errorf("invalid mulesoft cache path: path does not exist: %s", c.CachePath)
	}
	c.CachePath = filepath.Clean(c.CachePath)
	return
}

// AddConfigProperties - Adds the command properties needed for Mulesoft
func AddConfigProperties(props properties.Properties) {
	props.AddDurationProperty(pathAuthLifetime, 60*time.Minute, "Mulesoft session lifetime.")
	props.AddDurationProperty(pathPollInterval, 20*time.Second, "The interval at which Mulesoft is checked for updates.")
	props.AddStringProperty(pathAnypointExchangeURL, "https://anypoint.mulesoft.com", "Mulesoft Anypoint Exchange URL.")
	props.AddStringProperty(pathAuthPassword, "", "Mulesoft password.")
	props.AddStringProperty(pathAuthUsername, "", "Mulesoft username.")
	props.AddStringProperty(pathCachePath, "/tmp", "Mulesoft Cache Path")
	props.AddStringProperty(pathDiscoveryIgnoreTags, "", "APIs containing any of these tags are ignored. Takes precedence over "+pathDiscoveryIgnoreTags+".")
	props.AddStringProperty(pathDiscoveryTags, "", "APIs containing any of these tags are selected for discovery.")
	props.AddStringProperty(pathEnvironment, "", "Mulesoft Anypoint environment.")
	props.AddStringProperty(pathOrgName, "", "Mulesoft Anypoint Business Group.")

	// ssl properties and command flags
	props.AddStringSliceProperty(pathSSLNextProtos, []string{}, "List of supported application level protocols, comma separated.")
	props.AddBoolProperty(pathSSLInsecureSkipVerify, false, "Controls whether a client verifies the server's certificate chain and host name.")
	props.AddStringSliceProperty(pathSSLCipherSuites, corecfg.TLSDefaultCipherSuitesStringSlice(), "List of supported cipher suites, comma separated.")
	props.AddStringProperty(pathSSLMinVersion, corecfg.TLSDefaultMinVersionString(), "Minimum acceptable SSL/TLS protocol version.")
	props.AddStringProperty(pathSSLMaxVersion, "0", "Maximum acceptable SSL/TLS protocol version.")
}

// NewMulesoftConfig - parse the props and create an Mulesoft Configuration structure
func NewMulesoftConfig(props properties.Properties) *MulesoftConfig {
	return &MulesoftConfig{
		AnypointExchangeURL: props.StringPropertyValue(pathAnypointExchangeURL),
		CachePath:           props.StringPropertyValue(pathCachePath),
		DiscoveryIgnoreTags: props.StringPropertyValue(pathDiscoveryIgnoreTags),
		DiscoveryTags:       props.StringPropertyValue(pathDiscoveryTags),
		Environment:         props.StringPropertyValue(pathEnvironment),
		OrgName:             props.StringPropertyValue(pathOrgName),
		Password:            props.StringPropertyValue(pathAuthPassword),
		PollInterval:        props.DurationPropertyValue(pathPollInterval),
		ProxyURL:            props.StringPropertyValue(pathProxyURL),
		SessionLifetime:     props.DurationPropertyValue(pathAuthLifetime),
		Username:            props.StringPropertyValue(pathAuthUsername),
		TLS: &corecfg.TLSConfiguration{
			NextProtos:         props.StringSlicePropertyValue(pathSSLNextProtos),
			InsecureSkipVerify: props.BoolPropertyValue(pathSSLInsecureSkipVerify),
			CipherSuites:       corecfg.NewCipherArray(props.StringSlicePropertyValue(pathSSLCipherSuites)),
			MinVersion:         corecfg.TLSVersionAsValue(props.StringPropertyValue(pathSSLMinVersion)),
			MaxVersion:         corecfg.TLSVersionAsValue(props.StringPropertyValue(pathSSLMaxVersion)),
		},
	}
}
