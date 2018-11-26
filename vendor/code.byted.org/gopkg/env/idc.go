package env

import (
	"os"
	"os/exec"
	"strings"
	"sync/atomic"
)

const (
	UnknownIDC = "-"
	DC_HY      = "hy"
	DC_LF      = "lf"
	DC_HL      = "hl"
	DC_VA      = "va"
	DC_SG      = "sg"
	DC_CA      = "ca"    // West America
	DC_ALISG   = "alisg" // Singapore Aliyun
	DC_ALIVA   = "aliva"
	DC_MALIVA  = "maliva"
	DC_ALINC2  = "alinc2" //aliyun north
)

var (
	idc       atomic.Value
	idcPrefix = map[string][]string{
		DC_HY:     []string{"10.4."},
		DC_LF:     []string{"10.2.", "10.3.", "10.6.", "10.8.", "10.9.", "10.10.", "10.11.", "10.12.", "10.13.", "10.14."},
		DC_HL:     []string{},
		DC_VA:     []string{"10.100."},
		DC_SG:     []string{"10.101."},
		DC_CA:     []string{"10.106."},
		DC_ALISG:  []string{"10.115."},
		DC_ALIVA:  []string{},
		DC_MALIVA: []string{},
		DC_ALINC2: []string{},
	}
	FixedIDCList = []string{ // NOTE: new added idc must be append to the end
		UnknownIDC, DC_HY, DC_LF, DC_HL, DC_VA, DC_SG, DC_CA, DC_ALISG, DC_ALIVA, DC_MALIVA, DC_ALINC2,
	}
)

// IDC .
func IDC() string {
	if v := idc.Load(); v != nil {
		return v.(string)
	}

	if dc := os.Getenv("RUNTIME_IDC_NAME"); dc != "" {
		idc.Store(dc)
		return dc
	}

	ip := HostIP()
	for idcStr, pres := range idcPrefix {
		for _, p := range pres {
			if strings.HasPrefix(ip, p) {
				idc.Store(idcStr)
				return idcStr
			}
		}
	}

	cmd0 := exec.Command("/opt/tiger/consul_deploy/bin/determine_dc.sh")
	output0, err := cmd0.Output()
	if err == nil {
		dc := strings.TrimSpace(string(output0))
		if _, ok := idcPrefix[dc]; ok {
			idc.Store(dc)
			return dc
		}
	}

	cmd := exec.Command(`bash`, `-c`, `sd report|grep "Data center"|awk '{print $3}'`)
	output, err := cmd.Output()
	if err == nil {
		dc := strings.TrimSpace(string(output))
		if _, ok := idcPrefix[dc]; ok {
			idc.Store(dc)
			return dc
		}
	}

	idc.Store(UnknownIDC)
	return UnknownIDC
}

func GetIDCList() []string {
	idcList := []string{}
	for key, _ := range idcPrefix {
		idcList = append(idcList, key)
	}
	return idcList
}
