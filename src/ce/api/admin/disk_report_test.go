package admin_test

import (
	"encoding/json"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/lib/rediscache"
	"github.com/stormkit-io/stormkit-io/src/lib/utils/nixstore"
	"github.com/stormkit-io/stormkit-io/src/lib/utils/nixstore/nixstoretest"
	"github.com/stormkit-io/stormkit-io/src/mocks"
	"github.com/stretchr/testify/suite"
)

type DiskReportSuite struct {
	suite.Suite
	mockService  *mocks.MicroServiceInterface
	originalPath string
	restore      func()
}

func (s *DiskReportSuite) BeforeTest(_, _ string) {
	s.mockService = &mocks.MicroServiceInterface{}
	rediscache.DefaultService = s.mockService

	s.originalPath = nixstore.DefaultPath
	nixstore.DefaultPath = s.T().TempDir()
	s.restore = nixstoretest.StubLookPath(false)
}

func (s *DiskReportSuite) AfterTest(_, _ string) {
	rediscache.DefaultService = nil
	nixstore.DefaultPath = s.originalPath
	s.restore()
}

func (s *DiskReportSuite) Test_CurrentDiskReport() {
	report, err := admin.CurrentDiskReport()

	s.NoError(err)
	s.Greater(report.Root.TotalBytes, uint64(0))
	s.False(report.ReportedAt.IsZero())
	s.Nil(report.NixStore, "a container without Nix should not report a store")
}

func (s *DiskReportSuite) Test_CurrentDiskReport_WithNixStore() {
	s.restore()
	s.restore = nixstoretest.StubLookPath(true)

	report, err := admin.CurrentDiskReport()

	s.NoError(err)
	s.Require().NotNil(report.NixStore)
	s.Equal(nixstore.DefaultPath, report.NixStore.Path)
}

func (s *DiskReportSuite) Test_DiskReports() {
	services := []string{rediscache.ServiceHosting, rediscache.ServiceWorkerserver}

	s.mockService.On("List", services).Return([]*rediscache.MicroService{
		{ID: "ws-1", Name: rediscache.ServiceWorkerserver},
		{ID: "host-1", Name: rediscache.ServiceHosting},
	}, nil).Once()

	s.mockService.On("GetAll", rediscache.KEY_DISK_USAGE, services).Return(map[string]string{
		"ws-1":   s.encode(admin.DiskReport{Root: nixstore.Usage{TotalBytes: 100, FreeBytes: 40}}),
		"host-1": s.encode(admin.DiskReport{Root: nixstore.Usage{TotalBytes: 200, FreeBytes: 20}}),
	}, nil).Once()

	reports, err := admin.DiskReports()

	s.NoError(err)
	s.Require().Len(reports, 2)

	// Sorted by service name so the admin view does not reshuffle on every poll.
	s.Equal(rediscache.ServiceHosting, reports[0].ServiceName)
	s.Equal("host-1", reports[0].ServiceID)
	s.Equal(uint64(200), reports[0].Root.TotalBytes)
	s.Equal(rediscache.ServiceWorkerserver, reports[1].ServiceName)
}

// A container that has not published yet is left out rather than shown with
// zeroes, which would read as a full disk.
func (s *DiskReportSuite) Test_DiskReports_SkipsSilentServices() {
	services := []string{rediscache.ServiceHosting, rediscache.ServiceWorkerserver}

	s.mockService.On("List", services).Return([]*rediscache.MicroService{
		{ID: "host-1", Name: rediscache.ServiceHosting},
		{ID: "ws-1", Name: rediscache.ServiceWorkerserver},
	}, nil).Once()

	s.mockService.On("GetAll", rediscache.KEY_DISK_USAGE, services).Return(map[string]string{
		"host-1": s.encode(admin.DiskReport{Root: nixstore.Usage{TotalBytes: 100}}),
		"ws-1":   "",
	}, nil).Once()

	reports, err := admin.DiskReports()

	s.NoError(err)
	s.Require().Len(reports, 1)
	s.Equal("host-1", reports[0].ServiceID)
}

func (s *DiskReportSuite) Test_DiskReports_SkipsUndecodableReports() {
	services := []string{rediscache.ServiceHosting, rediscache.ServiceWorkerserver}

	s.mockService.On("List", services).Return([]*rediscache.MicroService{
		{ID: "host-1", Name: rediscache.ServiceHosting},
	}, nil).Once()

	s.mockService.On("GetAll", rediscache.KEY_DISK_USAGE, services).Return(map[string]string{
		"host-1": "not json",
	}, nil).Once()

	reports, err := admin.DiskReports()

	s.NoError(err)
	s.Empty(reports)
}

func (s *DiskReportSuite) encode(r admin.DiskReport) string {
	b, err := json.Marshal(r)
	s.Require().NoError(err)

	return string(b)
}

func TestDiskReportSuite(t *testing.T) {
	suite.Run(t, new(DiskReportSuite))
}
