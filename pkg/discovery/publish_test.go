package discovery

import (
	"testing"

	"github.com/Axway/agents-mulesoft/pkg/config"

	"github.com/Axway/agent-sdk/pkg/apic"

	"github.com/stretchr/testify/assert"
)

var sd = &ServiceDetail{
	APIName:           "Swagger Petstore - OpenAPI 3.0",
	APISpec:           []byte("{\"openapi\":\"3.0.1\",\"servers\":[{\"url\":\"google.com\"}]}"),
	APIUpdateSeverity: "updateseverity",
	AuthPolicy:        "pass-through",
	Description:       "description",
	Documentation:     []byte("docs"),
	ID:                "16810512",
	Image:             "",
	ImageContentType:  "",
	ResourceType:      "oas3",
	ServiceAttributes: map[string]string{"checksum": "110a1bc03b2e3d4875185b0b9f711e3ce2200e8fcc3743ad27dca5d34da9f64b"},
	Stage:             "Sandbox",
	State:             "state",
	Status:            "status",
	SubscriptionName:  "sub name",
	Tags:              []string{"tag1"},
	Title:             "petstore-3",
	URL:               "https://petstore3.us-e2.cloudhub.io",
	Version:           "1.0.0",
}

func TestPublish(t *testing.T) {
	mp := &mockAPIPublisher{
		hitCh: make(chan bool),
	}
	pub := &publisher{
		apiChan:     make(chan *ServiceDetail, 3),
		stopPublish: make(chan bool),
		publishAPI:  mp.mockPublishAPI,
	}
	go pub.Loop()
	// send 3 events to the channel
	pub.apiChan <- sd

	isDone := <-mp.hitCh

	assert.True(t, isDone)
	pub.OnConfigChange(&config.MulesoftConfig{})
	pub.Stop()
}

func TestPublishError(t *testing.T) {

}

func Test_buildServiceBody(t *testing.T) {
	apicSvc, err := BuildServiceBody(sd)
	assert.Nil(t, err)
	assert.Equal(t, sd.APIName, apicSvc.APIName)
	assert.Equal(t, sd.APISpec, apicSvc.SpecDefinition)
	assert.Equal(t, sd.APIUpdateSeverity, apicSvc.APIUpdateSeverity)
	assert.Equal(t, sd.AuthPolicy, apicSvc.AuthPolicy)
	assert.Equal(t, sd.Documentation, apicSvc.Documentation)
	assert.Equal(t, sd.ID, apicSvc.RestAPIID)
	assert.Equal(t, sd.Image, apicSvc.Image)
	assert.Equal(t, sd.ImageContentType, apicSvc.ImageContentType)
	assert.Equal(t, sd.ResourceType, apicSvc.ResourceType)
	assert.Equal(t, sd.ServiceAttributes, apicSvc.ServiceAttributes)
	assert.Equal(t, sd.Stage, apicSvc.Stage)
	assert.Equal(t, sd.State, apicSvc.State)
	assert.Equal(t, sd.Status, apicSvc.Status)
	assert.Equal(t, sd.SubscriptionName, apicSvc.SubscriptionName)
	assert.Equal(t, map[string]interface{}{"tag1": true}, apicSvc.Tags)
	assert.Equal(t, sd.Title, apicSvc.NameToPush)
	assert.Equal(t, sd.URL, apicSvc.URL)
	assert.Equal(t, sd.Version, apicSvc.Version)
}

type mockAPIPublisher struct {
	hitCh chan bool
	count int
}

func (mp *mockAPIPublisher) mockPublishAPI(_ apic.ServiceBody) error {
	mp.count = mp.count + 1
	mp.hitCh <- true
	return nil
}
