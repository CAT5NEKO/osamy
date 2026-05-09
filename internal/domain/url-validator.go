package domain

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

const MaxUrlLength = 2048

var privateNetworks []*net.IPNet

func init() {
	privateCidrs := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"127.0.0.0/8",
		"169.254.0.0/16",
		"100.64.0.0/10",
		"0.0.0.0/8",
		"::1/128",
		"fc00::/7",
		"fe80::/10",
		"ff00::/8",
	}

	for _, cidr := range privateCidrs {
		_, network, _ := net.ParseCIDR(cidr)
		privateNetworks = append(privateNetworks, network)
	}
}

func ValidateTargetUrl(targetUrl string) error {
	if len(targetUrl) > MaxUrlLength {
		return fmt.Errorf("url exceeds maximum length of %d characters", MaxUrlLength)
	}

	parsedUrl, parseError := url.Parse(targetUrl)
	if parseError != nil {
		return fmt.Errorf("invalid url format")
	}

	scheme := strings.ToLower(parsedUrl.Scheme)
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("unsupported url scheme")
	}

	hostname := parsedUrl.Hostname()
	if hostname == "" {
		return fmt.Errorf("empty hostname")
	}

	if IsPrivateHostname(hostname) {
		return fmt.Errorf("access to private addresses is not allowed")
	}

	return nil
}

func IsPrivateHostname(hostname string) bool {
	if hostname == "localhost" || strings.HasSuffix(hostname, ".local") {
		return true
	}

	parsedIp := net.ParseIP(hostname)
	if parsedIp == nil {
		return false
	}

	return IsPrivateIp(parsedIp)
}

func IsPrivateIp(ipAddress net.IP) bool {
	for _, network := range privateNetworks {
		if network.Contains(ipAddress) {
			return true
		}
	}
	return false
}
