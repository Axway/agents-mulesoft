package config

import (
	"errors"
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
	DiscoveryIgnoreTags string            `config:"discoveryIgnoreTags"`
	Filter              string            `config:"filter"`
	Username            string            `config:"auth.username"`
	Password            string            `config:"auth.password"`
	SessionLifetime     time.Duration     `config:"auth.lifetime"`
	TLS                 corecfg.TLSConfig `config:"ssl"`
	ProxyURL            string            `config:"proxyUrl"`
}

// NewMulesoftConfig creates an empty config.
func NewMulesoftConfig() MulesoftConfig {
	return MulesoftConfig{
		TLS: &corecfg.TLSConfiguration{},
	}
}

// ValidateCfg - Validates the gateway config
func (c *MulesoftConfig) ValidateCfg() (err error) {
	if c.AnypointExchangeURL == "" {
		return errors.New("Invalid mulesoft configuration: anypointExchangeUrl is not configured")
	}

	// TODO VALIDATION

	return
}

// ApplyResources - Applies the agent and dataplane resource to config
func (c *MulesoftConfig) ApplyResources(dataplaneResource *v1.ResourceInstance, agentResource *v1.ResourceInstance) error {
	// TODO: Extract config from SaaS model
	return nil
}

const (
	pathAnypointExchangeURL   = "mulesoft.anypointExchangeUrl"
	pathAuthUsername          = "mulesoft.auth.username"
	pathAuthPassword          = "mulesoft.auth.password"
	pathAuthLifetime          = "mulesoft.auth.lifetime"
	pathSSLNextProtos         = "mulesoft.ssl.nextProtos"
	pathSSLInsecureSkipVerify = "mulesoft.ssl.insecureSkipVerify"
	pathSSLCipherSuites       = "mulesoft.ssl.cipherSuites"
	pathSSLMinVersion         = "mulesoft.ssl.minVersion"
	pathSSLMaxVersion         = "mulesoft.ssl.maxVersion"
	pathProxyURL              = "mulesoft.proxyUrl"
)

// AddMulesoftConfigProperties - Adds the command properties needed for Mulesoft
func AddMulesoftConfigProperties(props properties.Properties) {
	props.AddStringProperty(pathAnypointExchangeURL, "https://anypoint.mulesoft.com", "Mulesoft Anypoint Exchange URL.")
	props.AddStringProperty(pathAuthUsername, "", "Mulesoft username")
	props.AddStringProperty(pathAuthPassword, "", "Mulesoft password")
	props.AddDurationProperty(pathAuthLifetime, 60*time.Minute, "Mulesoft session lifetime")

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
