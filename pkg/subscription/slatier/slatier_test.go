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
	schema.AddProperty(anypoint.AppName, "string", "Name of the new app", "", true, nil)
	slaFunc := NewSLATierContract(name, schema)
	slaPolicy := slaFunc(&anypoint.MockAnypointClient{})
	assert.Equal(t, name, slaPolicy.Name())
	assert.Equal(t, schema, slaPolicy.Schema())
	pd := config.PolicyDetail{
		Policy:     apic.Apikey,
		IsSlaBased: true,
		APIId:      name,
	}
	assert.True(t, slaPolicy.IsApplicable(pd))

	pd = config.PolicyDetail{
		Policy:     apic.Apikey,
		IsSlaBased: false,
		APIId:      name,
	}
	assert.False(t, slaPolicy.IsApplicable(pd))
}
