package discovery

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/getkin/kin-openapi/openapi3"
	openapi2 "github.com/go-openapi/spec"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/apic"
	"github.com/Axway/agent-sdk/pkg/cache"
	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/Axway/agents-mulesoft/pkg/anypoint"
	"sigs.k8s.io/yaml"
)

type APIDiscovery interface {
	DiscoveryLoop()
}

// discoveryLoop Discovery event loop.
func (a *Agent) discoveryLoop() {
	go func() {
		// Instant fist "tick"
		a.discoverAPIs()
		logrus.Info("Starting poller for Mulesoft APIs")
		ticker := time.NewTicker(a.pollInterval)
		for {
			select {
			case <-ticker.C:
				a.discoverAPIs()
				break
			case <-a.stopDiscovery:
				log.Debug("stopping discovery loop")
				ticker.Stop()
				break
			}
		}
	}()
}

// discoverAPIs Finds the APIs that are publishable.
func (a *Agent) discoverAPIs() {
	offset := 0
	pageSize := a.discoveryPageSize

	// Replacing asset cache rather than updating it
	freshAssetCache := cache.New()

	for {
		page := &anypoint.Page{Offset: offset, PageSize: pageSize}

		assets, err := a.anypointClient.ListAssets(page)
		if err != nil {
			log.Error(err)
		}

		for _, asset := range assets {

			svcDetails := a.getServiceDetails(&asset, freshAssetCache)
			if svcDetails != nil {
				for _, svc := range svcDetails {
					a.apiChan <- svc
				}
			}
		}

		if len(assets) != pageSize {
			break
		} else {
			offset += pageSize
		}
	}

	// Replace the cache
	a.assetCache = freshAssetCache
}

// getServiceDetails gathers the ServiceDetail for a single Mulesoft Asset. Each Asset has multiple versions and
// so can resolve to multiple ServiceDetails.
func (a *Agent) getServiceDetails(asset *anypoint.Asset, freshAssetCache cache.Cache) []*ServiceDetail {
	serviceDetails := []*ServiceDetail{}
	for _, api := range asset.APIs {
		// Cache - update the existing to ensure it contains anything new, but create fresh cache
		// to ensure deletions are detected.
		key := formatCacheKey(fmt.Sprint(api.ID), a.stage)
		a.assetCache.Set(key, api)
		freshAssetCache.Set(key, api) // Need to handle if the API exists but becomes undiscoverable

		serviceDetail, err := a.getServiceDetail(asset, &api)
		if err != nil {
			log.Errorf("Error gathering information for \"%s(%d)\": %s", asset.Name, asset.ID, err.Error())
			continue
		}
		if serviceDetail != nil {
			serviceDetails = append(serviceDetails, serviceDetail)
		}
	}
	return serviceDetails
}

// getServiceDetail gets the ServiceDetail for the API asset.
func (a *Agent) getServiceDetail(asset *anypoint.Asset, api *anypoint.API) (*ServiceDetail, error) {
	// Filtering
	if !a.shouldDiscoverAPI(api) {
		// Skip
		return nil, nil
	}

	if api.EndpointURI == "" {
		// If the API has no exposed endpoint we're not going to discover it.
		logrus.Debugf("consumer endpoint not found for %s", api.AssetID)
		return nil, nil
	}

	// Get the policies associated with the API
	policies, err := a.anypointClient.GetPolicies(api)
	if err != nil {
		return nil, err
	}
	authPolicy := a.getAuthPolicy(policies)

	// Change detection (asset + policies)
	checksum := checksum(api, authPolicy)
	if agent.IsAPIPublished(fmt.Sprint(api.ID)) {
		publishedChecksum := agent.GetAttributeOnPublishedAPI(fmt.Sprint(api.ID), "checksum")
		if checksum == publishedChecksum {
			return nil, nil
		}
		log.Debugf("Change detected in published asset %s(%d)", asset.AssetID, api.ID)
	}

	// Potentially discoverable API, gather the details
	log.Infof("Gathering details for %s(%d)", asset.AssetID, api.ID)

	exchangeAsset, err := a.anypointClient.GetExchangeAsset(api)
	if err != nil {
		return nil, err
	}

	exchangeFile, err := getExchangeAssetSpecFile(exchangeAsset) // SDK only supports OAS & WSDL
	if err != nil {
		return nil, err
	}

	if exchangeFile == nil {
		// SDK needs a spec
		log.Debugf("No supported specification file found for asset %s (%s)", api.AssetID, api.AssetVersion)
		return nil, nil
	}

	specContent, specType, err := a.getSpecFromExchangeFile(api, exchangeFile, authPolicy)
	if err != nil {
		return nil, err
	}

	icon, iconContentType, err := a.anypointClient.GetExchangeAssetIcon(exchangeAsset)
	if err != nil {
		return nil, err
	}

	return &ServiceDetail{
		APIName:           api.AssetID,
		APISpec:           specContent,
		AuthPolicy:        authPolicy,
		ID:                fmt.Sprint(api.ID),
		Image:             icon,
		ImageContentType:  iconContentType,
		ResourceType:      specType,
		ServiceAttributes: map[string]string{"checksum": checksum},
		Stage:             a.stage,
		Tags:              api.Tags,
		Title:             asset.ExchangeAssetName,
		Version:           api.AssetVersion,
	}, nil
}

