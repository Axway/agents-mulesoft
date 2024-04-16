package discovery

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/Axway/agent-sdk/pkg/cache"
	"github.com/Axway/agent-sdk/pkg/util/oas"
	"github.com/Axway/agents-mulesoft/pkg/common"
	"gopkg.in/yaml.v2"

	"github.com/sirupsen/logrus"

	subs "github.com/Axway/agents-mulesoft/pkg/subscription"
	"github.com/getkin/kin-openapi/openapi2"

	"github.com/getkin/kin-openapi/openapi3"

	"github.com/Axway/agent-sdk/pkg/apic"
	sdkUtil "github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agents-mulesoft/pkg/anypoint"
	"github.com/Axway/agents-mulesoft/pkg/config"
)

const (
	marketplace = "marketplace"
	catalog     = "unified-catalog"
)

// ServiceHandler converts a mulesoft asset to an array of ServiceDetails
type ServiceHandler interface {
	ToServiceDetails(asset *anypoint.Asset) []*ServiceDetail
	OnConfigChange(cfg *config.MulesoftConfig)
}

type serviceHandler struct {
	muleEnv              string
	discoveryTags        []string
	discoveryIgnoreTags  []string
	client               anypoint.Client
	schemas              subs.SchemaStore
	cache                cache.Cache
	discoverOriginalRaml bool
}

func (s *serviceHandler) OnConfigChange(cfg *config.MulesoftConfig) {
	s.discoveryTags = cleanTags(cfg.DiscoveryTags)
	s.discoveryIgnoreTags = cleanTags(cfg.DiscoveryIgnoreTags)
	s.muleEnv = cfg.Environment
}

// ToServiceDetails gathers the ServiceDetail for a single Mulesoft Asset. Each Asset has multiple versions and
// can resolve to multiple ServiceDetails.
func (s *serviceHandler) ToServiceDetails(asset *anypoint.Asset) []*ServiceDetail {
	var serviceDetails []*ServiceDetail
	logger := logrus.WithFields(logrus.Fields{
		"assetName": asset.AssetID,
		"assetID":   asset.ID,
	})

	for _, api := range asset.APIs {
		logger = logger.
			WithField("apiID", api.ID).
			WithField("apiAssetVersion", api.AssetVersion)

		if ok, msg := shouldDiscoverAPI(api.EndpointURI, s.discoveryTags, s.discoveryIgnoreTags, api.Tags); !ok {
			logger.WithField("endpoint", api.EndpointURI).Debugf("skipping discovery. %s", msg)
			continue
		}

		serviceDetail, err := s.getServiceDetail(asset, &api)
		if err != nil {
			logger.Errorf("error getting the service details: %s", err.Error())
			continue
		}
		if serviceDetail != nil {
			serviceDetails = append(serviceDetails, serviceDetail)
		}
	}
	return serviceDetails
}

