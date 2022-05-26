package discovery

import (
	coreAgent "github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/apic"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agents-mulesoft/pkg/config"
	"github.com/sirupsen/logrus"
)

// publisher implements the Repeater interface. Waits for for items on a channel and publishes them to central
type publisher struct {
	apiChan     chan *ServiceDetail
	stopPublish chan bool
	publishAPI  coreAgent.PublishAPIFunc
}

func (p *publisher) Stop() {
	p.stopPublish <- true
}

func (p *publisher) OnConfigChange(_ *config.MulesoftConfig) {
	// noop
}

// Loop publishes apis to central.
func (p *publisher) Loop() {
	for {
		select {
		case serviceDetail := <-p.apiChan:
			go p.publish(serviceDetail)
		case <-p.stopPublish:
			logrus.Debug("stopping publish listener")
			return
		}
	}
}

// publish Publishes the API to Amplify Central.
func (p *publisher) publish(serviceDetail *ServiceDetail) {
	log := logrus.WithFields(logrus.Fields{
		"name":    serviceDetail.APIName,
		"id":      serviceDetail.ID,
		"stage":   serviceDetail.Stage,
		"version": serviceDetail.Version,
	})
	log.Infof("Publishing to Amplify Central")

	serviceBody, err := BuildServiceBody(serviceDetail)
	if err != nil {
		log.WithError(err).Error("error building service body")
		return
	}
	err = p.publishAPI(serviceBody)
	if err != nil {
		log.WithError(err).Error("error publishing to Amplify Central")
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
		SetServiceAgentDetails(util.MapStringStringToMapStringInterface(service.AgentDetails)).
		SetServiceAttribute(service.ServiceAttributes).
		SetStage(service.Stage).
		SetState(service.State).
		SetStatus(service.Status).
		SetSubscriptionName(service.SubscriptionName).
		SetTags(tags).
		SetTitle(service.Title).
		SetURL(service.URL).
		SetVersion(service.Version).
		SetAccessRequestDefinitionName(service.AccessRequestDefinition, false).
		Build()
}
