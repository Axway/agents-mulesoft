package subscription

import (
	"testing"

	"github.com/Axway/agents-mulesoft/pkg/common"
	"github.com/stretchr/testify/assert"

	"github.com/Axway/agent-sdk/pkg/apic"
)

func TestSlaTier(t *testing.T) {
	name := "slatier"
	contract := NewSLATierContractSchemaUC(name, []string{"tier1", "tier2"})
	assert.Equal(t, name, contract.Name())
	assert.NotNil(t, contract.Schema().GetProperty(common.AppName))
	assert.NotNil(t, contract.Schema().GetProperty(common.Description))
	assert.NotNil(t, contract.Schema().GetProperty(common.TierLabel))
	pd := common.PolicyDetail{
		Policy:     apic.Apikey,
		IsSLABased: true,
		APIId:      name,
	}
	assert.True(t, contract.IsApplicable(pd))

	pd = common.PolicyDetail{
		Policy:     apic.Apikey,
		IsSLABased: false,
		APIId:      name,
	}
	assert.False(t, contract.IsApplicable(pd))
}
