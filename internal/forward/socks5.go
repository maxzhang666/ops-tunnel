package forward

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
)

// SOCKS5 protocol constants
const (
	socks5Version = 0x05

	authNone     = 0x00
	authUserPass = 0x02
	authNoAccept = 0xFF

	cmdConnect      = 0x01
	cmdBind         = 0x02
	cmdUDPAssociate = 0x03

	atypIPv4   = 0x01
	atypDomain = 0x03
	atypIPv6   = 0x04

	RepSuccess          = 0x00
	RepGeneralFailure   = 0x01
	RepNotAllowed       = 0x02
	RepNetUnreachable   = 0x03
	RepHostUnreachable  = 0x04
	RepConnRefused      = 0x05
	RepCmdNotSupported  = 0x07
	RepAddrNotSupported = 0x08
)

// readMethodSelection reads the SOCKS5 version and client's offered auth methods.
func readMethodSelection(r io.Reader) ([]byte, error) {
	header := make([]byte, 2)
	if _, err := io.ReadFull(r, header); err != nil {
		return nil, fmt.Errorf("read method header: %w", err)
	}
	if header[0] != socks5Version {
		return nil, fmt.Errorf("unsupported SOCKS version: %d", header[0])
	}
	nMethods := int(header[1])
	if nMethods == 0 {
		return nil, fmt.Errorf("no auth methods offered")
	}
	methods := make([]byte, nMethods)
	if _, err := io.ReadFull(r, methods); err != nil {
		return nil, fmt.Errorf("read methods: %w", err)
	}
	return methods, nil
}

// writeMethodChoice writes the server's chosen auth method.
func writeMethodChoice(w io.Writer, method byte) error {
	_, err := w.Write([]byte{socks5Version, method})
	return err
}

// readUsernamePassword reads RFC 1929 username/password sub-negotiation.
func readUsernamePassword(r io.Reader) (string, string, error) {
	ver := make([]byte, 1)
	if _, err := io.ReadFull(r, ver); err != nil {
		return "", "", err
	}
	if ver[0] != 0x01 {
		return "", "", fmt.Errorf("unsupported auth version: %d", ver[0])
	}

	uLen := make([]byte, 1)
	if _, err := io.ReadFull(r, uLen); err != nil {
		return "", "", err
	}
	uName := make([]byte, uLen[0])
	if _, err := io.ReadFull(r, uName); err != nil {
		return "", "", err
	}

	pLen := make([]byte, 1)
	if _, err := io.ReadFull(r, pLen); err != nil {
		return "", "", err
	}
	passwd := make([]byte, pLen[0])
	if _, err := io.ReadFull(r, passwd); err != nil {
		return "", "", err
	}

	return string(uName), string(passwd), nil
}

// writeAuthResult writes RFC 1929 auth result.
func writeAuthResult(w io.Writer, success bool) error {
	status := byte(0x01)
	if success {
		status = 0x00
	}
	_, err := w.Write([]byte{0x01, status})
	return err
}

// Request represents a parsed SOCKS5 request.
type Request struct {
	Cmd      byte
	AddrType byte
	Addr     string
	Port     uint16
}

// Target returns "host:port" suitable for net.Dial.
func (r *Request) Target() string {
	return net.JoinHostPort(r.Addr, fmt.Sprintf("%d", r.Port))
}

// readRequest reads a SOCKS5 request from the client.
func readRequest(r io.Reader) (*Request, error) {
	header := make([]byte, 4)
	if _, err := io.ReadFull(r, header); err != nil {
		return nil, fmt.Errorf("read request header: %w", err)
	}
	if header[0] != socks5Version {
		return nil, fmt.Errorf("unsupported version: %d", header[0])
	}

	req := &Request{
		Cmd:      header[1],
		AddrType: header[3],
	}

	switch req.AddrType {
	case atypIPv4:
		addr := make([]byte, 4)
		if _, err := io.ReadFull(r, addr); err != nil {
			return nil, fmt.Errorf("read IPv4 addr: %w", err)
		}
		req.Addr = net.IP(addr).String()
	case atypDomain:
		dLen := make([]byte, 1)
		if _, err := io.ReadFull(r, dLen); err != nil {
			return nil, fmt.Errorf("read domain length: %w", err)
		}
		domain := make([]byte, dLen[0])
		if _, err := io.ReadFull(r, domain); err != nil {
			return nil, fmt.Errorf("read domain: %w", err)
		}
		req.Addr = string(domain)
	case atypIPv6:
		addr := make([]byte, 16)
		if _, err := io.ReadFull(r, addr); err != nil {
			return nil, fmt.Errorf("read IPv6 addr: %w", err)
		}
		req.Addr = net.IP(addr).String()
	default:
		return nil, fmt.Errorf("unsupported address type: %d", req.AddrType)
	}

	portBuf := make([]byte, 2)
	if _, err := io.ReadFull(r, portBuf); err != nil {
		return nil, fmt.Errorf("read port: %w", err)
	}
	req.Port = binary.BigEndian.Uint16(portBuf)

	return req, nil
}

// writeReply writes a SOCKS5 reply. If bindAddr is nil, uses 0.0.0.0:0.
func writeReply(w io.Writer, rep byte, bindAddr net.Addr) error {
	var ip net.IP
	var port int

	if tcpAddr, ok := bindAddr.(*net.TCPAddr); ok && tcpAddr != nil {
		ip = tcpAddr.IP
		port = tcpAddr.Port
	}

	if ip == nil {
		ip = net.IPv4zero
	}

	reply := []byte{socks5Version, rep, 0x00}

	if ip4 := ip.To4(); ip4 != nil {
		reply = append(reply, atypIPv4)
		reply = append(reply, ip4...)
	} else {
		reply = append(reply, atypIPv6)
		reply = append(reply, ip.To16()...)
	}

	portBuf := make([]byte, 2)
	binary.BigEndian.PutUint16(portBuf, uint16(port))
	reply = append(reply, portBuf...)

	_, err := w.Write(reply)
	return err
}
