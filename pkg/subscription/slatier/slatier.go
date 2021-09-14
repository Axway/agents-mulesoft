package slatier

import (
	"github.com/Axway/agents-mulesoft/pkg/subscription"

	"github.com/Axway/agents-mulesoft/pkg/config"

	"github.com/Axway/agent-sdk/pkg/apic"
	"github.com/Axway/agents-mulesoft/pkg/anypoint"
)

type slaTier struct {
	name   string
	schema apic.SubscriptionSchema
}

// newSlaTier Creates a new *slaTier
func newSLATier(name string, schema apic.SubscriptionSchema) *slaTier {
	return &slaTier{
		name:   name,
		schema: schema,
	}
}

// NewSLATierContract creates a new subscribable contract for the sla-tier policy
func NewSLATierContract(name string, schema apic.SubscriptionSchema, client anypoint.Client) *subscription.SubStateManager {
	return subscription.NewSubStateManager(client, newSLATier(name, schema))
}

func (s *slaTier) Name() string {
	return s.name
}

func (s *slaTier) Schema() apic.SubscriptionSchema {
	return s.schema
}

func (s *slaTier) IsApplicable(pd config.PolicyDetail) bool {
	if pd.IsSLABased {
		return pd.APIId == s.name && pd.Policy == apic.Apikey
	}
	return false
}
