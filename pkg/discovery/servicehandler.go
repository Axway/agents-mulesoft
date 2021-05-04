package discovery

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	openapi2 "github.com/go-openapi/spec"

	"github.com/Axway/agent-sdk/pkg/agent"

	"github.com/Axway/agent-sdk/pkg/apic"
	"sigs.k8s.io/yaml"

	"github.com/sirupsen/logrus"

	"github.com/Axway/agent-sdk/pkg/cache"
	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/Axway/agents-mulesoft/pkg/anypoint"
	"github.com/Axway/agents-mulesoft/pkg/config"
)

// ServiceHandler converts a mulesoft asset to an array of ServiceDetails
type ServiceHandler interface {
	ToServiceDetails(asset *anypoint.Asset) []*ServiceDetail
	OnConfigChange(cfg *config.MulesoftConfig)
}

type serviceHandler struct {
	assetCache          cache.Cache
	freshCache          cache.Cache
	stage               string
	discoveryTags       []string
	discoveryIgnoreTags []string
	client              anypoint.Client
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
	for _, api := range asset.APIs {
		// Cache - update the existing to ensure it contains anything new, but create fresh cache
		// to ensure deletions are detected.
		key := formatCacheKey(fmt.Sprint(api.ID), s.stage)
		s.assetCache.Set(key, api)
		s.freshCache.Set(key, api) // Need to handle if the API exists but becomes undiscoverable

		serviceDetail, err := s.getServiceDetail(asset, &api)
		if err != nil {
			log.Errorf("Error gathering information for '%s(%d)': %s", asset.Name, asset.ID, err.Error())
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
	if !shouldDiscoverAPI(api.EndpointURI, s.discoveryTags, s.discoveryIgnoreTags, api.Tags) {
		return nil, nil
	}

	// Get the policies associated with the API
	policies, err := s.client.GetPolicies(api.ID)
	if err != nil {
		return nil, err
	}
	authPolicy := getAuthPolicy(policies)

	isAlreadyPublished, checksum := isPublished(api, authPolicy)
	if isAlreadyPublished {
		// If true, then the api is published and there were no changes detected
		return nil, nil
	}
	log.Debugf("Change detected in published asset %s(%d)", asset.AssetID, api.ID)

	// Potentially discoverable API, gather the details
	log.Infof("Gathering details for %s(%d)", asset.AssetID, api.ID)
	exchangeAsset, err := s.client.GetExchangeAsset(api.GroupID, api.AssetID, api.AssetVersion)
	if err != nil {
		return nil, err
	}

	exchFile := getExchangeAssetSpecFile(exchangeAsset.Files)

	if exchFile == nil {
		// SDK needs a spec
		log.Debugf("No supported specification file found for asset '%s (%s)'", api.AssetID, api.AssetVersion)
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
		return nil, fmt.Errorf("unknown spec type for '%s (%s)'", api.AssetID, api.AssetVersion)
	}
	modifiedSpec, err := updateSpec(specType, api.EndpointURI, authPolicy, specContent)
	if err != nil {
		return nil, err
	}

	icon, iconContentType, err := s.client.GetExchangeAssetIcon(exchangeAsset.Icon)
	if err != nil {
		return nil, err
	}

	return &ServiceDetail{
		APIName:           api.AssetID,
		APISpec:           modifiedSpec,
		AuthPolicy:        authPolicy,
		ID:                fmt.Sprint(api.ID),
		Image:             icon,
		ImageContentType:  iconContentType,
		ResourceType:      specType,
		ServiceAttributes: map[string]string{"checksum": checksum},
		Stage:             s.stage,
		Tags:              api.Tags,
		Title:             asset.ExchangeAssetName,
		Version:           api.AssetVersion,
	}, nil
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
func updateSpec(specType, endpointURI, authPolicy string, specContent []byte) ([]byte, error) {
	var err error
	// Make a best effort to update the endpoints - required because the SDK is parsing from spec and not setting the
	// endpoint information independently.
	switch specType {
	case apic.Oas2:
		specContent, err = setOAS2Endpoint(endpointURI, specContent)
		specContent, err = setOAS2policies(specContent, authPolicy)
	case apic.Oas3:
		specContent, err = setOAS3Endpoint(endpointURI, specContent)
		specContent, err = setOAS3policies(specContent, authPolicy)
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
func getAuthPolicy(policies anypoint.Policies) string {
	for _, policy := range policies.Policies {
		if policy.Template.AssetId == anypoint.ClientID || strings.Contains(policy.Template.AssetId,anypoint.SlaAuth){
			return apic.Apikey
		}

		if policy.Template.AssetId == anypoint.ExternalOauth {
			return apic.Oauth
		}
	}

	return apic.Passthrough
}

func setOAS2Endpoint(endpointURL string, specContent []byte) ([]byte, error) {
	endpoint, err := url.Parse(endpointURL)
	if err != nil {
		return specContent, err
	}
	spec := make(map[string]interface{})
	err = json.Unmarshal(specContent, &spec)
	if err != nil {
		return specContent, err
	}
	spec["host"] = endpoint.Host
	if endpoint.Path == "" {
		log.Debug("Empty base path, manually changing to /")
		spec["basePath"] = "/"
	} else {
		spec["basePath"] = endpoint.Path
	}
	spec["schemes"] = []string{endpoint.Scheme}
	return json.Marshal(spec)
}

func setOAS3Endpoint(url string, specContent []byte) ([]byte, error) {
	spec := make(map[string]interface{})

	err := json.Unmarshal(specContent, &spec)
	if err != nil {
		return specContent, err
	}

	spec["servers"] = []interface{}{
		map[string]string{"url": url},
	}

	return json.Marshal(spec)
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

// Helper function to remove existing back-end security policies
func removeOASpolicies(specContent []byte, oasVersion string) ([]byte, error) {
	spec := make(map[string]interface{})
	err := json.Unmarshal(specContent, &spec)
	if err != nil {
		return specContent, err
	}
	// Deleting any pre-existing security definitions
	if oasVersion == "v2" {
		if spec["securityDefinitions"] != nil {
			delete(spec, "securityDefinitions")
		}
	}
	if oasVersion == "v3" {

		oas3Spec := openapi3.Swagger{}
		err := json.Unmarshal(specContent, &oas3Spec)
		if err != nil {
			return specContent, err
		}

		// reset to empty
		oas3Spec.Components.SecuritySchemes = nil
		return json.Marshal(oas3Spec)

	}

	return json.Marshal(spec)
}

// TODO improve if fields are not complete, introduce per method swagger definitions, set up logic for multiple auth policies, use policy configurations
func setOAS2policies(sc []byte, authPolicy string) ([]byte, error) {
	// Removing pre-existing auth security policies
	sc, err := removeOASpolicies(sc, "v2")
	if err != nil {
		return sc, err
	}

	oas2Spec := openapi2.Swagger{}
	err = json.Unmarshal(sc, &oas2Spec)
	if err != nil {
		return nil, err
	}

	switch authPolicy {
	case apic.Apikey:
		ssp := openapi2.SecuritySchemeProps{
			Type:        "apiKey",
			Name:        "Authorization",
			In:          "header",
			Description: "Provided as: client_id:<INSERT_VALID_CLIENTID_HERE> client_secret:<INSERT_VALID_SECRET_HERE>",
		}
		ss := openapi2.SecurityScheme{
			VendorExtensible:    openapi2.VendorExtensible{},
			SecuritySchemeProps: ssp,
		}
		sd := openapi2.SecurityDefinitions{
			"client-id-enforcement": &ss,
		}

		oas2Spec.SwaggerProps.SecurityDefinitions = sd

	case apic.Oauth:
		ss := openapi2.OAuth2Implicit("dummy.io")
		sd := openapi2.SecurityDefinitions{
			"oauth": ss,
		}

		oas2Spec.SwaggerProps.SecurityDefinitions = sd
	}

	return json.Marshal(oas2Spec)
}

func setOAS3policies(sc []byte, authPolicy string) ([]byte, error) {
	//Removing pre-existing auth security policies
	sc, err := removeOASpolicies(sc, "v3")
	if err != nil {
		return sc, err
	}

	oas3Spec := openapi3.Swagger{}
	err = json.Unmarshal(sc, &oas3Spec)
	if err != nil {
		return sc, err
	}

	switch authPolicy {
	case apic.Apikey:
		ss := openapi3.SecurityScheme{
			Type:        "apiKey",
			Name:        "Authorization",
			In:          "header",
			Description: "Provided as: client_id:<INSERT_VALID_CLIENTID_HERE> client_secret:<INSERT_VALID_SECRET_HERE>",
		}

		ssr := openapi3.SecuritySchemeRef{
			Value: &ss,
		}

		oas3Spec.Components.SecuritySchemes = openapi3.SecuritySchemes{"client-id-enforcement": &ssr}

	case apic.Oauth:
		i := openapi3.OAuthFlow{
			ExtensionProps:   openapi3.ExtensionProps{},
			AuthorizationURL: "dummy.io",
			Scopes:           make(map[string]string),
		}
		oAuthFlow := openapi3.OAuthFlows{
			ExtensionProps: openapi3.ExtensionProps{},
			Implicit:       &i,
		}
		ss := openapi3.SecurityScheme{
			ExtensionProps: openapi3.ExtensionProps{},
			Type:           "oauth2",
			Description:    "This API uses OAuth 2 with the implicit grant flow",
			Flows:          &oAuthFlow,
		}
		ssr := openapi3.SecuritySchemeRef{
			Value: &ss,
		}

		oas3Spec.Components.SecuritySchemes = openapi3.SecuritySchemes{"Oauth": &ssr}
	}

	return json.Marshal(oas3Spec)
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
