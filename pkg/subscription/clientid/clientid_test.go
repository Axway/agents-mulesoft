package clientid

import (
	"testing"

	"github.com/Axway/agent-sdk/pkg/apic"
	"github.com/Axway/agents-mulesoft/pkg/anypoint"
	"github.com/Axway/agents-mulesoft/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestClientID(t *testing.T) {
	clientIDPolicy := newClientID()
	assert.Equal(t, name, clientIDPolicy.Name())
	assert.NotEmpty(t, clientIDPolicy.Schema().GetProperty(anypoint.AppName))
	assert.NotEmpty(t, clientIDPolicy.Schema().GetProperty(anypoint.Description))
	pd := config.PolicyDetail{
		Policy:     apic.Apikey,
		IsSLABased: false,
		APIId:      name,
	}
	assert.True(t, clientIDPolicy.IsApplicable(pd))
}
