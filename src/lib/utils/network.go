package utils

import (
	"fmt"
	"net"
)

// IsPortInUse checks if the given port is currently in use by any process.
func IsPortInUse(port int) bool {
	conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", port))

	if err != nil {
		// Port is not in use
		return false
	}

	if conn != nil {
		conn.Close()
	}

	return true
}
