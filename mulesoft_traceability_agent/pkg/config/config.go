package config

import (
	"errors"
	"os"
	"time"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/cmd/properties"
	corecfg "github.com/Axway/agent-sdk/pkg/config"
)

var config *AgentConfig

// SetConfig -
func SetConfig(newConfig *AgentConfig) {
	config = newConfig
}

// GetConfig -
func GetConfig() *AgentConfig {
	return config
}

// AgentConfig - represents the config for agent
type AgentConfig struct {
	CentralConfig  corecfg.CentralConfig `config:"central"`
	MulesoftConfig *MulesoftConfig       `config:"mulesoft"`
}

// MulesoftConfig - represents the config for gateway
type MulesoftConfig struct {
	corecfg.IConfigValidator
	corecfg.IResourceConfigCallback
	AnypointExchangeURL string            `config:"anypointExchangeUrl"`
	PollInterval        time.Duration     `config:"pollInterval"`
	OrganizationID      string            `config:"organizationID"`
	Environment         string            `config:"environment"`
	Username            string            `config:"auth.username"`
	Password            string            `config:"auth.password"`
	SessionLifetime     time.Duration     `config:"auth.lifetime"`
	TLS                 corecfg.TLSConfig `config:"ssl"`
	ProxyURL            string            `config:"proxyUrl"`
	CachePath           string            `config:"cachePath"`
}

// NewMulesoftConfig creates an empty config.
func NewMulesoftConfig() MulesoftConfig {
	return MulesoftConfig{
		PollInterval: time.Minute,
		TLS:          &corecfg.TLSConfiguration{},
	}
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

	if c.PollInterval == 0 {
		return errors.New("Invalid mulesoft configuration: pollInterval is invalid")
	}
	if _, err := os.Stat(c.CachePath); os.IsNotExist(err) {
		// path/to/whatever does not exist
		return errors.New("Invalid mulesoft cache path: path does not exist")
	}
	return
}

// ApplyResources - Applies the agent and dataplane resource to config
func (c *MulesoftConfig) ApplyResources(dataplaneResource *v1.ResourceInstance, agentResource *v1.ResourceInstance) error {
	// Currently there is not dataplane agent configuration for mulesoft.
	return nil
}

const (
	pathAnypointExchangeURL   = "mulesoft.anypointExchangeUrl"
	pathPollInterval          = "mulesoft.pollInterval"
	pathEnvironment           = "mulesoft.environment"
	pathAuthUsername          = "mulesoft.auth.username"
	pathAuthPassword          = "mulesoft.auth.password"
	pathAuthLifetime          = "mulesoft.auth.lifetime"
	pathSSLNextProtos         = "mulesoft.ssl.nextProtos"
	pathSSLInsecureSkipVerify = "mulesoft.ssl.insecureSkipVerify"
	pathSSLCipherSuites       = "mulesoft.ssl.cipherSuites"
	pathSSLMinVersion         = "mulesoft.ssl.minVersion"
	pathSSLMaxVersion         = "mulesoft.ssl.maxVersion"
	pathProxyURL              = "mulesoft.proxyUrl"
	pathCachePath             = "mulesoft.cachePath"
)

// AddMulesoftConfigProperties - Adds the command properties needed for Mulesoft
func AddMulesoftConfigProperties(props properties.Properties) {
	props.AddStringProperty(pathAnypointExchangeURL, "https://anypoint.mulesoft.com", "Mulesoft Anypoint Exchange URL.")
	props.AddDurationProperty(pathPollInterval, 60*time.Second, "The interval at which Mulesoft is checked for updates.")
	props.AddStringProperty(pathEnvironment, "", "Mulesoft Anypoint environment.")
	props.AddStringProperty(pathAuthUsername, "", "Mulesoft username")
	props.AddStringProperty(pathAuthPassword, "", "Mulesoft password")
	props.AddDurationProperty(pathAuthLifetime, 60*time.Minute, "Mulesoft session lifetime")
	props.AddStringProperty(pathCachePath, "/tmp", "Mulesoft Cache Path")

	// ssl properties and command flags
	props.AddStringSliceProperty(pathSSLNextProtos, []string{}, "List of supported application level protocols, comma separated")
	props.AddBoolProperty(pathSSLInsecureSkipVerify, false, "Controls whether a client verifies the server's certificate chain and host name")
	props.AddStringSliceProperty(pathSSLCipherSuites, corecfg.TLSDefaultCipherSuitesStringSlice(), "List of supported cipher suites, comma separated")
	props.AddStringProperty(pathSSLMinVersion, corecfg.TLSDefaultMinVersionString(), "Minimum acceptable SSL/TLS protocol version")
	props.AddStringProperty(pathSSLMaxVersion, "0", "Maximum acceptable SSL/TLS protocol version")
}

// ParseMulesoftConfig - parse the props and create an Mulesoft Configuration structure
func ParseMulesoftConfig(props properties.Properties) *MulesoftConfig {
	return &MulesoftConfig{
		AnypointExchangeURL: props.StringPropertyValue(pathAnypointExchangeURL),
		PollInterval:        props.DurationPropertyValue(pathPollInterval),
		CachePath:           props.StringPropertyValue(pathCachePath),
		Environment:         props.StringPropertyValue(pathEnvironment),
		Username:            props.StringPropertyValue(pathAuthUsername),
		Password:            props.StringPropertyValue(pathAuthPassword),
		SessionLifetime:     props.DurationPropertyValue(pathAuthLifetime),
		TLS: &corecfg.TLSConfiguration{
			NextProtos:         props.StringSlicePropertyValue(pathSSLNextProtos),
			InsecureSkipVerify: props.BoolPropertyValue(pathSSLInsecureSkipVerify),
			CipherSuites:       corecfg.NewCipherArray(props.StringSlicePropertyValue(pathSSLCipherSuites)),
			MinVersion:         corecfg.TLSVersionAsValue(props.StringPropertyValue(pathSSLMinVersion)),
			MaxVersion:         corecfg.TLSVersionAsValue(props.StringPropertyValue(pathSSLMaxVersion)),
		},
		ProxyURL: props.StringPropertyValue(pathProxyURL),
	}
}