// getServiceDetail gets the ServiceDetail for the API asset.
func (s *serviceHandler) getServiceDetail(asset *anypoint.Asset, api *anypoint.API) (*ServiceDetail, error) {
	api.ActiveContractsCount = 0
	logger := logrus.WithFields(logrus.Fields{
		"assetName":       asset.AssetID,
		"assetID":         asset.ID,
		"apiID":           api.ID,
		"apiAssetVersion": api.AssetVersion,
	})

	// Get the policies associated with the API
	policies, err := s.client.GetPolicies(api.ID)
	if err != nil {
		return nil, err
	}
	apicAuths, configuration, err := getApicAuthsAndConfig(policies)
	if err != nil {
		return nil, err
	}

	isAlreadyPublished, checksum := isPublished(api, configuration, s.cache)
	// If true, then the api is published and there were no changes detected
	if isAlreadyPublished {
		logger.Debug("api is already published")
		return nil, nil
	}
	logger = logger.WithField("authTypes", apicAuths)

	secondaryKey := common.FormatAPICacheKey(fmt.Sprint(api.ID), api.ProductVersion)
	// Setting with the checksum allows a way to see if the item changed.
	// Setting with the secondary key allows the subscription manager to find the api.
	err = s.cache.SetWithSecondaryKey(checksum, secondaryKey, *api)
	if err != nil {
		logger.Errorf("failed to save api to cache: %s", err)
	}

	crds := []string{}
	ard := ""
	apicAuthsToCRDMapper := map[string]string{
		apic.Oauth: provisioning.OAuthSecretCRD,
		apic.Basic: provisioning.BasicAuthCRD,
	}
	for _, auth := range apicAuths {
		if crd, ok := apicAuthsToCRDMapper[auth]; ok {
			ard = provisioning.APIKeyARD
			crds = []string{crd}
		}
	}

	exchangeAsset, err := s.client.GetExchangeAsset(api.GroupID, api.AssetID, api.AssetVersion)
	if err != nil {
		return nil, err
	}

	exchFile := getExchangeAssetSpecFile(exchangeAsset.Files, s.discoverOriginalRaml)
	if exchFile == nil {
		logger.Debugf("no supported specification file found")
		return nil, nil
	}

	rawSpec, wasConverted, err := s.client.GetExchangeFileContent(exchFile.ExternalLink, exchFile.Packaging, exchFile.MainFile, s.discoverOriginalRaml)
	if err != nil {
		return nil, err
	}
	if wasConverted {
		api.Tags = append(api.Tags, "converted-from-raml")
	}

	parser := apic.NewSpecResourceParser(rawSpec, "")
	parser.Parse()

	modifiedSpec, err := updateSpec(parser.GetSpecProcessor(), api.EndpointURI, configuration)
	if err != nil {
		return nil, err
	}

	icon, iconContentType, err := s.client.GetExchangeAssetIcon(exchangeAsset.Icon)
	if err != nil {
		return nil, err
	}

	status := apic.PublishedStatus
	if api.Deprecated {
		status = apic.DeprecatedStatus
	}

	return &ServiceDetail{
		ARD:         ard,
		CRDs:        crds,
		APIName:     api.AssetID,
		APISpec:     modifiedSpec,
		Description: api.Description,
		// Use the Asset ID for the externalAPIID so that apis linked to the asset are created as a revision
		ID:                fmt.Sprint(asset.ID),
		Image:             icon,
		ImageContentType:  iconContentType,
		ResourceType:      parser.GetSpecProcessor().GetResourceType(),
		ServiceAttributes: map[string]string{},
		AgentDetails: map[string]string{
			common.AttrAssetID:        fmt.Sprint(asset.ID),
			common.AttrAPIID:          fmt.Sprint(api.ID),
			common.AttrAssetVersion:   api.AssetVersion,
			common.AttrChecksum:       checksum,
			common.AttrProductVersion: api.ProductVersion,
		},
		Stage:            api.AssetVersion,
		Tags:             api.Tags,
		Title:            asset.ExchangeAssetName,
		Version:          api.AssetVersion,
		SubscriptionName: "",
		Status:           status,
	}, nil
}

// shouldDiscoverAPI determines if the API should be pushed to Central or not
func shouldDiscoverAPI(endpoint string, discoveryTags, ignoreTags, apiTags []string) (bool, string) {
	if endpoint == "" {
		// If the API has no exposed endpoint we're not going to discover it.
		return false, "no consumer endpoint configured"
	}

	if doesAPIContainAnyMatchingTag(ignoreTags, apiTags) {
		return false, "api contains tag found in the ignoreTags list"
	}

	if len(discoveryTags) > 0 {
		if !doesAPIContainAnyMatchingTag(discoveryTags, apiTags) {
			return false, "api does not contain necessary tags for discovery"
		}
	}

	return true, ""
}

// updateSpec Updates the spec endpoints based on the given type.
func updateSpec(processor apic.SpecProcessor, endpointURI string, configuration map[string]interface{}) ([]byte, error) {
	var err error
	var oas2Swagger *openapi2.T
	var oas3Swagger *openapi3.T
	specType := processor.GetResourceType()
	specBytes := processor.GetSpecBytes()

	switch specType {
	case apic.Oas2:
		oas2Swagger, err = oas.ParseOAS2(specBytes)
		if err != nil {
			return nil, err
		}
		err = oas.SetOAS2HostDetails(oas2Swagger, endpointURI)
		if err != nil {
			logrus.Debugf("failed to update the spec with the given endpoint: %s", endpointURI)
		}
		specBytes, err = setOAS2policies(oas2Swagger, configuration)

	case apic.Oas3:
		oas3Swagger, err = oas.ParseOAS3(specBytes)
		if err != nil {
			return nil, err
		}
		oas.SetOAS3Servers([]string{endpointURI}, oas3Swagger)
		specBytes, err = setOAS3policies(oas3Swagger, configuration)

	case apic.Raml:
		specBytes, err = setRamlHostAndAuth(specBytes, endpointURI, configuration)
	}

	return specBytes, err
}

// getExchangeAssetSpecFile gets the file entry for the Assets spec.
func getExchangeAssetSpecFile(exchangeFiles []anypoint.ExchangeFile, discoverOriginalRaml bool) *anypoint.ExchangeFile {
	if len(exchangeFiles) == 0 {
		return nil
	}
	sort.Sort(BySpecType(exchangeFiles))

	if discoverOriginalRaml {
		return getExchangeAssetWithRamlSpecFile(exchangeFiles)
	}
	// By default, the RAML spec will have a download link with it as already converted to OAS but have an empty MainFile field
	if exchangeFiles[0].Classifier != "oas" &&
		exchangeFiles[0].Classifier != "fat-oas" &&
		exchangeFiles[0].Classifier != "wsdl" {
		// Unsupported spec type
		return nil
	}
	return &exchangeFiles[0]
}

