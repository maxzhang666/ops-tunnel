package forward

import (
	"net"
	"testing"
)

func TestACL_AllowOnly(t *testing.T) {
	acl, err := NewACL([]string{"10.0.0.0/8", "192.168.0.0/16"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !acl.Check(net.ParseIP("10.0.0.1")) {
		t.Error("10.0.0.1 should be allowed")
	}
	if !acl.Check(net.ParseIP("192.168.1.1")) {
		t.Error("192.168.1.1 should be allowed")
	}
	if acl.Check(net.ParseIP("8.8.8.8")) {
		t.Error("8.8.8.8 should be rejected")
	}
}

func TestACL_DenyOverridesAllow(t *testing.T) {
	acl, err := NewACL([]string{"10.0.0.0/8"}, []string{"10.0.0.1/32"})
	if err != nil {
		t.Fatal(err)
	}
	if acl.Check(net.ParseIP("10.0.0.1")) {
		t.Error("10.0.0.1 should be denied")
	}
	if !acl.Check(net.ParseIP("10.0.0.2")) {
		t.Error("10.0.0.2 should be allowed")
	}
}

func TestACL_EmptyAllowAll(t *testing.T) {
	acl, err := NewACL(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !acl.Check(net.ParseIP("1.2.3.4")) {
		t.Error("empty ACL should allow all")
	}
	if !acl.Check(net.ParseIP("::1")) {
		t.Error("empty ACL should allow IPv6 too")
	}
}

func TestACL_DefaultReject(t *testing.T) {
	acl, err := NewACL([]string{"10.0.0.0/8"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if acl.Check(net.ParseIP("172.16.0.1")) {
		t.Error("172.16.0.1 not in allow list should be rejected")
	}
}

func TestACL_InvalidCIDR(t *testing.T) {
	_, err := NewACL([]string{"not-a-cidr"}, nil)
	if err == nil {
		t.Error("expected error for invalid CIDR")
	}
}

func TestACL_IPv6(t *testing.T) {
	acl, err := NewACL([]string{"fd00::/8"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !acl.Check(net.ParseIP("fd00::1")) {
		t.Error("fd00::1 should be allowed")
	}
	if acl.Check(net.ParseIP("2001:db8::1")) {
		t.Error("2001:db8::1 should be rejected")
	}
}
