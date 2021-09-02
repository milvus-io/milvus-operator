package milvus

import (
	"net"
	neturl "net/url"
	"strconv"
)

func GetHostPortFromURL(url string) (string, int32) {
	u, err := neturl.ParseRequestURI(url)
	if err != nil {
		return url, 80
	}

	portInt, err := strconv.Atoi(u.Port())
	if err != nil {
		return u.Hostname(), 80
	}

	return u.Hostname(), int32(portInt)
}

func GetHostPort(endpoint string) (string, int32) {
	host, port, err := net.SplitHostPort(endpoint)
	if err != nil {
		return endpoint, 80
	}

	portInt, err := strconv.Atoi(port)
	if err != nil {
		return host, 80
	}

	return host, int32(portInt)
}
