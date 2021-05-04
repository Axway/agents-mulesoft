package discovery

import (
	"github.com/Axway/agent-sdk/pkg/apic"
	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/Axway/agents-mulesoft/pkg/config"
)

type PublishAPI func(serviceBody apic.ServiceBody) error

// publisher implements the Repeater interface. Waits for for items on a channel and publishes them to central
type publisher struct {
	apiChan     chan *ServiceDetail
	stopPublish chan bool
	publishAPI  PublishAPI
}

func (p *publisher) Stop() {
	p.stopPublish <- true
}

func (p *publisher) OnConfigChange(_ *config.MulesoftConfig) {
	// noop
}

// publisher Publish event loop.
func (p *publisher) Loop() {
	for {
		select {
		case serviceDetail := <-p.apiChan:
			p.publish(serviceDetail)
		case <-p.stopPublish:
			log.Debug("stopping publish listener")
			return
		}
	}
}

// publish Publishes the API to Amplify Central.
func (p *publisher) publish(serviceDetail *ServiceDetail) {
	log.Infof("Publishing API '%s (%s)' to Amplify Central", serviceDetail.APIName, serviceDetail.ID)

	serviceBody, err := BuildServiceBody(serviceDetail)
	if err != nil {
		log.Errorf("Error building service body for API '%s (%s)': %s", serviceDetail.APIName, serviceDetail.ID, err.Error())
		return
	}
	err = p.publishAPI(serviceBody)
	if err != nil {
		log.Errorf("Error publishing API '%s (%s)' to Amplify Central: %s", serviceDetail.APIName, serviceDetail.ID, err.Error())
		return
	}
	log.Infof("Published API '%s (%s)' to Amplify Central", serviceDetail.APIName, serviceDetail.ID)
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
