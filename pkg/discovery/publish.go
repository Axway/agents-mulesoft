package discovery

import (
	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/apic"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

type APIPublisher interface {
	PublishLoop()
	Stop()
}

type publisher struct {
	apiChan     chan *ServiceDetail
	stopPublish chan bool
}

func (a *publisher) Stop() {
	a.stopPublish <- true
}

// publishLoop Publish event loop.
func (a *publisher) PublishLoop() {
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
func (a *publisher) publish(serviceDetail *ServiceDetail) error {
	log.Infof("Publishing API \"%s (%s)\" to Amplify Central", serviceDetail.APIName, serviceDetail.ID)

	serviceBody, err := BuildServiceBody(serviceDetail)
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

// BuildServiceBody - creates the service definition
func BuildServiceBody(service *ServiceDetail) (apic.ServiceBody, error) {
	tags := map[string]interface{}{}
	if service.Tags != nil {
		for _, tag := range service.Tags {
			tags[tag] = true
		}
	}
	return apic.NewServiceBodyBuilder().
		SetAPIName(service.APIName).
		SetAPISpec(service.APISpec).
		SetAPIUpdateSeverity(service.APIUpdateSeverity).
		SetAuthPolicy(service.AuthPolicy).
		SetDescription(service.Description).
		SetDocumentation(service.Documentation).
		SetID(service.ID).
		SetImage(service.Image).
		SetImageContentType(service.ImageContentType).
		SetResourceType(service.ResourceType).
		SetServiceAttribute(service.ServiceAttributes).
		SetStage(service.Stage).
		SetState(service.State).
		SetStatus(service.Status).
		SetSubscriptionName(service.SubscriptionName).
		SetTags(tags).
		SetTitle(service.Title).
		SetURL(service.URL).
		SetVersion(service.Version).
		Build()
}
