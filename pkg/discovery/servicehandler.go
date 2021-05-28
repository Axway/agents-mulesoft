package discovery

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/Axway/agents-mulesoft/pkg/common"

	"github.com/Axway/agent-sdk/pkg/cache"
	"github.com/Axway/agent-sdk/pkg/util/oas"

	"github.com/sirupsen/logrus"

	"github.com/Axway/agents-mulesoft/pkg/subscription/slatier"

	"github.com/Axway/agents-mulesoft/pkg/subscription"
	"github.com/getkin/kin-openapi/openapi2"

	"github.com/getkin/kin-openapi/openapi3"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/util/log"

	"github.com/Axway/agent-sdk/pkg/apic"
	"github.com/Axway/agents-mulesoft/pkg/anypoint"
	"github.com/Axway/agents-mulesoft/pkg/config"
	"sigs.k8s.io/yaml"
)

// ServiceHandler converts a mulesoft asset to an array of ServiceDetails
type ServiceHandler interface {
	ToServiceDetails(asset *anypoint.Asset) []*ServiceDetail
	OnConfigChange(cfg *config.MulesoftConfig)
}

type serviceHandler struct {
	muleEnv             string
	discoveryTags       []string
	discoveryIgnoreTags []string
	client              anypoint.Client
	subscriptionManager subscription.SchemaHandler
	cache               cache.Cache
}

func (s *serviceHandler) OnConfigChange(cfg *config.MulesoftConfig) {
	s.discoveryTags = cleanTags(cfg.DiscoveryTags)
	s.discoveryIgnoreTags = cleanTags(cfg.DiscoveryIgnoreTags)
	s.muleEnv = cfg.Environment
}

