package milvus

import (
	"strconv"
	"strings"
)

func GetAddressPort(endpoint string) (string, int32) {
	s := strings.Split(endpoint, ":")
	if len(s) > 1 {
		port, err := strconv.Atoi(s[1])
		if err == nil {
			return s[0], int32(port)
		}
	}

	return s[0], 0
}
