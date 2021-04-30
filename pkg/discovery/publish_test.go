package discovery

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// func TestPublish(t *testing.T) {
// 	sd := &ServiceDetail{
// 		Title:             "MYAPI",
// 		Version:           "1",
// 		APIName:           "myapi",
// 		URL:               "google.com",
// 		APISpec:           []byte("dummyspec"),
// 		Tags:              []string{"discover"},
// 		ServiceAttributes: nil,
// 	}
// 	agent := &Agent{}
// 	err := agent.publisher.publish(sd)
// 	if err != nil {
// 		t.Error(err)
// 	}
// }

func Test_buildServiceBody(t *testing.T) {
	sd := &ServiceDetail{
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
