package agent

import (
	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/apic"
	log "github.com/Axway/agent-sdk/pkg/util/log"
)

func (a *Agent) publishLoop() {
	for {
		select {
		case externalAPI := <-a.apiChan:
			if externalAPI.Instances == nil || len(externalAPI.Instances) == 0 {
				// If this Asset has no instances then skip it.
				log.Infof("Skipping API \"%s (%s)\" to AMPLIFY Central - no instances", externalAPI.Name, externalAPI.ID)
				continue
			}

			// SDK is only supporting WSDL, OAS2 & OAS3
			if externalAPI.CatalogType != apic.Oas2 &&
				externalAPI.CatalogType != apic.Oas3 &&
				externalAPI.CatalogType != apic.Wsdl {
				log.Infof("Skipping API \"%s (%s)\" to AMPLIFY Central - unsupported type %s", externalAPI.Name, externalAPI.ID, externalAPI.CatalogType)
				continue
			}

			err := a.publish(externalAPI)
			if err != nil {
				log.Errorf("Error publishing API \"%s:(%s)\":%s", externalAPI.Name, externalAPI.ID, err.Error())
			}
		case <-a.stopPublish:
			return
		}
	}
}

func (a *Agent) publish(externalAPI *ExternalAPI) error {
	log.Infof("Publishing API \"%s (%s)\" to AMPLIFY Central", externalAPI.Name, externalAPI.ID)

	serviceBody, err := a.buildServiceBody(externalAPI)
	if err != nil {
		log.Errorf("Error publishing API \"%s (%s)\" to AMPLIFY Central: %s", externalAPI.Name, externalAPI.ID, err.Error())
		return err
	}
	err = agent.PublishAPI(serviceBody)
	if err != nil {
		log.Errorf("Error publishing API \"%s (%s)\" to AMPLIFY Central: %s", externalAPI.Name, externalAPI.ID, err.Error())
		return err
	}
	log.Infof("Published API \"%s (%s)\" to AMPLIFY Central", externalAPI.Name, externalAPI.ID)
	return err
}

// buildServiceBody - creates the service definition
func (a *Agent) buildServiceBody(externalAPI *ExternalAPI) (apic.ServiceBody, error) {
	return apic.NewServiceBodyBuilder().
		SetID(externalAPI.ID).
		SetAPIName(externalAPI.Name).
		SetTitle(externalAPI.Name).
		SetURL(externalAPI.URL).
		SetDescription(externalAPI.Description).
		SetAPISpec(externalAPI.Spec).
		SetVersion(externalAPI.Version).
		SetAuthPolicy(apic.Passthrough).
		SetDocumentation(externalAPI.Documentation).
		SetResourceType(externalAPI.CatalogType).
		Build()
}
