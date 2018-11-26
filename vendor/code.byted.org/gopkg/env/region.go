package env

import "sync/atomic"

const (
	// UnknownRegion .
	UnknownRegion = "-"
	R_CN          = "CN"
	R_SG          = "SG"
	R_US          = "US"
	R_ALISG       = "ALISG" // Singapore Aliyun
	R_CA          = "CA"    // West America
)

var (
	region     atomic.Value
	regionIDCs = map[string][]string{
		R_CN:    []string{DC_HY, DC_LF, DC_HL},
		R_SG:    []string{DC_SG},
		R_US:    []string{DC_VA},
		R_CA:    []string{DC_CA},
		R_ALISG: []string{DC_ALISG},
	}
)

// Region .
func Region() string {
	if v := region.Load(); v != nil {
		return v.(string)
	}

	idc := IDC()
	for r, idcs := range regionIDCs {
		for _, dc := range idcs {
			if idc == dc {
				region.Store(r)
				return r
			}
		}
	}

	region.Store(UnknownRegion)
	return UnknownRegion
}
