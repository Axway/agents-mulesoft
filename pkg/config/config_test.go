package config

import (
	"testing"
	"time"

	"github.com/Axway/agent-sdk/pkg/cmd/properties"
	corecfg "github.com/Axway/agent-sdk/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestKongGatewayCfg(t *testing.T) {
	cfg := &MulesoftConfig{}

	err := cfg.ValidateCfg()
	assert.Equal(t, anypointExchangeUrlErr, err.Error())

	cfg.AnypointExchangeURL = "test.com"
	err = cfg.ValidateCfg()
	assert.Equal(t, clientCredentialsErr, err.Error())

	cfg.ClientID = "Tom"
	cfg.ClientSecret = "Jerry"
	err = cfg.ValidateCfg()
	assert.Equal(t, envErr, err.Error())

	cfg.Environment = "Backyard"
	err = cfg.ValidateCfg()
	assert.Equal(t, orgNameErr, err.Error())

	cfg.OrgName = "Warner Bros"
	err = cfg.ValidateCfg()
	assert.Equal(t, pollIntervalErr, err.Error())

	cfg.PollInterval = 20 * time.Minute
	err = cfg.ValidateCfg()
	assert.Equal(t, cachePathErr, err.Error())

	cfg.CachePath = "./"
	err = cfg.ValidateCfg()
	assert.Equal(t, nil, err)

}

type propData struct {
	pType string
	desc  string
	val   interface{}
}

type fakeProps struct {
	props map[string]propData
}

func (f *fakeProps) AddStringProperty(name string, defaultVal string, description string) {
	f.props[name] = propData{"string", description, defaultVal}
}

func (f *fakeProps) AddStringSliceProperty(name string, defaultVal []string, description string) {
	f.props[name] = propData{"string", description, defaultVal}
}

func (f *fakeProps) AddDurationProperty(name string, defaultVal time.Duration, description string, opts ...properties.DurationOpt) {
	f.props[name] = propData{"duration", description, defaultVal}
}

func (f *fakeProps) AddBoolProperty(name string, defaultVal bool, description string) {
	f.props[name] = propData{"bool", description, defaultVal}
}

func (f *fakeProps) StringPropertyValue(name string) string {
	if prop, ok := f.props[name]; ok {
		return prop.val.(string)
	}
	return ""
}

func (f *fakeProps) StringSlicePropertyValue(name string) []string {
	if prop, ok := f.props[name]; ok {
		return prop.val.([]string)
	}
	return []string{}
}

func (f *fakeProps) DurationPropertyValue(name string) time.Duration {
	if prop, ok := f.props[name]; ok {
		return prop.val.(time.Duration)
	}
	return 0
}

func (f *fakeProps) BoolPropertyValue(name string) bool {
	if prop, ok := f.props[name]; ok {
		return prop.val.(bool)
	}
	return false
}

