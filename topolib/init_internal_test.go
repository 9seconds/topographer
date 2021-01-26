package topolib

import (
	"context"
	"net"
	"time"

	"github.com/spf13/afero"
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

type OfflineProviderMock struct {
	ProviderMock
}

func (m *OfflineProviderMock) Shutdown() {
	m.Called()
}

func (m *OfflineProviderMock) UpdateEvery() time.Duration {
	return m.Called().Get(0).(time.Duration)
}

func (m *OfflineProviderMock) BaseDirectory() string {
	return m.Called().String(0)
}

func (m *OfflineProviderMock) Open(fs *afero.BasePathFs) error {
	return m.Called(fs).Error(0)
}

func (m *OfflineProviderMock) Download(ctx context.Context, fs afero.Afero) error {
	return m.Called(ctx, fs).Error(0)
}

type LoggerMock struct {
	mock.Mock
}

func (m *LoggerMock) LookupError(ip net.IP, name string, err error) {
	m.Called(ip, name, err)
}

func (m *LoggerMock) UpdateInfo(name, msg string) {
	m.Called(name, msg)
}

func (m *LoggerMock) UpdateError(name string, err error) {
	m.Called(name, err)
}
