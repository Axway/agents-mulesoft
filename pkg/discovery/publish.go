package discovery

import (
	"github.com/Axway/agent-sdk/pkg/apic"
	"github.com/Axway/agents-mulesoft/pkg/config"
	"github.com/sirupsen/logrus"
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
			logrus.Debug("stopping publish listener")
			return
		}
	}
}

// publish Publishes the API to Amplify Central.
func (p *publisher) publish(serviceDetail *ServiceDetail) {
	log := logrus.WithFields(logrus.Fields{
		"name":  serviceDetail.APIName,
		"id":    serviceDetail.ID,
		"stage": serviceDetail.Stage,
	})
	log.Infof("Publishing to Amplify Central")

	serviceBody, err := BuildServiceBody(serviceDetail)
	if err != nil {
		log.WithError(err).Errorf("error building service body")
		return
	}
	err = p.publishAPI(serviceBody)
	if err != nil {
		log.WithError(err).Errorf("error publishing to Amplify Central: %s")
		return
	}
	log.Infof("Published API to Amplify Central")
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
