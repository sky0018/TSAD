/*
   package util pack some useful tools.
*/

package util

import (
	"net"
	"time"
)

// LocalIP return IP
func LocalIP() (string, error) {
	inters, err := net.InterfaceAddrs()
	if err != nil {
		return "-", err
	}
	for _, inter := range inters {
		if ipnet, ok := inter.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ip := ipnet.IP.String()
				return ip, nil
			}
		}
	}
	return "-", nil
}

// UTCTime return string of utc time
func UTCTime() string {
	now := time.Now()
	utc, _ := time.LoadLocation("")
	now = now.In(utc)
	return now.Format(time.RFC3339)
}

// OUTCTime return time
func OUTCTime(utc string) time.Time {
	t, _ := time.ParseInLocation(time.RFC3339, utc, time.UTC)
	return t
}