func TestKongProperties(t *testing.T) {
	newProps := &fakeProps{props: map[string]propData{}}

	// validate add props
	AddConfigProperties(newProps)
	assert.Contains(t, newProps.props, pathAnypointExchangeURL)
	assert.Contains(t, newProps.props, pathEnvironment)
	assert.Contains(t, newProps.props, pathOrgName)
	assert.Contains(t, newProps.props, pathDiscoveryTags)
	assert.Contains(t, newProps.props, pathDiscoveryIgnoreTags)
	assert.Contains(t, newProps.props, pathAuthClientID)
	assert.Contains(t, newProps.props, pathAuthClientSecret)
	assert.Contains(t, newProps.props, pathAuthLifetime)
	assert.Contains(t, newProps.props, pathSSLNextProtos)
	assert.Contains(t, newProps.props, pathSSLInsecureSkipVerify)
	assert.Contains(t, newProps.props, pathSSLCipherSuites)
	assert.Contains(t, newProps.props, pathSSLMinVersion)
	assert.Contains(t, newProps.props, pathSSLMaxVersion)
	assert.Contains(t, newProps.props, pathPollInterval)
	assert.Contains(t, newProps.props, pathProxyURL)
	assert.Contains(t, newProps.props, pathCachePath)
	assert.Contains(t, newProps.props, pathDiscoverOriginalRaml)

	// validate defaults
	cfg := NewMulesoftConfig(newProps)
	assert.Equal(t, "https://anypoint.mulesoft.com", cfg.AnypointExchangeURL)
	assert.Equal(t, "", cfg.Environment)
	assert.Equal(t, "", cfg.OrgName)
	assert.Equal(t, "", cfg.DiscoveryTags)
	assert.Equal(t, "", cfg.DiscoveryIgnoreTags)
	assert.Equal(t, "", cfg.ClientID)
	assert.Equal(t, "", cfg.ClientSecret)
	assert.Equal(t, 60*time.Minute, cfg.SessionLifetime)
	assert.Equal(t, []string{}, cfg.TLS.GetNextProtos())
	assert.Equal(t, false, cfg.TLS.IsInsecureSkipVerify())
	assert.Equal(t, corecfg.NewCipherArray(corecfg.TLSDefaultCipherSuitesStringSlice()), cfg.TLS.GetCipherSuites())
	assert.Equal(t, corecfg.TLSVersionAsValue(corecfg.TLSDefaultMinVersionString()), cfg.TLS.GetMinVersion())
	assert.Equal(t, corecfg.TLSVersionAsValue("0"), cfg.TLS.GetMaxVersion())
	assert.Equal(t, time.Minute, cfg.PollInterval)
	assert.Equal(t, "", cfg.ProxyURL)
	assert.Equal(t, "/tmp", cfg.CachePath)
	assert.Equal(t, false, cfg.DiscoverOriginalRaml)

	// validate changed values
	newProps.props[pathAnypointExchangeURL] = propData{"string", "", "ok.com"}
	newProps.props[pathEnvironment] = propData{"string", "", "env"}
	newProps.props[pathOrgName] = propData{"string", "", "orgName"}
	newProps.props[pathDiscoveryTags] = propData{"string", "", "tag1"}
	newProps.props[pathDiscoveryIgnoreTags] = propData{"string", "", "tag-ignore"}
	newProps.props[pathAuthClientID] = propData{"string", "", "clientID"}
	newProps.props[pathAuthClientSecret] = propData{"string", "", "clientSecret"}
	newProps.props[pathAuthLifetime] = propData{"duration", "", time.Minute * 20}
	newProps.props[pathSSLNextProtos] = propData{"[]string", "", []string{"sslNextProtos1", "sslNextProtos2"}}
	newProps.props[pathSSLInsecureSkipVerify] = propData{"bool", "", true}
	newProps.props[pathSSLCipherSuites] = propData{"[]string", "", []string{"ECDHE-ECDSA-AES-128-CBC-SHA", "ECDHE-ECDSA-AES-128-CBC-SHA256", "ECDHE-ECDSA-AES-128-GCM-SHA256"}}
	newProps.props[pathSSLMinVersion] = propData{"string", "", "TLS1.0"}
	newProps.props[pathSSLMaxVersion] = propData{"string", "", "TLS1.2"}
	newProps.props[pathPollInterval] = propData{"duration", "", time.Minute * 20}
	newProps.props[pathProxyURL] = propData{"string", "", "proxy.ok.com"}
	newProps.props[pathCachePath] = propData{"string", "", "./config"}
	newProps.props[pathDiscoverOriginalRaml] = propData{"bool", "", true}

	cfg = NewMulesoftConfig(newProps)
	assert.Equal(t, "ok.com", cfg.AnypointExchangeURL)
	assert.Equal(t, "env", cfg.Environment)
	assert.Equal(t, "orgName", cfg.OrgName)
	assert.Equal(t, "tag1", cfg.DiscoveryTags)
	assert.Equal(t, "tag-ignore", cfg.DiscoveryIgnoreTags)
	assert.Equal(t, "clientID", cfg.ClientID)
	assert.Equal(t, "clientSecret", cfg.ClientSecret)
	assert.Equal(t, time.Minute*20, cfg.SessionLifetime)
	assert.Equal(t, []string{"sslNextProtos1", "sslNextProtos2"}, cfg.TLS.GetNextProtos())
	assert.Equal(t, true, cfg.TLS.IsInsecureSkipVerify())
	assert.Equal(t, corecfg.NewCipherArray([]string{"ECDHE-ECDSA-AES-128-CBC-SHA", "ECDHE-ECDSA-AES-128-CBC-SHA256", "ECDHE-ECDSA-AES-128-GCM-SHA256"}), cfg.TLS.GetCipherSuites())
	assert.Equal(t, corecfg.TLSVersionAsValue("TLS1.0"), cfg.TLS.GetMinVersion())
	assert.Equal(t, corecfg.TLSVersionAsValue("TLS1.2"), cfg.TLS.GetMaxVersion())
	assert.Equal(t, time.Minute*20, cfg.PollInterval)
	assert.Equal(t, "proxy.ok.com", cfg.ProxyURL)
	assert.Equal(t, "./config", cfg.CachePath)
	assert.Equal(t, true, cfg.DiscoverOriginalRaml)
}
