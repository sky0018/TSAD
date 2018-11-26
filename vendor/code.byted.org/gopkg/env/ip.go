package env

import (
	"os"

	"code.byted.org/gopkg/net2"
)

var inTCE bool
var tceAddr string
var hostIP string

func init() {
	if os.Getenv("IS_TCE_DOCKER_ENV") == "1" {
		inTCE = true
		tceAddr = os.Getenv("HOST_IP_ADDR")
		if tceAddr == "" {
			tceAddr = net2.GetLocalIp()
		}
	}
	hostIP = net2.GetLocalIp()
}

// HostIP .
func HostIP() string {
	if inTCE {
		return tceAddr
	}

	return hostIP
}