func getExchangeAssetWithRamlSpecFile(exchangeFiles []anypoint.ExchangeFile) *anypoint.ExchangeFile {
	for i := range exchangeFiles {
		c := exchangeFiles[i].Classifier
		if _, found := specPreference[c]; found && exchangeFiles[i].MainFile != "" {
			return &exchangeFiles[i]
		}
	}
	// Unsupported spec type
	return nil
}

// gets the API Central Authentication types based on the Mulesoft policy type.
func getApicAuthsAndConfig(policies []anypoint.Policy) ([]string, map[string]interface{}, error) {
	apicAuths := []string{}
	configs := map[string]interface{}{}
	for _, policy := range policies {
		if policy.PolicyTemplateID == common.OAuth2MuleOauthProviderPolicy {
			configs[apic.Oauth] = getMapFromInterface(policy.Configuration)
			apicAuths = append(apicAuths, apic.Oauth)
		} else if policy.PolicyTemplateID == common.BasicAuthSimplePolicy {
			configs[apic.Basic] = getMapFromInterface(policy.Configuration)
			apicAuths = append(apicAuths, apic.Basic)
		} else if policy.PolicyTemplateID == common.ClientIDEnforcementPolicy {
			config := getMapFromInterface(policy.Configuration)
			val, ok := config[common.CredOrigin]
			if !ok {
				continue
			}
			if v, ok := val.(string); ok && v != "customExpression" {
				configs[apic.Basic] = config
				apicAuths = append(apicAuths, apic.Basic)
			} else {
				return nil, nil, fmt.Errorf("incompatible Mulesoft Policies provided")
			}
		}
	}

	return sdkUtil.RemoveDuplicateValuesFromStringSlice(apicAuths), configs, nil
}

// makeChecksum generates a makeChecksum for the api for change detection
func makeChecksum(val interface{}, cfg interface{}) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%v%s", val, cfg)))
	return fmt.Sprintf("%x", sum)
}

// doesAPIContainAnyMatchingTag checks if the API has any of the tags
func doesAPIContainAnyMatchingTag(tags, apiTags []string) bool {
	for _, apiTag := range apiTags {
		apiTag = strings.ToLower(apiTag)
		for _, tag := range tags {
			if tag == apiTag {
				return true
			}
		}
	}
	return false
}

func setOAS2policies(swagger *openapi2.T, configuration map[string]interface{}) ([]byte, error) {
	// remove existing security
	swagger.SecurityDefinitions = make(map[string]*openapi2.SecurityScheme)
	swagger.Security = openapi2.SecurityRequirements{}
	for auth, config := range configuration {
		switch auth {
		case apic.Basic:
			ss := openapi2.SecurityScheme{
				Type:        common.BasicAuthScheme,
				Description: common.BasicAuthDesc,
			}
			swagger.SecurityDefinitions[common.BasicAuthName] = &ss
			swagger.Security = append(swagger.Security, map[string][]string{
				common.BasicAuthName: []string{},
			})

		case apic.Oauth:
			tokenURL := ""
			scopes := make(map[string]string)
			if cfg := config.(map[string]interface{})[common.TokenURL]; cfg != nil {
				tokenURL = cfg.(string)
			}
			if cfg := config.(map[string]interface{})[common.Scopes]; cfg != nil {
				// Mulesoft scopes should come separated by space (it's specified when you add scopes in the UI)
				scopesSlice := strings.Split(cfg.(string), " ")
				for _, scope := range scopesSlice {
					scopes[scope] = ""
				}
				swagger.Security = append(swagger.Security, map[string][]string{
					common.Oauth2Name: scopesSlice,
				})
			} else {
				swagger.Security = append(swagger.Security, map[string][]string{
					common.Oauth2Name: []string{},
				})
			}

			ss := openapi2.SecurityScheme{
				Description:      common.Oauth2Desc,
				Type:             common.Oauth2OASType,
				Flow:             common.AccessCode,
				AuthorizationURL: tokenURL,
				TokenURL:         tokenURL,
				Scopes:           scopes,
			}
			swagger.SecurityDefinitions[common.Oauth2Name] = &ss
		}
	}

	return json.Marshal(swagger)
}

