package admin

import (
	"github.com/stretchr/testify/mock"
)

// MockLifecycleManager is a mock implementation of ILifecycleManager
type MockLifecycleManager struct {
	mock.Mock
}

func (m *MockLifecycleManager) CreateInstance(port int) (*Instance, error) {
	args := m.Called(port)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Instance), args.Error(1)
}

func (m *MockLifecycleManager) CreateInstanceWithConfig(port int, customConfig *InstanceConfig) (*Instance, error) {
	args := m.Called(port, customConfig)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Instance), args.Error(1)
}

func (m *MockLifecycleManager) UpdateInstanceConfig(port int, customConfig *InstanceConfig) (*Instance, error) {
	args := m.Called(port, customConfig)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Instance), args.Error(1)
}

func (m *MockLifecycleManager) ListInstances() ([]*Instance, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*Instance), args.Error(1)
}

func (m *MockLifecycleManager) GetInstance(port int) (*Instance, error) {
	args := m.Called(port)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Instance), args.Error(1)
}

func (m *MockLifecycleManager) DeleteInstance(port int) error {
	args := m.Called(port)
	return args.Error(0)
}

func (m *MockLifecycleManager) IsHealthy() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockLifecycleManager) Ping() error {
	args := m.Called()
	return args.Error(0)
}
