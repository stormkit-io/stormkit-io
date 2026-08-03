package sysstats

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
)

type ParserSuite struct {
	suite.Suite
	fixture string
}

func (s *ParserSuite) SetupTest() {
	data, err := os.ReadFile("testdata/node_exporter.txt")
	s.Require().NoError(err)
	s.fixture = string(data)
}

func (s *ParserSuite) parse(payload string) *parser {
	p, err := newParser(strings.NewReader(payload))
	s.Require().NoError(err)
	return p
}

func (s *ParserSuite) Test_Sample_ReadsMachineWideValues() {
	sample, _ := s.parse(s.fixture).sample(nil)

	s.True(sample.Reachable)
	s.Equal(2, sample.CPUCores)
	s.Equal(uint64(16000000000), sample.MemTotalBytes)
	s.Equal(uint64(6000000000), sample.MemAvailableBytes)
	s.Equal(uint64(10000000000), sample.MemUsedBytes())
	s.Equal(0.85, sample.Load1)
	s.Equal(0.61, sample.Load5)
	s.Equal(0.42, sample.Load15)
	s.Equal(int64(1754100000), sample.BootTime)
}

// Every real disk must be present. This is the requirement that reading stats
// from inside the container could not meet.
func (s *ParserSuite) Test_Filesystems_IncludesEveryRealDisk() {
	sample, _ := s.parse(s.fixture).sample(nil)

	s.Require().Len(sample.Filesystems, 2)

	root := sample.Filesystems[0]
	s.Equal("/", root.Mountpoint)
	s.Equal("/dev/sda1", root.Device)
	s.Equal("ext4", root.FSType)
	s.Equal(uint64(50000000000), root.SizeBytes)
	s.Equal(uint64(20000000000), root.AvailBytes)
	s.Equal(uint64(30000000000), root.UsedBytes())

	// A second physical disk with no relation to Stormkit's own volume.
	data := sample.Filesystems[1]
	s.Equal("/mnt/data", data.Mountpoint)
	s.Equal(uint64(1000000000000), data.SizeBytes)
}

func (s *ParserSuite) Test_Filesystems_SkipsVirtualAndEmpty() {
	sample, _ := s.parse(s.fixture).sample(nil)

	for _, fs := range sample.Filesystems {
		s.NotEqual("tmpfs", fs.FSType)
		s.NotEqual("overlay", fs.FSType)
		s.NotEqual("/mnt/empty", fs.Mountpoint, "zero-sized filesystems carry no information")
	}
}

func (s *ParserSuite) Test_Network_ExcludesLoopbackAndBridges() {
	sample, _ := s.parse(s.fixture).sample(nil)

	s.Equal(uint64(1500000000), sample.NetReceiveBytes)
	s.Equal(uint64(750000000), sample.NetTransmitBytes)
}

func (s *ParserSuite) Test_CPU_IsInvalidOnFirstSample() {
	sample, counters := s.parse(s.fixture).sample(nil)

	s.False(sample.CPUValid, "no rate exists without a previous reading")
	s.Equal(float64(0), sample.CPUPercent)
	s.Equal(float64(2200), counters.idle)
	s.Equal(float64(2700), counters.total)
}

func (s *ParserSuite) Test_CPU_RateAcrossTwoSamples() {
	_, first := s.parse(s.fixture).sample(nil)

	// 100s elapsed, 75s of it idle across both cores => 25% busy.
	next := strings.NewReplacer(
		`mode="idle"} 1000`, `mode="idle"} 1050`,
		`mode="idle"} 1200`, `mode="idle"} 1225`,
		`cpu="0",mode="user"} 200`, `cpu="0",mode="user"} 215`,
		`cpu="1",mode="user"} 150`, `cpu="1",mode="user"} 160`,
	).Replace(s.fixture)

	sample, _ := s.parse(next).sample(&first)

	s.True(sample.CPUValid)
	s.InDelta(25, sample.CPUPercent, 0.001)
}

// An exporter restart resets the counters. Reporting a gap is correct; a
// negative or wildly large spike is not.
func (s *ParserSuite) Test_CPU_DiscardsIntervalOnCounterReset() {
	_, first := s.parse(s.fixture).sample(nil)

	restarted := strings.NewReplacer(
		`mode="idle"} 1000`, `mode="idle"} 5`,
		`mode="idle"} 1200`, `mode="idle"} 6`,
		`cpu="0",mode="system"} 100`, `cpu="0",mode="system"} 1`,
		`cpu="0",mode="user"} 200`, `cpu="0",mode="user"} 2`,
		`cpu="1",mode="system"} 50`, `cpu="1",mode="system"} 1`,
		`cpu="1",mode="user"} 150`, `cpu="1",mode="user"} 2`,
	).Replace(s.fixture)

	sample, _ := s.parse(restarted).sample(&first)

	s.False(sample.CPUValid)
	s.Equal(float64(0), sample.CPUPercent)
}

func (s *ParserSuite) Test_CPU_ClampsToRange() {
	percent, valid := cpuPercent(&cpuCounters{idle: 100, total: 100}, cpuCounters{idle: 100, total: 200})
	s.True(valid)
	s.Equal(float64(100), percent, "no idle time in the interval means fully busy")

	percent, valid = cpuPercent(&cpuCounters{idle: 100, total: 100}, cpuCounters{idle: 200, total: 200})
	s.True(valid)
	s.Equal(float64(0), percent)
}

func (s *ParserSuite) Test_Sample_ToleratesMissingSeries() {
	sample, _ := s.parse("# TYPE node_load1 gauge\nnode_load1 1.5\n").sample(nil)

	s.True(sample.Reachable)
	s.Equal(1.5, sample.Load1)
	s.Empty(sample.Filesystems)
	s.Equal(uint64(0), sample.MemTotalBytes)
}

func TestParserSuite(t *testing.T) {
	suite.Run(t, new(ParserSuite))
}