// ToServiceDetails gathers the ServiceDetail for a single Mulesoft Asset. Each Asset has multiple versions and
// can resolve to multiple ServiceDetails.
func (s *serviceHandler) ToServiceDetails(asset *anypoint.Asset) []*ServiceDetail {
	serviceDetails := []*ServiceDetail{}
	logger := logrus.WithFields(logrus.Fields{
		"assetName": asset.AssetID,
		"assetID":   asset.ID,
	})
	for _, api := range asset.APIs {
		logger.
			WithField("apiID", api.ID).
			WithField("apiAssetVersion", api.AssetVersion)

		if !shouldDiscoverAPI(api.EndpointURI, s.discoveryTags, s.discoveryIgnoreTags, api.Tags) {
			logger.WithField("endpoint", api.EndpointURI).Debug("skipping discovery for api")
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
	authPolicy, configuration, isSlaBased := getAuthPolicy(policies)

	// TODO can be refactored to not use authPolicy in checksum and use policy
	isAlreadyPublished, checksum := isPublished(api, authPolicy, s.cache)
	// If true, then the api is published and there were no changes detected
	if isAlreadyPublished {
		logger.WithFields(logrus.Fields{"policy": authPolicy, "msg": "api is already published"})
		return nil, nil
	}

	// TODO Handle purging of cache
	secondaryKey := common.FormatAPICacheKey(fmt.Sprint(api.ID), api.ProductVersion)
	// Setting with the checksum allows a way to see if the item changed.
	// Setting with the secondary key allows the subscription manager to find the api.
	err = s.cache.SetWithSecondaryKey(checksum, secondaryKey, *api)
	if err != nil {
		logger.Error(err)
	}

	apiID := strconv.FormatInt(api.ID, 10)

	subSchName := s.subscriptionManager.GetSubscriptionSchemaName(config.PolicyDetail{
		Policy:     authPolicy,
		IsSlaBased: isSlaBased,
		APIId:      apiID,
	})

	// If the API has a new SLA Tier policy, create a new subscription schema for it
	if subSchName == "" && isSlaBased {
		// Get details of the SLA tiers
		tiers, err1 := s.client.GetSLATiers(api.ID)
		if err1 != nil {
			return nil, err1
		}
		schema, err1 := s.createSubscriptionSchemaForSLATier(apiID, tiers, agent.GetCentralClient())
		if err1 != nil {
			return nil, err1
		}

		subSchName = schema.GetSubscriptionName()
	}

	exchangeAsset, err := s.client.GetExchangeAsset(api.GroupID, api.AssetID, api.AssetVersion)
	if err != nil {
		return nil, err
	}

	exchFile := getExchangeAssetSpecFile(exchangeAsset.Files)

	if exchFile == nil {
		// SDK needs a spec
		logger.Debugf("no supported specification file found")
		return nil, nil
	}

	rawSpec, err := s.client.GetExchangeFileContent(exchFile.ExternalLink, exchFile.Packaging, exchFile.MainFile)
	if err != nil {
		return nil, err
	}
	specContent := specYAMLToJSON(rawSpec) // SDK does not support YAML specifications
	specType, err := getSpecType(exchFile, specContent)
	if err != nil {
		return nil, err
	}
	if specType == "" {
		return nil, fmt.Errorf("unknown spec type")
	}
	modifiedSpec, err := updateSpec(specType, api.EndpointURI, authPolicy, configuration, specContent)
	if err != nil {
		return nil, err
	}

	icon, iconContentType, err := s.client.GetExchangeAssetIcon(exchangeAsset.Icon)
	if err != nil {
		return nil, err
	}

	status := apic.PublishedStatus
	if api.Deprecated == true {
		status = apic.DeprecatedStatus
	}

	return &ServiceDetail{
		APIName:          api.AssetID,
		APISpec:          modifiedSpec,
		AuthPolicy:       authPolicy,
		Description:      api.Description,
		ID:               fmt.Sprint(asset.ID),
		Image:            icon,
		ImageContentType: iconContentType,
		ResourceType:     specType,
		ServiceAttributes: map[string]string{
			common.AttrAssetID:        fmt.Sprint(asset.ID),
			common.AttrAPIID:          fmt.Sprint(api.ID),
			common.AttrAssetVersion:   api.AssetVersion,
			common.AttrChecksum:       checksum,
			common.AttrProductVersion: api.ProductVersion,
		},
		Stage:            api.ProductVersion,
		Tags:             api.Tags,
		Title:            asset.ExchangeAssetName,
		Version:          api.ProductVersion,
		SubscriptionName: subSchName,
		Status:           status,
	}, nil
}

func (s *serviceHandler) createSubscriptionSchemaForSLATier(
	apiID string,
	tiers *anypoint.Tiers,
	centralClient apic.Client,
) (apic.SubscriptionSchema, error) {

	var names []string

	for _, tier := range tiers.Tiers {
		t := fmt.Sprintf("%v-%s", tier.ID, tier.Name)
		names = append(names, t)
	}

	// Create a subscription schema to represent SLA Tiers.
	schema := apic.NewSubscriptionSchema(apiID)
	schema.AddProperty(anypoint.AppName, "string", "Name of the new app", "", true, nil)
	schema.AddProperty(anypoint.Description, "string", "", "", false, nil)
	schema.AddProperty(anypoint.TierLabel, "string", "", "", true, names)

	// Register the schema with the mule subscription manager
	s.subscriptionManager.RegisterNewSchema(slatier.NewSLATierContract(apiID, schema, s.client))

	// Register the schema with the agent-sdk subscription manager
	if err := centralClient.RegisterSubscriptionSchema(schema, true); err != nil {
		return nil, fmt.Errorf("failed to register subscription schema %s: %w", schema.GetSubscriptionName(), err)
	}

	log.Infof("Schema registered: %s", schema.GetSubscriptionName())

	return schema, nil
}

// shouldDiscoverAPI determines if the API should be pushed to Central or not
func shouldDiscoverAPI(endpoint string, discoveryTags, ignoreTags, apiTags []string) bool {
	if endpoint == "" {
		// If the API has no exposed endpoint we're not going to discover it.
		return false
	}

	if doesAPIContainAnyMatchingTag(ignoreTags, apiTags) {
		return false
	}

	if len(discoveryTags) > 0 {
		if !doesAPIContainAnyMatchingTag(discoveryTags, apiTags) {
			return false // ignore
		}
	}
	return true
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

	case apic.Wsdl:
		specContent, err = setWSDLEndpoint(endpointURI, specContent)
	}

	return specContent, err
}

// getExchangeAssetSpecFile gets the file entry for the Assets spec.
func getExchangeAssetSpecFile(exchangeFiles []anypoint.ExchangeFile) *anypoint.ExchangeFile {
	if exchangeFiles == nil || len(exchangeFiles) == 0 {
		return nil
	}

	sort.Sort(BySpecType(exchangeFiles))
	if exchangeFiles[0].Classifier != "oas" &&
		exchangeFiles[0].Classifier != "fat-oas" &&
		exchangeFiles[0].Classifier != "wsdl" {
		// Unsupported spec type
		return nil
	}
	return &exchangeFiles[0]
}

// specYAMLToJSON - if the spec is yaml convert it to json, SDK doesn't handle yaml.
func specYAMLToJSON(specContent []byte) []byte {
	specMap := make(map[string]interface{})
	// check if the content is already json
	err := json.Unmarshal(specContent, &specMap)
	if err == nil {
		return specContent
	}

	// check if the content is already yaml
	err = yaml.Unmarshal(specContent, &specMap)
	if err != nil {
		// Not yaml, nothing more to be done
		return specContent
	}

	transcoded, err := yaml.YAMLToJSON(specContent)
	if err != nil {
		// Not json encodeable, nothing more to be done
		return specContent
	}
	return transcoded
}

// getSpecType determines the correct resource type for the asset.
func getSpecType(file *anypoint.ExchangeFile, specContent []byte) (string, error) {
	if file.Classifier == apic.Wsdl {
		return apic.Wsdl, nil
	}

	if specContent != nil {
		jsonMap := make(map[string]interface{})
		err := json.Unmarshal(specContent, &jsonMap)
		if err != nil {
			return "", err
		}
		if _, isSwagger := jsonMap["swagger"]; isSwagger {
			return apic.Oas2, nil
		} else if _, isOpenAPI := jsonMap["openapi"]; isOpenAPI {
			return apic.Oas3, nil
		}
	}
	return "", nil
}

// getAuthPolicy gets the authentication policy type.
func getAuthPolicy(policies anypoint.Policies) (string, map[string]interface{}, bool) {
	for _, policy := range policies.Policies {
		if policy.Template.AssetID == anypoint.ClientID {
			conf := getMapFromInterface(policy.Configuration)
			return apic.Apikey, conf, false
		}

		if strings.Contains(policy.Template.AssetID, anypoint.SlaAuth) {
			conf := getMapFromInterface(policy.Configuration)
			return apic.Apikey, conf, true
		}

		if policy.Template.AssetID == anypoint.ExternalOauth {
			conf := getMapFromInterface(policy.Configuration)
			return apic.Oauth, conf, false
		}
	}

	return apic.Passthrough, map[string]interface{}{}, false
}

func setWSDLEndpoint(_ string, specContent []byte) ([]byte, error) {
	// TODO
	return specContent, nil
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

// TODO improve if fields are not complete, introduce per method swagger definitions, set up logic for multiple auth policies, use policy configurations
func setOAS2policies(swagger *openapi2.T, authPolicy string, configuration map[string]interface{}) ([]byte, error) {
	// remove existing security
	swagger.SecurityDefinitions = make(map[string]*openapi2.SecurityScheme)
	switch authPolicy {
	case apic.Apikey:

		desc := anypoint.DescClienCred
		if configuration[anypoint.CredOrigin] != nil {
			desc += configuration[anypoint.CredOrigin].(string)
		}

		ss := openapi2.SecurityScheme{
			Type:        anypoint.ApiKey,
			Name:        anypoint.Authorization,
			In:          anypoint.Header,
			Description: desc,
		}

		sd := map[string]*openapi2.SecurityScheme{
			anypoint.ClientID: &ss,
		}
		swagger.SecurityDefinitions = sd

	case apic.Oauth:
		var tokenUrl string
		scopes := make(map[string]string)

		if configuration[anypoint.TokenUrl] != nil {
			tokenUrl = configuration[anypoint.TokenUrl].(string)
		}

		if configuration[anypoint.Scopes] != nil {
			scopes[anypoint.Scopes] = configuration[anypoint.Scopes].(string)
		}

		ssp := openapi2.SecurityScheme{
			Description:      anypoint.DescOauth2,
			Type:             anypoint.Oauth2,
			Flow:             anypoint.AccessCode,
			TokenURL:         tokenUrl,
			AuthorizationURL: tokenUrl,
			Scopes:           scopes,
		}

		sd := map[string]*openapi2.SecurityScheme{
			anypoint.Oauth2: &ssp,
		}

		swagger.SecurityDefinitions = sd
	}
	// return sc, agenterrors.New(1161, anypoint.ErrAuthNotSupported)
	return json.Marshal(swagger)
}

func setOAS3policies(spec *openapi3.T, authPolicy string, configuration map[string]interface{}) ([]byte, error) {
	// remove existing security
	spec.Components.SecuritySchemes = make(openapi3.SecuritySchemes)
	switch authPolicy {
	case apic.Apikey:
		desc := anypoint.DescClienCred
		if configuration[anypoint.CredOrigin] != nil {
			desc += configuration[anypoint.CredOrigin].(string)
		}

		ss := openapi3.SecurityScheme{
			Type:        anypoint.ApiKey,
			Name:        anypoint.Authorization,
			In:          anypoint.Header,
			Description: desc}

		ssr := openapi3.SecuritySchemeRef{
			Value: &ss,
		}

		spec.Components.SecuritySchemes = openapi3.SecuritySchemes{anypoint.ClientID: &ssr}
	case apic.Oauth:
		var tokenUrl string
		scopes := make(map[string]string)

		if configuration[anypoint.TokenUrl] != nil {
			var ok bool
			tokenUrl, ok = configuration[anypoint.TokenUrl].(string)
			if !ok {
				return nil, fmt.Errorf("Unable to perform type assertion on %#v", configuration[anypoint.TokenUrl])
			}
		}

		if configuration[anypoint.Scopes] != nil {
			scopes[anypoint.Scopes] = configuration[anypoint.Scopes].(string)

		}

		ac := openapi3.OAuthFlow{
			TokenURL:         tokenUrl,
			AuthorizationURL: tokenUrl,
			Scopes:           scopes,
		}
		oAuthFlow := openapi3.OAuthFlows{
			AuthorizationCode: &ac,
		}
		ss := openapi3.SecurityScheme{
			Type:        anypoint.Oauth2,
			Description: anypoint.DescOauth2,
			Flows:       &oAuthFlow,
		}
		ssr := openapi3.SecuritySchemeRef{
			Value: &ss,
		}

		spec.Components.SecuritySchemes = openapi3.SecuritySchemes{anypoint.Oauth2: &ssr}
	}

	// return sc, agenterrors.New(1162, anypoint.ErrAuthNotSupported)
	return json.Marshal(spec)
}

// isPublished checks if an api is published with the latest changes. Returns true if it is, and false if it is not.
func isPublished(api *anypoint.API, authPolicy string, c cache.Cache) (bool, string) {
	// Change detection (asset + policies)
	checksum := makeChecksum(api, authPolicy)
	item, err := c.Get(checksum)
	if err != nil || item == nil {
		return false, checksum
	} else {
		return true, checksum
	}
}

func getMapFromInterface(item interface{}) map[string]interface{} {
	conf, ok := item.(map[string]interface{})
	if !ok {
		logrus.Errorf("unable to perform type assertion on %#v", item)
		return map[string]interface{}{}
	}
	return conf
}
