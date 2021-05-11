package mocks

import (
	"github.com/Axway/agent-sdk/pkg/apic"
	"github.com/Axway/agents-mulesoft/pkg/config"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/mock"
)

type MockContract struct {
	mock.Mock
}

func (m *MockContract) Schema() apic.SubscriptionSchema {
	args := m.Called()
	return apic.NewSubscriptionSchema(args.String(0))
}

func (m *MockContract) Name() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockContract) IsApplicable(config.PolicyDetail) bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockContract) Subscribe(logrus.FieldLogger, apic.Subscription) error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockContract) Unsubscribe(logrus.FieldLogger, apic.Subscription) error {
	return nil
}
