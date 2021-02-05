package agent

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/apic"
	"github.com/Axway/agent-sdk/pkg/cache"
	log "github.com/Axway/agent-sdk/pkg/util/log"
	anypoint "github.com/Axway/agents-mulesoft/mulesoft_discovery_agent/pkg/anypoint"
	"sigs.k8s.io/yaml"
)

// discoveryLoop Discovery event loop.
func (a *Agent) discoveryLoop() {
	go func() {
		// Instant fist "tick"
		a.discoverAPIs()
		// Loop
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
		key := a.formatCacheKey(fmt.Sprint(api.ID), a.stage)
		a.assetCache.Set(key, api)
		freshAssetCache.Set(key, api)

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

	// Get the policies associated with the API
	policies, err := a.anypointClient.GetPolicies(api)
	if err != nil {
		return nil, err
	}
	authPolicy := a.getAuthPolicy(policies)

	// Change detection (asset + policies)
	checksum := a.checksum(api, authPolicy)
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

	exchangeFile, err := a.getExchangeAssetSpecFile(exchangeAsset) // SDK only supports OAS & WSDL
	if err != nil {
		return nil, err
	}

	if exchangeFile == nil {
		// SDK needs a spec
		log.Debugf("No supported specification file found for asset %s (%s)", api.AssetID, api.AssetVersion)
		return nil, nil
	}

	specContent, err := a.anypointClient.GetExchangeFileContent(exchangeFile)
	if err != nil {
		return nil, err
	}
	specContent = a.specYAMLToJSON(specContent) // SDK does not support YAML specifications

	specType, err := a.getSpecType(exchangeFile, specContent)
	if err != nil {
		return nil, err
	}
	if specType == "" {
		return nil, fmt.Errorf("Unknown spec type for \"%s(%s)\"", api.AssetID, api.AssetVersion)
	}

	icon, iconContentType, err := a.anypointClient.GetExchangeAssetIcon(exchangeAsset)
	if err != nil {
		return nil, err
	}

	return &ServiceDetail{
		ID:                fmt.Sprint(api.ID),
		Title:             asset.ExchangeAssetName,
		Version:           api.AssetVersion,
		APIName:           api.AssetID,
		Stage:             a.stage,
		APISpec:           specContent,
		ResourceType:      specType,
		AuthPolicy:        authPolicy,
		Image:             icon,
		ImageContentType:  iconContentType,
		Tags:              api.Tags,
		ServiceAttributes: map[string]string{"checksum": checksum},
	}, nil
}

// shouldDiscoverAPI - callback used determine if the API should be pushed to Central or not
func (a *Agent) shouldDiscoverAPI(api *anypoint.API) bool {
	if a.doesAPIContainAnyMatchingTag(a.discoveryIgnoreTags, api) {
		return false // ignore
	}

	if len(a.discoveryTags) > 0 {
		if !a.doesAPIContainAnyMatchingTag(a.discoveryTags, api) {
			return false // ignore
		}
	}
	return true
}

// doesAPIContainAnyMatchingTag checks if the API has any of the tags
func (a *Agent) doesAPIContainAnyMatchingTag(tags []string, api *anypoint.API) bool {
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

// getExchangeAssetSpecFile gets the file entry for the Assets spec.
func (a *Agent) getExchangeAssetSpecFile(asset *anypoint.ExchangeAsset) (*anypoint.ExchangeFile, error) {
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
func (a *Agent) specYAMLToJSON(specContent []byte) []byte {
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
func (a *Agent) getSpecType(file *anypoint.ExchangeFile, specContent []byte) (string, error) {
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
func (a *Agent) getAuthPolicy(policies []anypoint.Policy) string {
	if policies == nil || len(policies) == 0 {
		return apic.Passthrough
	}

	for _, policy := range policies {
		if policy.PolicyTemplateID == "client-id-enforcement" {
			return apic.Apikey
		}
	}

	return apic.Passthrough
}

// checksum generates a checksum for the api for change detection
func (a *Agent) checksum(val interface{}, authPolicy string) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%v%s", val, authPolicy)))
	return fmt.Sprintf("%x", sum)
}
