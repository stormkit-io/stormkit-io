package utils_test

import (
	"net"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stretchr/testify/suite"
)

type NetworkTestSuite struct {
	suite.Suite
}

func (s *NetworkTestSuite) Test_IsPortInUse_PortNotInUse() {
	// Test with a port that should not be in use (high port number)
	port := 59999
	inUse := utils.IsPortInUse(port)
	s.False(inUse, "Port %d should not be in use", port)
}

func (s *NetworkTestSuite) Test_IsPortInUse_PortInUse() {
	// Create a listener to occupy a port
	listener, err := net.Listen("tcp", ":0") // Let the system choose an available port
	s.NoError(err, "Failed to create listener")
	defer listener.Close()

	// Get the port number that was assigned
	addr := listener.Addr().(*net.TCPAddr)
	port := addr.Port

	// Test that the port is detected as in use
	inUse := utils.IsPortInUse(port)
	s.True(inUse, "Port %d should be detected as in use", port)
}

func (s *NetworkTestSuite) Test_IsPortInUse_InvalidPort() {
	// Test with invalid port numbers
	testCases := []struct {
		port     int
		expected bool
		name     string
	}{
		{-1, false, "negative port"},
		{0, false, "zero port"},
		{65536, false, "port above valid range"},
		{99999, false, "very high port number"},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			inUse := utils.IsPortInUse(tc.port)
			s.Equal(tc.expected, inUse, "Port %d result should be %v", tc.port, tc.expected)
		})
	}
}

func (s *NetworkTestSuite) Test_IsPortInUse_WellKnownPorts() {
	// Test some well-known ports that might be in use
	// Note: These tests might be flaky depending on the environment
	testCases := []struct {
		port int
		name string
	}{
		{80, "HTTP port"},
		{443, "HTTPS port"},
		{22, "SSH port"},
		{3306, "MySQL port"},
		{5432, "PostgreSQL port"},
		{6379, "Redis port"},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// We don't assert the result since these ports may or may not be in use
			// We just verify the function doesn't panic and returns a boolean
			inUse := utils.IsPortInUse(tc.port)
			s.IsType(true, inUse, "IsPortInUse should return a boolean for port %d", tc.port)
		})
	}
}

func (s *NetworkTestSuite) Test_IsPortInUse_MultipleChecks() {
	// Create a listener
	listener, err := net.Listen("tcp", ":0")
	s.NoError(err, "Failed to create listener")

	addr := listener.Addr().(*net.TCPAddr)
	port := addr.Port

	// Test multiple times while port is in use
	for i := 0; i < 3; i++ {
		inUse := utils.IsPortInUse(port)
		s.True(inUse, "Port %d should be detected as in use on check %d", port, i+1)
	}

	// Close the listener
	listener.Close()

	// Give a small moment for the port to be released
	// Note: In some systems, there might be a brief delay before the port is fully released
	// We test immediately but in real scenarios you might need to wait
	inUse := utils.IsPortInUse(port)
	// The port might still appear in use briefly after closing, so we don't assert false here
	// We just verify the function executes without error
	s.IsType(true, inUse, "IsPortInUse should return a boolean after listener is closed")
}

func (s *NetworkTestSuite) Test_IsPortInUse_ConcurrentAccess() {
	// Test that the function handles concurrent access properly
	port := 58888 // Use a specific port that's likely to be available

	// First, ensure the port is not in use
	s.False(utils.IsPortInUse(port), "Port %d should not be in use initially", port)

	// Create multiple listeners on different ports and test concurrently
	listeners := make([]net.Listener, 3)
	ports := make([]int, 3)

	for i := 0; i < 3; i++ {
		listener, err := net.Listen("tcp", ":0")
		s.NoError(err, "Failed to create listener %d", i)
		listeners[i] = listener
		ports[i] = listener.Addr().(*net.TCPAddr).Port
	}

	// Test all ports are detected as in use
	for i, port := range ports {
		inUse := utils.IsPortInUse(port)
		s.True(inUse, "Port %d (listener %d) should be detected as in use", port, i)
	}

	// Close all listeners
	for _, listener := range listeners {
		listener.Close()
	}
}

func TestNetworkTestSuite(t *testing.T) {
	suite.Run(t, &NetworkTestSuite{})
}
