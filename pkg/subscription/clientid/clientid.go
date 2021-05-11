package clientid

import (
	"github.com/Axway/agent-sdk/pkg/apic"
	"github.com/Axway/agents-mulesoft/pkg/anypoint"
	"github.com/Axway/agents-mulesoft/pkg/config"
	"github.com/Axway/agents-mulesoft/pkg/subscription"
)

const name = "client-id-enforcement"

type clientID struct {
	name   string
	schema apic.SubscriptionSchema
}

// NewClientIDContract creates a new subscribable contract for the client-id policy
func NewClientIDContract(client anypoint.Client) subscription.StateManager {
	return subscription.NewSubStateManager(client, newClientID())
}

// newClientID Creates a newClientID *clientID
func newClientID() *clientID {
	schema := apic.NewSubscriptionSchema(name)

	schema.AddProperty(anypoint.AppName,
		"string",
		"Name of the new app",
		"",
		true,
		nil,
	)

	schema.AddProperty(anypoint.Description,
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

func (c *clientID) IsApplicable(pd config.PolicyDetail) bool {
	return pd.Policy == apic.Apikey && pd.IsSlaBased == false
}

func (c *clientID) Schema() apic.SubscriptionSchema {
	return c.schema
}

func (c *clientID) Name() string {
	return c.name
}
