package subscription

import (
	"github.com/Axway/agent-sdk/pkg/apic"
	"github.com/Axway/agents-mulesoft/pkg/common"
)

const name = "client-id-enforcement"

type clientID struct {
	name   string
	schema apic.SubscriptionSchema
}

// NewClientIDContract creates a new  client ID schema
func NewClientIDContract() SubSchema {
	schema := apic.NewSubscriptionSchema(name)

	schema.AddProperty(common.AppName,
		"string",
		"Name of the new app",
		"",
		true,
		nil,
	)

	schema.AddProperty(common.Description,
		"string",
		"Description",
		"",
		false,
		nil,
	)

	return &clientID{
		name:   name,
		schema: schema,
	}
}

func (c *clientID) IsApplicable(pd common.PolicyDetail) bool {
	return pd.Policy == apic.Apikey && pd.IsSLABased == false
}

func (c *clientID) Schema() apic.SubscriptionSchema {
	return c.schema
}

func (c *clientID) Name() string {
	return c.name
}
