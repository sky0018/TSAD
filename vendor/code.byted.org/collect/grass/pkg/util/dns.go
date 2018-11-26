package util

import (
	"net"
)

func ParseDns(domain string) ([]string, error) {
	ips, err := net.LookupIP(domain)
	if err != nil {
		return nil, err
	}

	var strIps []string

	for _, ip := range ips {
		strIps = append(strIps, ip.String())
	}

	return strIps, nil
}
