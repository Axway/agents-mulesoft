package subscription

import (
	"fmt"
	"testing"

	"github.com/Axway/agents-mulesoft/pkg/common"

	"github.com/Axway/agents-mulesoft/pkg/config"

	"github.com/Axway/agents-mulesoft/pkg/subscription/mocks"

	"github.com/stretchr/testify/mock"

	"github.com/stretchr/testify/assert"

	"github.com/Axway/agent-sdk/pkg/apic"
	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agents-mulesoft/pkg/anypoint"
	"github.com/sirupsen/logrus"
)

var subID = "8a2e92fd79191ffb017958aca9076781"
var mockRemoteAPIID = "16806513"

var mockSub = &apic.MockSubscription{
	ID:             subID,
	Description:    "description",
	Name:           "name",
	ApicID:         "8a2e924a7943c8d501795810a5fd1bbc",
	RemoteAPIID:    mockRemoteAPIID,
	RemoteAPIStage: "1.0.5",
	CatalogID:      "8a2e855979191de701795810a8f82f3a",
	UserID:         "5",
	State:          apic.PublishedState,
	PropertyVals: map[string]interface{}{
		anypoint.AppName:     "mule-app",
		anypoint.Description: "desc",
		anypoint.TierLabel:   "666892-gold",
	},
	RemoteAPIAttributes: map[string]string{
		common.AttrAPIID:          "16810513",
		common.AttrProductVersion: "1.0.5",
	},
	ReceivedValues:               map[string]interface{}{},
	ReceivedAppName:              "",
	ReceivedUpdatedEnum:          "",
	UpdateStateErr:               nil,
	UpdateEnumErr:                nil,
	UpdatePropertiesErr:          nil,
	UpdatePropertyValErr:         nil,
	UpdateStateWithPropertiesErr: nil,
}

func TestManagerRegisterNewSchema(t *testing.T) {
	cig := &mockConsumerInstanceGetter{}

	manager := New(logrus.StandardLogger(), cig)
	assert.NotNil(t, manager)

	mh1 := &mocks.MockContract{}
	mh1.On("Name").Return("first")
	mh1.On("Schema").Return("sofake schema")

	mh2 := &mocks.MockContract{}
	mh2.On("Name").Return("second")
	mh2.On("Schema").Return("sofake schema")

	manager.RegisterNewSchema(mh1)
	manager.RegisterNewSchema(mh2)

	assert.Equal(t, 2, len(manager.handlers))
	assert.Contains(t, manager.handlers, "first")
	assert.Contains(t, manager.handlers, "second")
	assert.Equal(t, 2, len(manager.Schemas()))

	assert.Equal(t, true, manager.ValidateSubscription(mockSub))
	assert.Equal(t, false, manager.ValidateSubscription(mockSub))
}

func Test_processForState(t *testing.T) {
	type cigReturns struct {
		consumer *v1alpha1.ConsumerInstance
		err      error
	}

	tests := []struct {
		name              string
		cigReturns        cigReturns
		err               error
		sub               apic.Subscription
		subHandlerReturns error
		state             apic.SubscriptionState
	}{
		{
			name:              "should create a subscription",
			state:             apic.SubscriptionApproved,
			subHandlerReturns: nil,
			err:               nil,
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
		{
			name:              "should return an error when calling Subscribe on the handdler",
			state:             apic.SubscriptionApproved,
			subHandlerReturns: fmt.Errorf("errs"),
			err:               fmt.Errorf("errs"),
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
		{
			name:              "should return an error when calling GetConsumerInstanceByID",
			state:             apic.SubscriptionApproved,
			subHandlerReturns: nil,
			err:               fmt.Errorf("errrr"),
			cigReturns: cigReturns{
				err:      fmt.Errorf("errrr"),
				consumer: nil,
			},
			sub: mockSub,
		},
		{
			name:              "should unsubscribe from a subscription",
			state:             apic.SubscriptionUnsubscribeInitiated,
			subHandlerReturns: nil,
			err:               nil,
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

			manager := New(logrus.StandardLogger(), cig)

			mh := &mocks.MockContract{}
			mh.On("Name").Return("sofake")
			mh.On("Subscribe").Return(tc.subHandlerReturns)

			manager.RegisterNewSchema(mh)

			err := manager.processForState(tc.sub, &logrus.Logger{}, tc.state)
			if tc.err != nil {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func Test_setLogFields(t *testing.T) {
	fields := logFields(mockSub)
	assert.Contains(t, fields, "subscriptionName")
	assert.Contains(t, fields, "subscriptionID")
	assert.Contains(t, fields, "catalogItemID")
	assert.Contains(t, fields, "remoteID")
	assert.Contains(t, fields, "consumerInstanceID")
	assert.Contains(t, fields, "currentState")
}

func TestValidateSubscription(t *testing.T) {
	cig := &mockConsumerInstanceGetter{}
	schemaName := "first"
	mc := &mocks.MockContract{}
	mc.On("Name").Return("first")
	mc.On("Schema").Return("sofake")
	mc.On("IsApplicable").Return(true)

	manager := New(logrus.StandardLogger(), cig, mc)
	isTrue := manager.ValidateSubscription(mockSub)
	assert.True(t, isTrue)

	assert.Equal(t, 1, len(manager.Schemas()))

	isFalse := manager.ValidateSubscription(mockSub)
	assert.False(t, isFalse)

	name := manager.GetSubscriptionSchemaName(config.PolicyDetail{
		Policy:     anypoint.ClientID,
		IsSLABased: false,
		APIId:      "1",
	})

	assert.Equal(t, schemaName, name)
}

type mockConsumerInstanceGetter struct {
	mock.Mock
}

func (m *mockConsumerInstanceGetter) GetConsumerInstanceByID(id string) (*v1alpha1.ConsumerInstance, error) {
	args := m.Called()
	ci := args.Get(0).(*v1alpha1.ConsumerInstance)
	return ci, args.Error(1)
}
