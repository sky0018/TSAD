package env

import (
	"os"
	"strconv"
)

var tceDebugPort = ""

func init() {
	portStr := os.Getenv("TCE_DEBUG_PORT")
	port, err := strconv.Atoi(portStr)
	if portStr != "" && err == nil && port > 0 {
		tceDebugPort = portStr
	}
}

func TCEDebugPort() string {
	return tceDebugPort
}
