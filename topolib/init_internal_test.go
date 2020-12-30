package topolib

import (
	"context"
	"net"

	"github.com/stretchr/testify/mock"
)

type ProviderMock struct {
	mock.Mock
}

func (m *ProviderMock) Lookup(ctx context.Context, ip net.IP) (ProviderLookupResult, error) {
	args := m.Called(ctx, ip)

	return args.Get(0).(ProviderLookupResult), args.Error(1)
}

func (m *ProviderMock) Name() string {
	return m.Called().String(0)
}
