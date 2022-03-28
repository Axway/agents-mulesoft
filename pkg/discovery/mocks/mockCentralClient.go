package mocks

import (
	mc "github.com/Axway/agent-sdk/pkg/apic/mock"
	"github.com/stretchr/testify/mock"
)

// MockCentralClient wraps the mock client defined in the sdk, and uses the mock package to define mocks to return
type MockCentralClient struct {
	mock.Mock
	mc.Client
}
