package discovery

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
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

	"github.com/Axway/agent-sdk/pkg/agent"

	"github.com/Axway/agent-sdk/pkg/apic"
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
	mode                 string
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
	authPolicy, configuration, isSLABased := getAuthPolicy(policies, s.mode)
	logger = logger.WithField("policy", authPolicy)

	isAlreadyPublished, checksum := isPublished(api, authPolicy, s.cache)
	// If true, then the api is published and there were no changes detected
	if isAlreadyPublished {
		logger.Debug("api is already published")
		return nil, nil
	}

	secondaryKey := common.FormatAPICacheKey(fmt.Sprint(api.ID), api.ProductVersion)
	// Setting with the checksum allows a way to see if the item changed.
	// Setting with the secondary key allows the subscription manager to find the api.
	err = s.cache.SetWithSecondaryKey(checksum, secondaryKey, *api)
	if err != nil {
		logger.Errorf("failed to save api to cache: %s", err)
	}

	apiID := strconv.FormatInt(api.ID, 10)

	var crds []string
	subSchName := ""
	ard := ""
	if s.mode == marketplace {
		if authPolicy == apic.Oauth {
			ard = provisioning.APIKeyARD
			crds = []string{provisioning.OAuthSecretCRD}
		}
	} else {
		subSchName = s.schemas.GetSubscriptionSchemaName(common.PolicyDetail{
			Policy:     authPolicy,
			IsSLABased: isSLABased,
			APIId:      apiID,
		})

		// If the API has a new SLA Tier policy, create a new subscription schema for it
		if subSchName == "" && isSLABased {
			// Get details of the SLA tiers
			tiers, err1 := s.client.GetSLATiers(api.ID)
			if err1 != nil {
				return nil, err1
			}
			schema, err1 := s.createSLATierSchema(apiID, tiers, agent.GetCentralClient())
			if err1 != nil {
				return nil, err1
			}

			logger.Infof("schema registered")

			subSchName = schema.GetSubscriptionName()
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
	processor := parser.GetSpecProcessor()

	modifiedSpec, err := updateSpec(processor.GetResourceType(), api.EndpointURI, authPolicy, configuration, processor.GetSpecBytes())
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
		AccessRequestDefinition: ard,
		CRDs:                    crds,
		APIName:                 api.AssetID,
		APISpec:                 modifiedSpec,
		AuthPolicy:              authPolicy,
		Description:             api.Description,
		// Use the Asset ID for the externalAPIID so that apis linked to the asset are created as a revision
		ID:                fmt.Sprint(asset.ID),
		Image:             icon,
		ImageContentType:  iconContentType,
		ResourceType:      processor.GetResourceType(),
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
		SubscriptionName: subSchName,
		Status:           status,
	}, nil
}

