package config

import "strings"

func GetSailSan(discoveryAddress string) string {
	discHost := strings.Split(discoveryAddress, ":")[0]
	// For local debugging - the discoveryAddress is set to localhost, but the cert issued for normal SA.
	if discHost == "localhost" {
		discHost = "dubbod.dubbo-system.svc"
	}
	return discHost
}
