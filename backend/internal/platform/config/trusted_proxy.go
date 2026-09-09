package config

import (
	"errors"
	"net"
	"strings"
)

// ValidateTrustedProxyCIDRs ensures every configured CIDR is parseable.
func ValidateTrustedProxyCIDRs(raw string) error {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	for part := range strings.SplitSeq(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if !strings.Contains(part, "/") {
			part += "/32"
		}
		if _, _, err := net.ParseCIDR(part); err != nil {
			return errors.New("TRUSTED_PROXY_CIDRS contains an invalid CIDR")
		}
	}
	return nil
}
