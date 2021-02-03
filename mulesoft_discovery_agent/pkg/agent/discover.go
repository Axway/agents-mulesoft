package agent

import (
	"encoding/json"
	"fmt"
	"sort"

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
				log.Errorf("Error gathering information for \"%s(%d)\": %s", asset.Name, asset.ID, err.Error())
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

func (a *Agent) getExternalAPI(asset *anypoint.Asset) (*ServiceDetail, error) {
	exchangeAsset, err := a.anypointClient.GetExchangeAsset(asset)
	if err != nil {
		return nil, err
	}

	exchangeFile, err := a.getExchangeAssetSpecFile(exchangeAsset)
	if err != nil {
		return nil, err
	}

	if exchangeFile == nil {
		// SDK needs a spec
		log.Debugf("No supported specification file found for asset %s (%d)", asset.Name, asset.ID)
		return nil, nil
	}

	specContent, err := a.anypointClient.GetExchangeFileContent(exchangeFile)
	if err != nil {
		return nil, err
	}

	specType, err := a.getSpecType(asset, exchangeFile, specContent)
	if err != nil {
		return nil, err
	}

	icon, iconContentType, err := a.anypointClient.GetExchangeAssetIcon(exchangeAsset)
	if err != nil {
		return nil, err
	}

	return &ServiceDetail{
		ID:               fmt.Sprint(asset.ID),
		Title:            asset.ExchangeAssetName,
		APIName:          asset.AssetID,
		Stage:            a.stage, // Or perhaps it should be the asset api stage
		APISpec:          specContent,
		ResourceType:     specType,
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

// getSpecType determines the correct resource type for the asset.
func (a *Agent) getSpecType(asset *anypoint.Asset, file *anypoint.ExchangeFile, specContent []byte) (string, error) {
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

	return "", fmt.Errorf("Unknown spec type for \"%s(%d)\"", asset.Name, asset.ID)
}
