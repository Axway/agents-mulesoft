package subscription

import (
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/stretchr/testify/assert"

	"github.com/Axway/agent-sdk/pkg/apic"
	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	uc "github.com/Axway/agent-sdk/pkg/apic/unifiedcatalog/models"
	"github.com/Axway/agents-mulesoft/pkg/anypoint"
	"github.com/sirupsen/logrus"
)

var subID = "1"

var mockSub = &apic.MockSubscription{
	ID:                   subID,
	Description:          "description",
	Name:                 "name",
	ApicID:               "2",
	RemoteAPIID:          "3",
	RemoteAPIStage:       "sandbox",
	CatalogID:            "4",
	UserID:               "5",
	State:                apic.PublishedState,
	PropertyVals:         map[string]string{},
	ReceivedValues:       map[string]interface{}{},
	ReceivedAppName:      "app",
	ReceivedUpdatedEnum:  "",
	UpdateStateErr:       nil,
	UpdateEnumErr:        nil,
	UpdatePropertiesErr:  nil,
	UpdatePropertyValErr: nil,
}

var centralSub = apic.CentralSubscription{
	CatalogItemSubscription: &uc.CatalogItemSubscription{
		Id: subID,
	},
	ApicID:         "apicid",
	RemoteAPIID:    "remoteid",
	RemoteAPIStage: "remotestge",
}

func TestManagerRegisterNewSchema(t *testing.T) {
	cig := &mockConsumerInstanceGetter{}
	sg := &mockSubscriptionGetter{}
	client := &anypoint.MockAnypointClient{}
	manager := New(logrus.StandardLogger(), cig, sg, client)
	assert.NotNil(t, manager)

	sc1 := func(client anypoint.Client) Handler {
		mh := &mockHandler{}
		mh.On("Name").Return("first")
		return mh
	}
	sc2 := func(client anypoint.Client) Handler {
		mh := &mockHandler{}
		mh.On("Name").Return("second")
		return mh
	}
	manager.RegisterNewSchema(sc1, client)
	manager.RegisterNewSchema(sc2, client)
	assert.Equal(t, 2, len(manager.handlers))
	assert.Contains(t, manager.handlers, "first")
	assert.Contains(t, manager.handlers, "second")
	assert.Equal(t, 2, len(manager.Schemas()))

	assert.Equal(t, true, manager.ValidateSubscription(mockSub))
	assert.Equal(t, false, manager.ValidateSubscription(mockSub))
}

func TestProcessSubscribe(t *testing.T) {
	t.Skip()

	type sgReturns struct {
		centralSubs []apic.CentralSubscription
		err         error
	}
	type cigReturns struct {
		consumer *v1alpha1.ConsumerInstance
		err      error
	}

	tests := []struct {
		name       string
		sgReturns  sgReturns
		cigReturns cigReturns
		err        error
		sub        apic.Subscription
	}{
		{
			name: "should create a subscription",
			sgReturns: sgReturns{
				centralSubs: []apic.CentralSubscription{centralSub},
				err:         nil,
			},
			cigReturns: cigReturns{
				err: nil,
				consumer: &v1alpha1.ConsumerInstance{
					Spec: v1alpha1.ConsumerInstanceSpec{
						Subscription: v1alpha1.ConsumerInstanceSpecSubscription{
							Enabled:                true,
							AutoSubscribe:          false,
							SubscriptionDefinition: "sofake",
						},
					},
				},
			},
			sub: mockSub,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cig := &mockConsumerInstanceGetter{}
			cig.On("GetConsumerInstanceByID").Return(tc.cigReturns.consumer, tc.cigReturns.err)

			sg := &mockSubscriptionGetter{}
			sg.On("GetSubscriptionsForCatalogItem").Return(tc.sgReturns.centralSubs, tc.sgReturns.err)

			client := &anypoint.MockAnypointClient{}

			manager := New(logrus.StandardLogger(), cig, sg, client)

			sc := func(client anypoint.Client) Handler {
				mh := &mockHandler{}
				mh.On("Name").Return("sofake")
				return mh
			}
			manager.RegisterNewSchema(sc, client)

			err := manager.processSubscribe(tc.sub)
			assert.Nil(t, err)
		})
	}
}

type mockConsumerInstanceGetter struct {
	mock.Mock
}

func (m *mockConsumerInstanceGetter) GetConsumerInstanceByID(id string) (*v1alpha1.ConsumerInstance, error) {
	args := m.Called()
	ci := args.Get(0).(*v1alpha1.ConsumerInstance)
	return ci, args.Error(1)
}

type mockSubscriptionGetter struct {
	mock.Mock
}

func (m *mockSubscriptionGetter) GetSubscriptionsForCatalogItem(states []string, id string) ([]apic.CentralSubscription, error) {
	args := m.Called()
	cs := args.Get(0).([]apic.CentralSubscription)
	return cs, args.Error(1)
}

type mockHandler struct {
	mock.Mock
}

func (m *mockHandler) Schema() apic.SubscriptionSchema {
	return nil
}

func (m *mockHandler) Name() string {
	args := m.Called()
	return args.String(0)
}

func (m *mockHandler) IsApplicable(policyDetail PolicyDetail) bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *mockHandler) Subscribe(log logrus.FieldLogger, subs apic.Subscription) error {
	args := m.Called()
	return args.Error(0)
}

func (m *mockHandler) Unsubscribe(log logrus.FieldLogger, subs apic.Subscription) error {
	return nil
}
