package discovery

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/Axway/agents-mulesoft/pkg/subscription"
	"github.com/Axway/agents-mulesoft/pkg/subscription/slatier"
	"github.com/getkin/kin-openapi/openapi2"

	"github.com/getkin/kin-openapi/openapi3"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/util/log"

	"github.com/Axway/agent-sdk/pkg/apic"
	"sigs.k8s.io/yaml"

	"github.com/sirupsen/logrus"

	"github.com/Axway/agent-sdk/pkg/cache"
	"github.com/Axway/agents-mulesoft/pkg/anypoint"
	"github.com/Axway/agents-mulesoft/pkg/config"
)

// ServiceHandler converts a mulesoft asset to an array of ServiceDetails
type ServiceHandler interface {
	ToServiceDetails(asset *anypoint.Asset) []*ServiceDetail
	OnConfigChange(cfg *config.MulesoftConfig)
}

type serviceHandler struct {
	stage               string
	discoveryTags       []string
	discoveryIgnoreTags []string
	client              anypoint.Client
	subscriptionManager subscription.SchemaHandler
}

func (s *serviceHandler) OnConfigChange(cfg *config.MulesoftConfig) {
	s.discoveryTags = cleanTags(cfg.DiscoveryTags)
	s.discoveryIgnoreTags = cleanTags(cfg.DiscoveryIgnoreTags)
	s.stage = cfg.Environment
}