func (s *serviceHandler) createSLATierSchema(
	apiID string, tiers *anypoint.Tiers, client apic.Client,
) (apic.SubscriptionSchema, error) {
	var tierNames []string
	for _, tier := range tiers.Tiers {
		t := fmt.Sprintf("%v-%s", tier.ID, tier.Name)
		tierNames = append(tierNames, t)
	}

	slaSchema := subs.NewSLATierContractSchemaUC(apiID, tierNames)
	s.schemas.RegisterNewSchema(slaSchema)
	schema := slaSchema.Schema()

	// Register the consumer subscription definition with the agent-sdk subscription manager
	if err := client.RegisterSubscriptionSchema(schema, true); err != nil {
		return nil, fmt.Errorf("failed to register sla tier subscription schema for api %s: %w", apiID, err)
	}

	mpSchema := subs.NewSLATierContractSchemaMP(apiID, tierNames)
	agent.NewAccessRequestBuilder().
		SetName(schema.GetSubscriptionName()).
		SetRequestSchema(mpSchema).
		Register()

	return schema, nil
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
func updateSpec(
	specType, endpointURI, authPolicy string, configuration map[string]interface{}, specContent []byte,
) ([]byte, error) {
	var err error
	var oas2Swagger *openapi2.T
	var oas3Swagger *openapi3.T

	switch specType {
	case apic.Oas2:
		oas2Swagger, err = oas.ParseOAS2(specContent)
		if err != nil {
			return nil, err
		}
		err = oas.SetOAS2HostDetails(oas2Swagger, endpointURI)
		if err != nil {
			logrus.Debugf("failed to update the spec with the given endpoint: %s", endpointURI)
		}
		specContent, err = setOAS2policies(oas2Swagger, authPolicy, configuration)
	case apic.Oas3:
		oas3Swagger, err = oas.ParseOAS3(specContent)
		if err != nil {
			return nil, err
		}
		oas.SetOAS3Servers([]string{endpointURI}, oas3Swagger)
		specContent, err = setOAS3policies(oas3Swagger, authPolicy, configuration)
	case apic.Raml:
		specContent, err = setRamlEndpoints(specContent, endpointURI)
	}

	return specContent, err
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

// getAuthPolicy gets the authentication policy type.
func getAuthPolicy(policies []anypoint.Policy, mode string) (string, map[string]interface{}, bool) {
	authPolicy := apic.Apikey
	if mode == marketplace {
		authPolicy = apic.Oauth
	}

	for _, policy := range policies {
		if policy.PolicyTemplateID == common.ClientIDEnforcement {
			conf := getMapFromInterface(policy.Configuration)
			return authPolicy, conf, false
		}

		if strings.Contains(policy.PolicyTemplateID, common.SLABased) {
			conf := getMapFromInterface(policy.Configuration)
			return authPolicy, conf, true
		}

		if policy.PolicyTemplateID == common.ExternalOauth {
			conf := getMapFromInterface(policy.Configuration)
			return apic.Oauth, conf, false
		}
	}

	return apic.Passthrough, map[string]interface{}{}, false
}

func setRamlEndpoints(spec []byte, endpoints string) ([]byte, error) {
	var ramlDef map[string]interface{}
	// We know that this is a valid raml file from the parser, so this is never fails. We need this because yaml unmarshal drops the version line
	ramlVersion := append(spec[0:10], []byte("\n")...)
	yaml.Unmarshal(spec, &ramlDef)
	ramlDef["baseUri"] = endpoints

	modifiedSpec, err := yaml.Marshal(ramlDef)
	if err != nil {
		return spec, err
	}
	return append(ramlVersion, modifiedSpec...), nil
}

// makeChecksum generates a makeChecksum for the api for change detection
func makeChecksum(val interface{}, authPolicy string) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%v%s", val, authPolicy)))
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

func setOAS2policies(swagger *openapi2.T, authPolicy string, configuration map[string]interface{}) ([]byte, error) {
	// remove existing security
	swagger.SecurityDefinitions = make(map[string]*openapi2.SecurityScheme)
	switch authPolicy {
	case apic.Apikey:
		desc := common.DescClientCred
		if configuration[common.CredOrigin] != nil {
			desc += configuration[common.CredOrigin].(string)
		}

		ss := openapi2.SecurityScheme{
			Type:        common.APIKey,
			Name:        common.Authorization,
			In:          common.Header,
			Description: desc,
		}

		sd := map[string]*openapi2.SecurityScheme{
			common.ClientIDEnforcement: &ss,
		}
		swagger.SecurityDefinitions = sd

	case apic.Oauth:
		var tokenURL string
		scopes := make(map[string]string)

		if configuration[common.TokenURL] != nil {
			tokenURL = configuration[common.TokenURL].(string)
		}

		if configuration[common.Scopes] != nil {
			scopes[common.Scopes] = configuration[common.Scopes].(string)
		}

		ssp := openapi2.SecurityScheme{
			Description:      common.DescOauth2,
			Type:             common.Oauth2,
			Flow:             common.AccessCode,
			TokenURL:         tokenURL,
			AuthorizationURL: tokenURL,
			Scopes:           scopes,
		}

		sd := map[string]*openapi2.SecurityScheme{
			common.Oauth2: &ssp,
		}

		swagger.SecurityDefinitions = sd
	}
	return json.Marshal(swagger)
}

func setOAS3policies(spec *openapi3.T, authPolicy string, configuration map[string]interface{}) ([]byte, error) {
	// remove existing security
	spec.Components.SecuritySchemes = make(openapi3.SecuritySchemes)
	switch authPolicy {
	case apic.Apikey:
		desc := common.DescClientCred
		if configuration[common.CredOrigin] != nil {
			desc += configuration[common.CredOrigin].(string)
		}

		ss := openapi3.SecurityScheme{
			Type:        common.APIKey,
			Name:        common.Authorization,
			In:          common.Header,
			Description: desc}

		ssr := openapi3.SecuritySchemeRef{
			Value: &ss,
		}

		spec.Components.SecuritySchemes = openapi3.SecuritySchemes{common.ClientIDEnforcement: &ssr}
	case apic.Oauth:
		var tokenURL string
		scopes := make(map[string]string)

		if configuration[common.TokenURL] != nil {
			var ok bool
			tokenURL, ok = configuration[common.TokenURL].(string)
			if !ok {
				return nil, fmt.Errorf("unable to perform type assertion on %#v", configuration[common.TokenURL])
			}
		}

		if configuration[common.Scopes] != nil {
			scopes[common.Scopes] = configuration[common.Scopes].(string)
		}

		ac := openapi3.OAuthFlow{
			TokenURL:         tokenURL,
			AuthorizationURL: tokenURL,
			Scopes:           scopes,
		}
		oAuthFlow := openapi3.OAuthFlows{
			AuthorizationCode: &ac,
		}
		ss := openapi3.SecurityScheme{
			Type:        common.Oauth2,
			Description: common.DescOauth2,
			Flows:       &oAuthFlow,
		}
		ssr := openapi3.SecuritySchemeRef{
			Value: &ss,
		}

		spec.Components.SecuritySchemes = openapi3.SecuritySchemes{common.Oauth2: &ssr}
	}

	return json.Marshal(spec)
}

// isPublished checks if an api is published with the latest changes. Returns true if it is, and false if it is not.
func isPublished(api *anypoint.API, authPolicy string, c cache.Cache) (bool, string) {
	// Change detection (asset + policies)
	checksum := makeChecksum(api, authPolicy)
	item, err := c.Get(checksum)
	if err != nil || item == nil {
		return false, checksum
	}

	return true, checksum
}

func getMapFromInterface(item interface{}) map[string]interface{} {
	conf, ok := item.(map[string]interface{})
	if !ok {
		logrus.Errorf("unable to perform type assertion on %#v", item)
		return map[string]interface{}{}
	}
	return conf
}
