package slatier

import (
	"testing"

	"github.com/Axway/agents-mulesoft/pkg/config"
	"github.com/stretchr/testify/assert"

	"github.com/Axway/agent-sdk/pkg/apic"
	"github.com/Axway/agents-mulesoft/pkg/anypoint"
)

func TestSlaTier(t *testing.T) {
	name := "slatier"
	schema := apic.NewSubscriptionSchema(name)
	client := &anypoint.MockAnypointClient{}
	schema.AddProperty(anypoint.AppName, "string", "Name of the new app", "", true, nil)
	contract := NewSLATierContract(name, schema, client)
	assert.Equal(t, name, contract.Name())
	assert.Equal(t, schema, contract.Schema())
	pd := config.PolicyDetail{
		Policy:     apic.Apikey,
		IsSLABased: true,
		APIId:      name,
	}
	assert.True(t, contract.IsApplicable(pd))

	pd = config.PolicyDetail{
		Policy:     apic.Apikey,
		IsSLABased: false,
		APIId:      name,
	}
	assert.False(t, contract.IsApplicable(pd))
}
