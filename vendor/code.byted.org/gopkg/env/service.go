package env

import "os"

const (
	PSMUnknown     = "-"
	ClusterDefault = "default"
)

var psm string
var cluster string

func init() {
	psm = os.Getenv("LOAD_SERVICE_PSM")
	if psm == "" {
		psm = PSMUnknown
	}

	cluster = os.Getenv("SERVICE_CLUSTER")
	if cluster == "" {
		cluster = ClusterDefault
	}
}

// PSM .
func PSM() string {
	return psm
}

// Cluster .
func Cluster() string {
	return cluster
}
