package discovery

import (
	"fmt"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/apic"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

// publishLoop Publish event loop.
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

// publish Publishes the API to Amplify Central.
func (a *Agent) publish(serviceDetail *ServiceDetail) error {
	log.Infof("Publishing API \"%s (%s)\" to Amplify Central", serviceDetail.APIName, serviceDetail.ID)

	serviceBody, err := a.buildServiceBody(serviceDetail)
	if err != nil {
		log.Errorf("Error building service body for API \"%s (%s)\": %s", serviceDetail.APIName, serviceDetail.ID, err.Error())
		return err
	}
	err = agent.PublishAPI(serviceBody)
	if err != nil {
		log.Errorf("Error publishing API \"%s (%s)\" to Amplify Central: %s", serviceDetail.APIName, serviceDetail.ID, err.Error())
		return err
	}
	log.Infof("Published API \"%s (%s)\" to Amplify Central", serviceDetail.APIName, serviceDetail.ID)
	return err
}

// buildServiceBody - creates the service definition
func (a *Agent) buildServiceBody(service *ServiceDetail) (apic.ServiceBody, error) {
	tags := map[string]interface{}{}
	if service.Tags != nil {
		for _, tag := range service.Tags {
			tags[tag] = true
		}
	}
	return apic.NewServiceBodyBuilder().
		SetID(service.ID).
		SetTitle(fmt.Sprintf("%s (%s)", service.Title, service.Version)).
		SetAPIName(service.APIName).
		SetURL(service.URL).
		SetStage(service.Stage).
		SetDescription(service.Description).
		SetVersion(service.Version).
		SetAuthPolicy(service.AuthPolicy).
		SetAPISpec(service.APISpec).
		SetDocumentation(service.Documentation).
		SetTags(tags).
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
