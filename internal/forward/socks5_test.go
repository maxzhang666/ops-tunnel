package forward

import (
	"bytes"
	"encoding/binary"
	"net"
	"testing"
)

func TestReadMethodSelection(t *testing.T) {
	buf := bytes.NewBuffer([]byte{0x05, 0x02, 0x00, 0x02})
	methods, err := readMethodSelection(buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(methods) != 2 || methods[0] != 0x00 || methods[1] != 0x02 {
		t.Errorf("methods = %v, want [0x00, 0x02]", methods)
	}
}

func TestReadMethodSelection_BadVersion(t *testing.T) {
	buf := bytes.NewBuffer([]byte{0x04, 0x01, 0x00})
	_, err := readMethodSelection(buf)
	if err == nil {
		t.Error("expected error for SOCKS4")
	}
}

func TestWriteMethodChoice(t *testing.T) {
	var buf bytes.Buffer
	err := writeMethodChoice(&buf, 0x02)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(buf.Bytes(), []byte{0x05, 0x02}) {
		t.Errorf("got %x, want 0502", buf.Bytes())
	}
}

func TestReadUsernamePassword(t *testing.T) {
	data := []byte{0x01, 0x04, 'u', 's', 'e', 'r', 0x04, 'p', 'a', 's', 's'}
	user, pass, err := readUsernamePassword(bytes.NewBuffer(data))
	if err != nil {
		t.Fatal(err)
	}
	if user != "user" || pass != "pass" {
		t.Errorf("got user=%q pass=%q, want user/pass", user, pass)
	}
}

func TestWriteAuthResult(t *testing.T) {
	var buf bytes.Buffer
	writeAuthResult(&buf, true)
	if !bytes.Equal(buf.Bytes(), []byte{0x01, 0x00}) {
		t.Errorf("success: got %x, want 0100", buf.Bytes())
	}

	buf.Reset()
	writeAuthResult(&buf, false)
	if !bytes.Equal(buf.Bytes(), []byte{0x01, 0x01}) {
		t.Errorf("failure: got %x, want 0101", buf.Bytes())
	}
}

func TestReadRequest_ConnectIPv4(t *testing.T) {
	data := []byte{0x05, 0x01, 0x00, 0x01, 10, 0, 0, 1, 0x1F, 0x90}
	req, err := readRequest(bytes.NewBuffer(data))
	if err != nil {
		t.Fatal(err)
	}
	if req.Cmd != cmdConnect {
		t.Errorf("Cmd = %d, want %d", req.Cmd, cmdConnect)
	}
	if req.Addr != "10.0.0.1" {
		t.Errorf("Addr = %q, want %q", req.Addr, "10.0.0.1")
	}
	if req.Port != 8080 {
		t.Errorf("Port = %d, want 8080", req.Port)
	}
	if req.Target() != "10.0.0.1:8080" {
		t.Errorf("Target = %q, want %q", req.Target(), "10.0.0.1:8080")
	}
}

func TestReadRequest_ConnectDomain(t *testing.T) {
	domain := "example.com"
	data := []byte{0x05, 0x01, 0x00, 0x03, byte(len(domain))}
	data = append(data, []byte(domain)...)
	data = append(data, 0x00, 0x50)
	req, err := readRequest(bytes.NewBuffer(data))
	if err != nil {
		t.Fatal(err)
	}
	if req.AddrType != atypDomain {
		t.Errorf("AddrType = %d, want %d", req.AddrType, atypDomain)
	}
	if req.Addr != "example.com" {
		t.Errorf("Addr = %q, want %q", req.Addr, "example.com")
	}
	if req.Port != 80 {
		t.Errorf("Port = %d, want 80", req.Port)
	}
}

func TestReadRequest_ConnectIPv6(t *testing.T) {
	ip := net.ParseIP("::1")
	data := []byte{0x05, 0x01, 0x00, 0x04}
	data = append(data, ip.To16()...)
	data = append(data, 0x01, 0xBB)
	req, err := readRequest(bytes.NewBuffer(data))
	if err != nil {
		t.Fatal(err)
	}
	if req.AddrType != atypIPv6 {
		t.Errorf("AddrType = %d, want %d", req.AddrType, atypIPv6)
	}
	if req.Addr != "::1" {
		t.Errorf("Addr = %q, want %q", req.Addr, "::1")
	}
	if req.Port != 443 {
		t.Errorf("Port = %d, want 443", req.Port)
	}
}

func TestReadRequest_Bind(t *testing.T) {
	data := []byte{0x05, 0x02, 0x00, 0x01, 0, 0, 0, 0, 0x00, 0x00}
	req, err := readRequest(bytes.NewBuffer(data))
	if err != nil {
		t.Fatal(err)
	}
	if req.Cmd != cmdBind {
		t.Errorf("Cmd = %d, want %d", req.Cmd, cmdBind)
	}
}

func TestWriteReply_Success(t *testing.T) {
	var buf bytes.Buffer
	addr := &net.TCPAddr{IP: net.IPv4(192, 168, 1, 1), Port: 9090}
	err := writeReply(&buf, RepSuccess, addr)
	if err != nil {
		t.Fatal(err)
	}
	b := buf.Bytes()
	if b[0] != 0x05 || b[1] != RepSuccess || b[2] != 0x00 || b[3] != atypIPv4 {
		t.Errorf("header = %x, want 05 00 00 01", b[:4])
	}
	if !bytes.Equal(b[4:8], []byte{192, 168, 1, 1}) {
		t.Errorf("addr = %v, want 192.168.1.1", b[4:8])
	}
	port := binary.BigEndian.Uint16(b[8:10])
	if port != 9090 {
		t.Errorf("port = %d, want 9090", port)
	}
}

func TestWriteReply_NilAddr(t *testing.T) {
	var buf bytes.Buffer
	err := writeReply(&buf, RepGeneralFailure, nil)
	if err != nil {
		t.Fatal(err)
	}
	b := buf.Bytes()
	if b[1] != RepGeneralFailure {
		t.Errorf("rep = %d, want %d", b[1], RepGeneralFailure)
	}
	if b[3] != atypIPv4 {
		t.Errorf("atyp = %d, want %d", b[3], atypIPv4)
	}
}

func TestWriteReply_IPv6(t *testing.T) {
	var buf bytes.Buffer
	addr := &net.TCPAddr{IP: net.ParseIP("::1"), Port: 443}
	err := writeReply(&buf, RepSuccess, addr)
	if err != nil {
		t.Fatal(err)
	}
	b := buf.Bytes()
	if b[3] != atypIPv6 {
		t.Errorf("atyp = %d, want %d", b[3], atypIPv6)
	}
	if len(b) != 22 {
		t.Errorf("len = %d, want 22", len(b))
	}
}
