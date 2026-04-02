package forward

import (
	"fmt"
	"net"
)

// ACL implements CIDR-based access control.
type ACL struct {
	deny  []*net.IPNet
	allow []*net.IPNet
}

// NewACL creates an ACL from allow/deny CIDR lists.
func NewACL(allowCIDRs, denyCIDRs []string) (*ACL, error) {
	a := &ACL{}
	for _, cidr := range denyCIDRs {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			return nil, fmt.Errorf("parse deny CIDR %q: %w", cidr, err)
		}
		a.deny = append(a.deny, ipNet)
	}
	for _, cidr := range allowCIDRs {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			return nil, fmt.Errorf("parse allow CIDR %q: %w", cidr, err)
		}
		a.allow = append(a.allow, ipNet)
	}
	return a, nil
}

// Check returns true if the IP is allowed.
func (a *ACL) Check(ip net.IP) bool {
	if len(a.deny) == 0 && len(a.allow) == 0 {
		return true
	}
	for _, n := range a.deny {
		if n.Contains(ip) {
			return false
		}
	}
	for _, n := range a.allow {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}
