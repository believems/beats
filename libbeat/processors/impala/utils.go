package impala

import (
	"net"
	"os"
)

func GetLocalIP() string {
	hostname, err := os.Hostname()
	if err != nil {
		return ""
	}
	ips, err := net.LookupIP(hostname)
	if err != nil {
		return ""
	}
	for _, ip := range ips {
		if !ip.IsLoopback() && ip.To4() != nil {
			return ip.String()
		}
	}
	return ""
}
