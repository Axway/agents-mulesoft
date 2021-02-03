package agent

import (
	"encoding/json"

	"github.com/Axway/agent-sdk/pkg/apic"
	log "github.com/Axway/agent-sdk/pkg/util/log"
	anypoint "github.com/Axway/agents-mulesoft/mulesoft_discovery_agent/pkg/anypoint"
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
			externalAPI, err := a.getExternalAPI(&asset)
			if err != nil {
				log.Errorf("Error gathering information for \"%s(%s)\": %s", asset.Name, asset.AssetID, err.Error())
				continue
			}
			if externalAPI != nil {
				a.apiChan <- externalAPI
			}
		}

		if len(assets) != pageSize {
			break
		} else {
			offset += pageSize
		}
	}
}

func (a *Agent) getExternalAPI(asset *anypoint.Asset) (*ExternalAPI, error) {
	assetDetail, err := a.anypointClient.GetAssetDetails(asset)
	if err != nil {
		return nil, err
	}

	specContent, packaging, err := a.anypointClient.GetAssetSpecification(assetDetail)
	if err != nil {
		return nil, err
	}

	icon, err := a.anypointClient.GetAssetIcon(asset)
	if err != nil {
		return nil, err
	}

	catalogType := a.getCatalogType(assetDetail, packaging, specContent)

	return &ExternalAPI{
		Name:        asset.Name,
		ID:          asset.AssetID, // Not sure this will be valid
		URL:         "",            // TODO
		Spec:        specContent,
		Icon:        icon,
		Instances:   assetDetail.Instances,
		Packaging:   packaging,
		CatalogType: catalogType,
	}, nil
}

// getCatalogType determines the correct type for the asset in Unified Catalog.
func (a *Agent) getCatalogType(asset *anypoint.AssetDetails, packaging string, specContent []byte) string {
	switch apiType := asset.AssetType; apiType {
	case "soap-api":
		return apic.Wsdl
	case "rest-api":
		if packaging == "zip" {
			return "raml"
		}
		if specContent != nil {
			jsonMap := make(map[string]interface{})
			err := json.Unmarshal(specContent, &jsonMap)
			if err != nil {
				return apiType
			}
			if _, isSwagger := jsonMap["swagger"]; isSwagger {
				return apic.Oas2
			}
			return apiType
		}
		return apic.Oas3
	default:
		return apiType
	}
}
