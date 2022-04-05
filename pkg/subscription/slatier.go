package subscription

import (
	"github.com/Axway/agent-sdk/pkg/apic"
	prov "github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/Axway/agents-mulesoft/pkg/anypoint"
	"github.com/Axway/agents-mulesoft/pkg/common"
)

type slaTier struct {
	name   string
	schema apic.SubscriptionSchema
}

func NewSLATierContractSchemaMP(name string, tierNames []string) prov.SchemaBuilder {
	tier := prov.NewSchemaPropertyBuilder().
		SetName(common.SlaTier).
		SetLabel(anypoint.TierLabel).
		SetRequired().
		IsString().
		SetEnumValues(tierNames)

	return prov.NewSchemaBuilder().
		SetName(name).
		AddProperty(tier)
}

// NewSLATierContractSchemaUC creates a new subscribable sla-tier policy
func NewSLATierContractSchemaUC(name string, tierNames []string) SubSchema {
	schema := apic.NewSubscriptionSchema(name)
	schema.AddProperty(
		anypoint.AppName,
		"string",
		"Name of the new app",
		"",
		true,
		nil,
	)
	schema.AddProperty(
		anypoint.Description,
		"string",
		"",
		"",
		false,
		nil,
	)
	schema.AddProperty(
		anypoint.TierLabel,
		"string",
		"",
		"",
		true,
		tierNames,
	)

	return &slaTier{
		name:   name,
		schema: schema,
	}
}

func (s *slaTier) Name() string {
	return s.name
}

func (s *slaTier) Schema() apic.SubscriptionSchema {
	return s.schema
}

func (s *slaTier) IsApplicable(pd common.PolicyDetail) bool {
	if pd.IsSLABased {
		return pd.APIId == s.name && pd.Policy == apic.Apikey
	}
	return false
}