// shouldDiscoverAPI - callback used determine if the API should be pushed to Central or not
func (a *Agent) shouldDiscoverAPI(api *anypoint.API) bool {
	if doesAPIContainAnyMatchingTag(a.discoveryIgnoreTags, api) {
		return false // ignore
	}

	if len(a.discoveryTags) > 0 {
		if !doesAPIContainAnyMatchingTag(a.discoveryTags, api) {
			return false // ignore
		}
	}
	return true
}

// getSpecFromExchangeFile gets the spec content and injects the api endpoint.
func (a *Agent) getSpecFromExchangeFile(api *anypoint.API, exchangeFile *anypoint.ExchangeFile, authPolicy string) ([]byte, string, error) {
	specContent, err := a.anypointClient.GetExchangeFileContent(exchangeFile)
	if err != nil {
		return nil, "", err
	}

	specContent = specYAMLToJSON(specContent) // SDK does not support YAML specifications
	specType, err := getSpecType(exchangeFile, specContent)
	if err != nil {
		return nil, "", err
	}
	if specType == "" {
		return nil, specType, fmt.Errorf("Unknown spec type for \"%s(%s)\"", api.AssetID, api.AssetVersion)
	}

	// Make a best effort to update the endpoints - required because the SDK is parsing from spec and not setting the
	// endpoint information independently.
	switch specType {
	case apic.Oas2:
		specContent, err = setOAS2Endpoint(api.EndpointURI, specContent)
		specContent, err = a.setOAS2policies(specContent, authPolicy)
	case apic.Oas3:
		specContent, err = setOAS3Endpoint(api.EndpointURI, specContent)
		specContent, err = a.setOAS3policies(specContent, authPolicy)
	case apic.Wsdl:
		specContent, err = setWSDLEndpoint(api.EndpointURI, specContent)
	}

	return specContent, specType, err
}

// getExchangeAssetSpecFile gets the file entry for the Assets spec.
func getExchangeAssetSpecFile(asset *anypoint.ExchangeAsset) (*anypoint.ExchangeFile, error) {
	if asset.Files == nil || len(asset.Files) == 0 {
		return nil, nil
	}

	sort.Sort(BySpecType(asset.Files))
	if asset.Files[0].Classifier != "oas" &&
		asset.Files[0].Classifier != "fat-oas" &&
		asset.Files[0].Classifier != "wsdl" {
		// Unsupported spec type
		return nil, nil
	}
	return &asset.Files[0], nil
}

// specYAMLToJSON - if the spec is yaml convert it to json, SDK doesn't handle yaml.
func specYAMLToJSON(specContent []byte) []byte {
	specMap := make(map[string]interface{})
	err := json.Unmarshal(specContent, &specMap)
	if err == nil {
		return specContent
	}

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
	if file.Classifier == "wsdl" {
		return apic.Wsdl, nil
	} else if specContent != nil {
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
func (a *Agent) getAuthPolicy(policies anypoint.Policies) string {

	if len(policies.Policies) == 0 {
		return apic.Passthrough
	}

	for _, policy := range policies.Policies {
		if policy.Template.AssetId == "client-id-enforcement" {
			return apic.Apikey
		}

		if policy.Template.AssetId == "external-oauth2-access-token-enforcement" {
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
func doesAPIContainAnyMatchingTag(tags []string, api *anypoint.API) bool {
	for _, apiTag := range api.Tags {
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
func (a *Agent) removeOASpolicies(specContent []byte, oasVersion string) ([]byte, error) {
	spec := make(map[string]interface{})
	err := json.Unmarshal(specContent, &spec)
	if err != nil {
		return specContent, err
	}
	//Deleting any pre-existing security definitions
	if oasVersion == "v2" {
		if spec["securityDefinitions"] != nil {
			delete(spec, "securityDefinitions")
		}
	}
	if oasVersion == "v3" {

		oas3Spec := openapi3.Swagger{}
		json.Unmarshal(specContent, &oas3Spec)
		if err != nil {
			return specContent, err
		}

		//reset to empty
		oas3Spec.Components.SecuritySchemes = nil
		return json.Marshal(oas3Spec)

	}

	return json.Marshal(spec)
}

//TODO improve if fields are not complete, introduce per method swagger definitions, set up logic for multiple auth policies, use policy configurations
func (a *Agent) setOAS2policies(sc []byte, authPolicy string) ([]byte, error) {

	//Removing pre-existing auth security policies
	sc, err := a.removeOASpolicies(sc, "v2")
	if err != nil {
		return sc, err
	}

	oas2Spec := openapi2.Swagger{}
	json.Unmarshal(sc, &oas2Spec)

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

func (a *Agent) setOAS3policies(sc []byte, authPolicy string) ([]byte, error) {

	//Removing pre-existing auth security policies
	sc, err := a.removeOASpolicies(sc, "v3")
	if err != nil {
		return sc, err
	}

	oas3Spec := openapi3.Swagger{}
	err = json.Unmarshal(sc, &oas3Spec)
	if err != nil {
		return sc, err
	}

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

