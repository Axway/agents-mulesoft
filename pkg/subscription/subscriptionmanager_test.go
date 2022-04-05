package subscription

import (
	"testing"

	"github.com/Axway/agents-mulesoft/pkg/common"
	"github.com/stretchr/testify/mock"

	"github.com/stretchr/testify/assert"

	"github.com/Axway/agent-sdk/pkg/apic"
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
	mms := &MockMuleSubscriptionClient{}
	manager := NewManager(logrus.StandardLogger(), mms)
	assert.NotNil(t, manager)

	mh1 := &MockSubSchema{}
	mh1.On("Name").Return("first")
	mh1.On("Schema").Return("sofake schema")

	mh2 := &MockSubSchema{}
	mh2.On("Name").Return("second")
	mh2.On("Schema").Return("sofake schema")

	manager.RegisterNewSchema(mh1)
	manager.RegisterNewSchema(mh2)

	assert.Equal(t, 2, len(manager.schemas))
	assert.Contains(t, manager.schemas, "first")
	assert.Contains(t, manager.schemas, "second")
	assert.Equal(t, 2, len(manager.schemas))

	assert.Equal(t, true, manager.ValidateSubscription(mockSub))
	assert.Equal(t, false, manager.ValidateSubscription(mockSub))
}

func Test_processForState(t *testing.T) {
	tests := []struct {
		name              string
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
			sub:               mockSub,
		},
		{
			name:              "should unsubscribe from a subscription",
			state:             apic.SubscriptionUnsubscribeInitiated,
			subHandlerReturns: nil,
			err:               nil,
			sub:               mockSub,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {

			mms := &MockMuleSubscriptionClient{
				app: &anypoint.Application{ID: 123},
				err: tc.err,
			}
			manager := NewManager(logrus.StandardLogger(), mms)

			mc := &MockSubSchema{}
			mc.On("Name").Return("fake")
			mc.On("Subscribe").Return(tc.subHandlerReturns)

			manager.RegisterNewSchema(mc)

			err := manager.processForState(tc.sub, tc.state)
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
	assert.Contains(t, fields, "catalogItemID")
}

func TestValidateSubscription(t *testing.T) {
	schemaName := "first"
	mc := &MockSubSchema{}
	mc.On("Name").Return("first")
	mc.On("Schema").Return("sofake")
	mc.On("IsApplicable").Return(true)

	mms := &MockMuleSubscriptionClient{}
	manager := NewManager(logrus.StandardLogger(), mms, mc)
	isTrue := manager.ValidateSubscription(mockSub)
	assert.True(t, isTrue)

	assert.Equal(t, 1, len(manager.schemas))

	isFalse := manager.ValidateSubscription(mockSub)
	assert.False(t, isFalse)

	name := manager.GetSubscriptionSchemaName(common.PolicyDetail{
		Policy:     anypoint.ClientID,
		IsSLABased: false,
		APIId:      "1",
	})

	assert.Equal(t, schemaName, name)
}

type MockSubSchema struct {
	mock.Mock
}

func (m *MockSubSchema) Schema() apic.SubscriptionSchema {
	args := m.Called()
	return apic.NewSubscriptionSchema(args.String(0))
}

func (m *MockSubSchema) Name() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockSubSchema) IsApplicable(common.PolicyDetail) bool {
	args := m.Called()
	return args.Bool(0)
}