func setOAS3policies(spec *openapi3.T, configuration map[string]interface{}) ([]byte, error) {
	// remove existing security
	spec.Components.SecuritySchemes = make(openapi3.SecuritySchemes)
	spec.Security = *openapi3.NewSecurityRequirements()
	for auth, config := range configuration {
		switch auth {
		case apic.Basic:
			ssr := openapi3.SecuritySchemeRef{
				Value: &openapi3.SecurityScheme{
					Type:        common.BasicAuthOASType,
					Scheme:      common.BasicAuthScheme,
					Description: common.BasicAuthDesc,
				},
			}

			spec.Components.SecuritySchemes[common.BasicAuthName] = &ssr
			spec.Security = *spec.Security.With(openapi3.NewSecurityRequirement().Authenticate(common.BasicAuthName, ""))
		case apic.Oauth:
			tokenURL := ""
			scopes := make(map[string]string)

			if cfg := config.(map[string]interface{})[common.TokenURL]; cfg != nil {
				tokenURL = cfg.(string)
			}
			if cfg := config.(map[string]interface{})[common.Scopes]; cfg != nil {
				// Mulesoft scopes should come separated by space (it's defined when you add scopes in the UI)
				scopesSlice := strings.Split(cfg.(string), " ")
				for _, scope := range scopesSlice {
					scopes[scope] = ""
				}
				spec.Security = *spec.Security.With(
					openapi3.NewSecurityRequirement().Authenticate(
						common.Oauth2Name, scopesSlice...),
				)
			}

			ssr := openapi3.SecuritySchemeRef{
				Value: &openapi3.SecurityScheme{
					Type:        common.Oauth2OASType,
					Description: common.Oauth2Desc,
					Flows: &openapi3.OAuthFlows{
						AuthorizationCode: &openapi3.OAuthFlow{
							TokenURL:         tokenURL,
							AuthorizationURL: tokenURL,
							Scopes:           scopes,
						},
					},
				},
			}
			spec.Components.SecuritySchemes[common.Oauth2Name] = &ssr
		}
	}

	return json.Marshal(spec)
}

func setRamlHostAndAuth(spec []byte, endpoint string, configuration map[string]interface{}) ([]byte, error) {
	var ramlDef map[string]interface{}
	// We know that this is a valid raml file from the parser, so this never fails. We need this because yaml unmarshal drops the version line
	ramlVersion := append(spec[0:10], []byte("\n")...)
	yaml.Unmarshal(spec, &ramlDef)

	securitySchemes := map[string]interface{}{}
	securedBy := []interface{}{}
	for auth, config := range configuration {
		switch auth {
		case apic.Basic:
			securitySchemes[common.BasicAuthName] = map[string]interface{}{
				"description": common.BasicAuthDesc,
				"type":        common.BasicAuthRAMLType,
			}
			securedBy = append(securedBy, common.BasicAuthName)
		case apic.Oauth:
			tokenURL := ""
			if token := config.(map[string]interface{})[common.TokenURL]; token != nil {
				tokenURL = token.(string)
			}

			oAuthSettings := map[string]interface{}{
				"accessTokenUri":   tokenURL,
				"authorizationUri": tokenURL,
			}
			if s := config.(map[string]interface{})[common.Scopes]; s != nil {
				// formats correctly for raml securedBy format
				scopesStr := s.(string)
				scopesSlice := strings.Split(scopesStr, " ")
				securedBy = append(securedBy,
					map[string]interface{}{
						common.Oauth2Name: map[string]interface{}{
							"scopes": scopesSlice,
						},
					},
				)
			} else {
				// if no scopes defined, this means that the security is applied globally
				securedBy = append(securedBy, common.Oauth2Name)
			}

			securitySchemes[common.Oauth2Name] = map[string]interface{}{
				"description": common.Oauth2Desc,
				"type":        common.Oauth2RAMLType,
				"settings":    oAuthSettings,
				"describedBy": map[string]interface{}{
					"headers": map[string]interface{}{
						"Authorization": map[string]interface{}{
							"description": common.Oauth2Desc,
							"type":        "string",
						},
					},
				},
			}
		}
	}
	ramlDef["baseUri"] = endpoint
	ramlDef["securitySchemes"] = securitySchemes
	ramlDef["securedBy"] = securedBy

	modifiedSpec, err := yaml.Marshal(ramlDef)
	if err != nil {
		return spec, err
	}
	return append(ramlVersion, modifiedSpec...), nil
}

// isPublished checks if an api is published with the latest changes. Returns true if it is, and false if it is not.
func isPublished(api *anypoint.API, configuration map[string]interface{}, c cache.Cache) (bool, string) {
	// Change detection (asset + policies)
	checksum := makeChecksum(api, configuration)
	item, err := c.Get(checksum)
	if err != nil || item == nil {
		return false, checksum
	}

	return true, ""
}

func getMapFromInterface(item interface{}) map[string]interface{} {
	conf, ok := item.(map[string]interface{})
	if !ok {
		logrus.Errorf("unable to perform type assertion on %#v", item)
		return map[string]interface{}{}
	}
	return conf
}