// ToServiceDetails gathers the ServiceDetail for a single Mulesoft Asset. Each Asset has multiple versions and
// can resolve to multiple ServiceDetails.
func (s *serviceHandler) ToServiceDetails(asset *anypoint.Asset) []*ServiceDetail {
	serviceDetails := []*ServiceDetail{}
	log := logrus.WithFields(logrus.Fields{
		"assetName": asset.AssetID,
		"assetID":   asset.ID,
	})
	for _, api := range asset.APIs {
		fields := logrus.Fields{
			"apiID":           api.ID,
			"apiAssetVersion": api.AssetVersion,
		}

		key := formatCacheKey(fmt.Sprint(api.ID), s.stage)

		// TODO Implement deletion of items from cache
		err := cache.GetCache().Set(key, api)
		if err != nil {
			log.WithFields(fields).Errorf("unable to set cache", err)
		}

		// TODO Handle purging of cache
		err = cache.GetCache().SetWithSecondaryKey(key, strconv.FormatInt(api.ID, 10), api)
		if err != nil {
			log.WithFields(fields).Errorf("unable to set cache with secondary key", err)
		}

		serviceDetail, err := s.getServiceDetail(asset, &api)
		if err != nil {
			log.WithFields(fields).Errorf("error getting the service details: %s", err.Error())
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
	log := logrus.WithFields(logrus.Fields{
		"assetName":       asset.AssetID,
		"assetID":         asset.ID,
		"apiID":           api.ID,
		"apiAssetVersion": api.AssetVersion,
	})
	if !shouldDiscoverAPI(api.EndpointURI, s.discoveryTags, s.discoveryIgnoreTags, api.Tags) {
		log.WithFields(logrus.Fields{
			"endpoint": api.EndpointURI,
		}).Debug("skipping discovery for api")
		return nil, nil
	}

	// Get the policies associated with the API
	policies, err := s.client.GetPolicies(api.ID)
	if err != nil {
		return nil, err
	}
	authPolicy, configuration, isSlaBased := getAuthPolicy(policies)

	apiID := strconv.FormatInt(api.ID, 10)

	subSchName := s.subscriptionManager.GetSubscriptionSchemaName(subscription.PolicyDetail{
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
		schema, err1 := s.createSubscriptionSchemaForSLATier(apiID, tiers)
		if err1 != nil {
			return nil, err1
		}

		subSchName = schema.GetSubscriptionName()
	}

	//TODO can be refactored to not use authpolicy in checksum and use policy
	isAlreadyPublished, checksum := isPublished(api, authPolicy)
	if isAlreadyPublished {
		// If true, then the api is published and there were no changes detected
		log.WithFields(logrus.Fields{
			"policy": authPolicy,
			"msg":    "api is already published",
		})
		return nil, nil
	}
	log.WithField("policy", authPolicy).Debugf("change detected in published asset")

	// Potentially discoverable API, gather the details
	exchangeAsset, err := s.client.GetExchangeAsset(api.GroupID, api.AssetID, api.AssetVersion)
	if err != nil {
		return nil, err
	}

	exchFile := getExchangeAssetSpecFile(exchangeAsset.Files)

	if exchFile == nil {
		// SDK needs a spec
		log.Debugf("no supported specification file found")
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

	return &ServiceDetail{
		APIName:          api.AssetID,
		APISpec:          modifiedSpec,
		AuthPolicy:       authPolicy,
		ID:               fmt.Sprint(api.ID),
		Image:            icon,
		ImageContentType: iconContentType,
		ResourceType:     specType,
		ServiceAttributes: map[string]string{
			"checksum": checksum,
			"assetID":  fmt.Sprint(asset.ID),
		},
		Stage:            s.stage,
		Tags:             api.Tags,
		Title:            asset.ExchangeAssetName,
		Version:          api.AssetVersion,
		SubscriptionName: subSchName,
		Status:           apic.PublishedStatus,
	}, nil
}

func (s *serviceHandler) createSubscriptionSchemaForSLATier(
	apiID string,
	tiers *anypoint.Tiers,
) (apic.SubscriptionSchema, error) {
	schema := apic.NewSubscriptionSchema(apiID)

	var names []string

	for _, tier := range tiers.Tiers {
		t := fmt.Sprintf("%v-%s", tier.Id, tier.Name)
		names = append(names, t)
	}

	schema.AddProperty(anypoint.AppName, "string", "Name of the new app", "", true, nil)
	schema.AddProperty(anypoint.Description, "string", "", "", false, nil)
	schema.AddProperty(anypoint.TierLabel, "string", "", "", true, names)

	constructor := func(apic anypoint.Client) subscription.Handler {
		return slatier.New(apiID, apic, schema)
	}
	s.subscriptionManager.RegisterNewSchema(constructor, s.client)

	if err := agent.GetCentralClient().RegisterSubscriptionSchema(schema, true); err != nil {
		return nil, fmt.Errorf("failed to register subscription schema %s: %w", schema.GetSubscriptionName(), err)
	}
	log.Infof("Schema registered: %s", schema.GetSubscriptionName())

	return schema, nil
}

// shouldDiscoverAPI determines if the API should be pushed to Central or not
func shouldDiscoverAPI(endpoint string, discoveryTags, ignoreTags, apiTags []string) bool {
	if endpoint == "" {
		// If the API has no exposed endpoint we're not going to discover it.
		logrus.Debugf("consumer endpoint not found")
		return false
	}

	if doesAPIContainAnyMatchingTag(ignoreTags, apiTags) {
		return false // ignore
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
		oas2Swagger, err = apic.ParseOAS2(specContent)
		if err != nil {
			return nil, err
		}
		err := apic.SetHostDetails(oas2Swagger, endpointURI)
		if err != nil {
			logrus.Debug("failed to update the spec with the given endpoint: %s", endpointURI)
		}
		specContent, err = setOAS2policies(oas2Swagger, authPolicy, configuration)

	case apic.Oas3:
		oas3Swagger, err = apic.ParseOAS3(specContent)
		if err != nil {
			return nil, err
		}
		apic.SetServers([]string{endpointURI}, oas3Swagger)
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
		if policy.Template.AssetId == anypoint.ClientID {
			return apic.Apikey, policy.Configuration, false
		}

		if strings.Contains(policy.Template.AssetId, anypoint.SlaAuth) {
			return apic.Apikey, policy.Configuration, true
		}

		if policy.Template.AssetId == anypoint.ExternalOauth {
			return apic.Oauth, policy.Configuration, false
		}
	}

	return apic.Passthrough, nil, false
}

func setWSDLEndpoint(_ string, specContent []byte) ([]byte, error) {
	// TODO
	return specContent, nil
}

// checksum generates a checksum for the api for change detection
func checksum(val interface{}, authPolicy string) string {
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
	// return sc, errors.New(anypoint.ErrAuthNotSupported)
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

	// return sc, errors.New(anypoint.ErrAuthNotSupported)
	return json.Marshal(spec)
}

// isPublished checks if an api is published with the latest changes. Returns true if it is, and false if it is not.
func isPublished(api *anypoint.API, authPolicy string) (bool, string) {
	// Change detection (asset + policies)
	checksum := checksum(api, authPolicy)
	if agent.IsAPIPublishedByID(fmt.Sprint(api.ID)) {
		publishedChecksum := agent.GetAttributeOnPublishedAPIByID(fmt.Sprint(api.ID), "checksum")
		if checksum == publishedChecksum {
			// the api is already published with the latest changes
			return true, checksum
		}
	}
	// the api is not published
	return false, checksum
}
