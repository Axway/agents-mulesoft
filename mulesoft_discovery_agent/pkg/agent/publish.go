package agent

import (
	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/apic"
	log "github.com/Axway/agent-sdk/pkg/util/log"
)

func (a *Agent) publishLoop() {
	for {
		select {
		case serviceDetail := <-a.apiChan:
			err := a.publish(serviceDetail)
			if err != nil {
				log.Errorf("Error publishing API \"%s:(%s)\":%s", serviceDetail.APIName, serviceDetail.ID, err.Error())
			}
		case <-a.stopPublish:
			return
		}
	}
}

func (a *Agent) publish(serviceDetail *ServiceDetail) error {
	log.Infof("Publishing API \"%s (%s)\" to AMPLIFY Central", serviceDetail.APIName, serviceDetail.ID)

	serviceBody, err := a.buildServiceBody(serviceDetail)
	if err != nil {
		log.Errorf("Error publishing API \"%s (%s)\" to AMPLIFY Central: %s", serviceDetail.APIName, serviceDetail.ID, err.Error())
		return err
	}
	err = agent.PublishAPI(serviceBody)
	if err != nil {
		log.Errorf("Error publishing API \"%s (%s)\" to AMPLIFY Central: %s", serviceDetail.APIName, serviceDetail.ID, err.Error())
		return err
	}
	log.Infof("Published API \"%s (%s)\" to AMPLIFY Central", serviceDetail.APIName, serviceDetail.ID)
	return err
}

// buildServiceBody - creates the service definition
func (a *Agent) buildServiceBody(service *ServiceDetail) (apic.ServiceBody, error) {
	return apic.NewServiceBodyBuilder().
		SetID(service.ID).
		SetTitle(service.Title).
		SetAPIName(service.APIName).
		SetURL(service.URL).
		SetStage(service.Stage).
		SetDescription(service.Description).
		SetVersion(service.Instances[0].Version). // TODO ALL VERSIONS
		SetAuthPolicy(service.AuthPolicy).
		SetAPISpec(service.APISpec). // TODO UPDATE FOR INSTANCES
		SetDocumentation(service.Documentation).
		SetTags(service.Tags).
		SetImage(service.Image).
		SetImageContentType(service.ImageContentType).
		SetResourceType(service.ResourceType).
		SetSubscriptionName(service.SubscriptionName).
		SetAPIUpdateSeverity(service.APIUpdateSeverity).
		SetState(service.State).
		SetStatus(service.Status).
		SetServiceAttribute(service.ServiceAttributes).
		Build()
}
