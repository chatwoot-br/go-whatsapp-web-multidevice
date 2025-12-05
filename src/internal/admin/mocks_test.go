package admin

import (
	"context"

	"github.com/stretchr/testify/mock"
)

// MockLifecycleManager is a mock implementation of ILifecycleManager
type MockLifecycleManager struct {
	mock.Mock
}

func (m *MockLifecycleManager) CreateInstance(ctx context.Context, port int) (*Instance, error) {
	args := m.Called(ctx, port)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Instance), args.Error(1)
}

func (m *MockLifecycleManager) CreateInstanceWithConfig(ctx context.Context, port int, customConfig *InstanceConfig) (*Instance, error) {
	args := m.Called(ctx, port, customConfig)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Instance), args.Error(1)
}

func (m *MockLifecycleManager) UpdateInstanceConfig(ctx context.Context, port int, customConfig *InstanceConfig) (*Instance, error) {
	args := m.Called(ctx, port, customConfig)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Instance), args.Error(1)
}

func (m *MockLifecycleManager) ListInstances(ctx context.Context) ([]*Instance, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*Instance), args.Error(1)
}

func (m *MockLifecycleManager) GetInstance(ctx context.Context, port int) (*Instance, error) {
	args := m.Called(ctx, port)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Instance), args.Error(1)
}

func (m *MockLifecycleManager) DeleteInstance(ctx context.Context, port int) error {
	args := m.Called(ctx, port)
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
