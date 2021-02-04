package agent

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/Axway/agent-sdk/pkg/apic"
	log "github.com/Axway/agent-sdk/pkg/util/log"
	anypoint "github.com/Axway/agents-mulesoft/mulesoft_discovery_agent/pkg/anypoint"
	"sigs.k8s.io/yaml"
)

func (a *Agent) discoveryLoop() {
	// TODO: Periodic triggering, delta detection, error channel etc.
	a.discoverAPIs()
}

func (a *Agent) discoverAPIs() {
	offset := 0
	pageSize := 20
	for {
		page := &anypoint.Page{Offset: offset, PageSize: pageSize}

		assets, err := a.anypointClient.ListAssets(page)
		if err != nil {
			log.Error(err)
		}

		for _, asset := range assets {
			log.Debugf("Gathering details for %s(%d)", asset.AssetID, asset.ID)
			svcDetails := a.getServiceDetails(&asset)
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
}

func (a *Agent) getServiceDetails(asset *anypoint.Asset) []*ServiceDetail {
	serviceDetails := []*ServiceDetail{}
	for _, api := range asset.APIs {
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
	// Single asset has multiple versions
	policies, err := a.anypointClient.GetPolicies(api)
	if err != nil {
		return nil, err
	}
	authPolicy := a.getAuthPolicy(policies)

	exchangeAsset, err := a.anypointClient.GetExchangeAsset(api)
	if err != nil {
		return nil, err
	}

	exchangeFile, err := a.getExchangeAssetSpecFile(exchangeAsset)
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
	specContent = a.specYAMLToJSON(specContent)

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
		ID:               fmt.Sprint(api.ID),
		Title:            asset.ExchangeAssetName,
		APIName:          api.AssetID,
		Stage:            a.stage, // Or perhaps it should be the asset api stage
		APISpec:          specContent,
		ResourceType:     specType,
		AuthPolicy:       authPolicy,
		Image:            icon,
		ImageContentType: iconContentType,
		Instances:        exchangeAsset.Instances,
		// TODO: everyhing else too
	}, nil
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
