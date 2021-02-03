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
	serviceBody, err := a.buildServiceBody(externalAPI)
	if err != nil {
		return err
	}
	err = agent.PublishAPI(serviceBody)
	if err != nil {
		return err
	}
	log.Info("Published API " + serviceBody.APIName + "to AMPLIFY Central")
	return err
}

// buildServiceBody - creates the service definition
func (a *Agent) buildServiceBody(externalAPI *ExternalAPI) (apic.ServiceBody, error) {
	return apic.NewServiceBodyBuilder().
		SetID(externalAPI.ID).
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
