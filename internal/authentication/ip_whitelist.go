package authentication

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/JustusvonderBeek/shoppinglist-server/internal/util"
	"log"
)

var whitelistedIPs map[string]bool

func SetupWhitelistedIPs() error {
	loadedWhitelistedIPs, err := readIpWhitelistFromFile()
	if err != nil {
		return err
	}
	whitelistedIPs = loadedWhitelistedIPs
	return nil
}

func readIpWhitelistFromFile() (map[string]bool, error) {
	content, err := util.ReadFileFromRoot("resources/whitelisted_ips.json")
	if err != nil {
		return nil, err
	}
	var whitelistedIPs []string
	if err = json.Unmarshal(content, &whitelistedIPs); err != nil {
		return nil, err
	}
	log.Printf("Found IP Whitelist with %d IPs", len(whitelistedIPs))
	whiteIps := make(map[string]bool, len(whitelistedIPs))
	for _, ip := range whitelistedIPs {
		whiteIps[ip] = true
	}
	return whiteIps, nil
}

func IsSourceIPAllowedForAccess(ip string) error {
	if whitelistedIPs[ip] {
		return nil
	}
	formattedString := fmt.Sprintf("ip %s not whitelisted", ip)
	return errors.New(formattedString)
}
