package net2

import (
	"net"
	"strconv"
	"strings"
	"sync/atomic"
)

const UnknownIPAddr = "-"

var localIP atomic.Value

func GetLocalIp() string {
	if v := localIP.Load(); v != nil {
		return v.(string)
	}
	ip := GetLocalIpNoCache()
	localIP.Store(ip)
	return ip
}

// use carefully
func GetLocalIpNoCache() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return UnknownIPAddr
	}

	for _, addr := range addrs {
		ipnet, ok := addr.(*net.IPNet)
		if !ok {
			continue
		}
		if ipnet.IP.IsLoopback() {
			continue
		}
		ip := ipnet.IP.String()
		if IsPrivateIp(ip) {
			return ip
		}
	}
	return UnknownIPAddr
}

func IsPrivateIp(s string) bool {
	if strings.HasPrefix(s, "10.") {
		return true
	}
	if strings.HasPrefix(s, "192.168.") {
		return true
	}
	if !strings.HasPrefix(s, "172.") {
		return false
	}
	for i := 16; i <= 31; i++ {
		if strings.HasPrefix(s, "172."+strconv.Itoa(i)+".") {
			return true
		}
	}
	return false
}
