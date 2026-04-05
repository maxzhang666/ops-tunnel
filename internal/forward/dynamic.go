package forward

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/maxzhang666/ops-tunnel/internal/config"
	gossh "golang.org/x/crypto/ssh"
)

// DynamicForwarder implements dynamic (-D) proxy forwarding with auto-detection
// of SOCKS5 and HTTP CONNECT protocols on the same port.
type DynamicForwarder struct {
	mapping    config.Mapping
	acl        *ACL
	listenAddr string
	logFn      LogFunc

	mu        sync.RWMutex
	listener  net.Listener
	state     string // "stopped" | "listening" | "error"
	lastErr   string
	done      chan struct{}
	sshClient *gossh.Client

	active    sync.WaitGroup
	activeCnt atomic.Int32
	totalCnt  atomic.Int64
	bytesIn   atomic.Int64
	bytesOut  atomic.Int64
}

func (f *DynamicForwarder) SetLogger(fn LogFunc) { f.logFn = fn }

func (f *DynamicForwarder) log(level, msg string) {
	if f.logFn != nil {
		f.logFn(level, msg)
	}
}

func NewDynamicForwarder(m config.Mapping) *DynamicForwarder {
	return &DynamicForwarder{
		mapping: m,
		state:   "stopped",
	}
}

func (f *DynamicForwarder) Status() Status {
	f.mu.RLock()
	defer f.mu.RUnlock()
	listen := f.listenAddr
	if listen == "" {
		listen = fmt.Sprintf("%s:%d", f.mapping.Listen.Host, f.mapping.Listen.Port)
	}
	return Status{
		MappingID:   f.mapping.ID,
		State:       f.state,
		Listen:      listen,
		ActiveConns: int(f.activeCnt.Load()),
		TotalConns:  f.totalCnt.Load(),
		BytesIn:     f.bytesIn.Load(),
		BytesOut:    f.bytesOut.Load(),
		LastError:   f.lastErr,
	}
}

func (f *DynamicForwarder) Start(ctx context.Context, sshClient *gossh.Client) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.state == "listening" {
		return fmt.Errorf("already listening")
	}

	var allowCIDRs, denyCIDRs []string
	if f.mapping.Socks5 != nil {
		allowCIDRs = f.mapping.Socks5.AllowCIDRs
		denyCIDRs = f.mapping.Socks5.DenyCIDRs
	}
	acl, err := NewACL(allowCIDRs, denyCIDRs)
	if err != nil {
		return fmt.Errorf("build ACL: %w", err)
	}
	f.acl = acl

	addr := fmt.Sprintf("%s:%d", f.mapping.Listen.Host, f.mapping.Listen.Port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", addr, err)
	}

	f.listener = ln
	f.listenAddr = ln.Addr().String()
	f.sshClient = sshClient
	f.state = "listening"
	f.lastErr = ""
	f.done = make(chan struct{})

	f.log("info", fmt.Sprintf("SOCKS5/HTTP proxy listening on %s", f.listenAddr))
	go f.acceptLoop()
	return nil
}

func (f *DynamicForwarder) acceptLoop() {
	defer close(f.done)
	for {
		conn, err := f.listener.Accept()
		if err != nil {
			return
		}
		f.active.Add(1)
		f.activeCnt.Add(1)
		go f.handleConn(conn)
	}
}

func (f *DynamicForwarder) handleConn(conn net.Conn) {
	defer func() {
		f.activeCnt.Add(-1)
		f.totalCnt.Add(1)
		f.active.Done()
	}()

	br := bufio.NewReader(conn)
	first, err := br.Peek(1)
	if err != nil {
		conn.Close()
		return
	}

	// Wrap conn so subsequent reads go through the buffered reader.
	bc := &bufferedConn{Conn: conn, r: br}

	if first[0] == socks5Version {
		f.handleSOCKS5(bc)
	} else {
		f.handleHTTP(bc)
	}
}

// bufferedConn wraps a net.Conn with a bufio.Reader for peeked bytes.
type bufferedConn struct {
	net.Conn
	r *bufio.Reader
}

func (c *bufferedConn) Read(b []byte) (int, error) { return c.r.Read(b) }

func (f *DynamicForwarder) handleSOCKS5(conn net.Conn) {
	if err := f.negotiate(conn); err != nil {
		f.log("warn", fmt.Sprintf("SOCKS5 handshake failed from %s: %s", conn.RemoteAddr(), err))
		conn.Close()
		return
	}

	req, err := readRequest(conn)
	if err != nil {
		f.log("warn", fmt.Sprintf("SOCKS5 request read failed from %s: %s", conn.RemoteAddr(), err))
		conn.Close()
		return
	}

	switch req.Cmd {
	case cmdConnect:
		f.handleConnect(conn, req)
	case cmdBind:
		f.handleBind(conn, req)
	default:
		writeReply(conn, RepCmdNotSupported, nil)
		conn.Close()
	}
}

func (f *DynamicForwarder) handleHTTP(conn net.Conn) {
	br := bufio.NewReader(conn)
	reqLine, err := br.ReadString('\n')
	if err != nil {
		conn.Close()
		return
	}

	// Parse "CONNECT host:port HTTP/1.x" or other methods
	parts := strings.Fields(strings.TrimSpace(reqLine))
	if len(parts) < 3 || !strings.HasPrefix(parts[2], "HTTP/") {
		io.WriteString(conn, "HTTP/1.1 400 Bad Request\r\n\r\n")
		conn.Close()
		return
	}

	// Consume remaining headers
	for {
		line, err := br.ReadString('\n')
		if err != nil || strings.TrimSpace(line) == "" {
			break
		}
	}

	method := strings.ToUpper(parts[0])
	if method != "CONNECT" {
		io.WriteString(conn, "HTTP/1.1 405 Method Not Allowed\r\n\r\n")
		conn.Close()
		return
	}

	target := parts[1]
	// Ensure host:port format
	if _, _, err := net.SplitHostPort(target); err != nil {
		target = net.JoinHostPort(target, "443")
	}

	host, _, _ := net.SplitHostPort(target)
	if ip := net.ParseIP(host); ip != nil && !f.acl.Check(ip) {
		io.WriteString(conn, "HTTP/1.1 403 Forbidden\r\n\r\n")
		conn.Close()
		f.mu.Lock()
		f.lastErr = fmt.Sprintf("ACL denied %s", host)
		f.mu.Unlock()
		return
	}

	if f.sshClient == nil {
		io.WriteString(conn, "HTTP/1.1 502 Bad Gateway\r\n\r\n")
		conn.Close()
		f.log("error", "HTTP CONNECT failed: no SSH client")
		return
	}

	f.log("info", fmt.Sprintf("HTTP CONNECT %s", target))

	remote, err := f.sshClient.Dial("tcp", target)
	if err != nil {
		io.WriteString(conn, "HTTP/1.1 502 Bad Gateway\r\n\r\n")
		conn.Close()
		errMsg := fmt.Sprintf("dial %s: %s", target, err)
		f.mu.Lock()
		f.lastErr = errMsg
		f.mu.Unlock()
		f.log("error", fmt.Sprintf("HTTP CONNECT %s failed: %s", target, err))
		return
	}

	io.WriteString(conn, "HTTP/1.1 200 Connection Established\r\n\r\n")
	biCopyCount(conn, remote, &f.bytesIn, &f.bytesOut)
}

func (f *DynamicForwarder) negotiate(conn net.Conn) error {
	methods, err := readMethodSelection(conn)
	if err != nil {
		return err
	}

	requiredMethod := byte(authNone)
	if f.mapping.Socks5 != nil && f.mapping.Socks5.Auth == config.Socks5UserPass {
		requiredMethod = authUserPass
	}

	offered := false
	for _, m := range methods {
		if m == requiredMethod {
			offered = true
			break
		}
	}
	if !offered {
		writeMethodChoice(conn, authNoAccept)
		return fmt.Errorf("required auth method %d not offered", requiredMethod)
	}

	writeMethodChoice(conn, requiredMethod)

	if requiredMethod == authUserPass {
		user, pass, err := readUsernamePassword(conn)
		if err != nil {
			return fmt.Errorf("read userpass: %w", err)
		}
		if user != f.mapping.Socks5.Username || pass != f.mapping.Socks5.Password {
			writeAuthResult(conn, false)
			return fmt.Errorf("auth failed for user %q", user)
		}
		writeAuthResult(conn, true)
	}

	return nil
}

func (f *DynamicForwarder) handleConnect(conn net.Conn, req *Request) {
	if req.AddrType == atypIPv4 || req.AddrType == atypIPv6 {
		ip := net.ParseIP(req.Addr)
		if ip != nil && !f.acl.Check(ip) {
			writeReply(conn, RepNotAllowed, nil)
			conn.Close()
			f.mu.Lock()
			f.lastErr = fmt.Sprintf("ACL denied %s", req.Addr)
			f.mu.Unlock()
			return
		}
	}

	if f.sshClient == nil {
		writeReply(conn, RepGeneralFailure, nil)
		conn.Close()
		f.mu.Lock()
		f.lastErr = "no SSH client"
		f.mu.Unlock()
		f.log("error", "CONNECT failed: no SSH client")
		return
	}

	target := req.Target()
	f.log("info", fmt.Sprintf("CONNECT %s", target))

	remote, err := f.sshClient.Dial("tcp", target)
	if err != nil {
		writeReply(conn, RepHostUnreachable, nil)
		conn.Close()
		errMsg := fmt.Sprintf("dial %s: %s", target, err)
		f.mu.Lock()
		f.lastErr = errMsg
		f.mu.Unlock()
		f.log("error", fmt.Sprintf("CONNECT %s failed: %s", target, err))
		return
	}

	writeReply(conn, RepSuccess, remote.RemoteAddr())
	biCopyCount(conn, remote, &f.bytesIn, &f.bytesOut)
}

func (f *DynamicForwarder) handleBind(conn net.Conn, req *Request) {
	if f.sshClient == nil {
		writeReply(conn, RepGeneralFailure, nil)
		conn.Close()
		return
	}

	ln, err := f.sshClient.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		writeReply(conn, RepGeneralFailure, nil)
		conn.Close()
		f.mu.Lock()
		f.lastErr = fmt.Sprintf("remote listen: %s", err)
		f.mu.Unlock()
		return
	}

	writeReply(conn, RepSuccess, ln.Addr())

	type acceptResult struct {
		conn net.Conn
		err  error
	}
	ch := make(chan acceptResult, 1)
	go func() {
		c, err := ln.Accept()
		ch <- acceptResult{c, err}
	}()

	var inbound net.Conn
	select {
	case res := <-ch:
		ln.Close()
		if res.err != nil {
			writeReply(conn, RepGeneralFailure, nil)
			conn.Close()
			return
		}
		inbound = res.conn
	case <-time.After(60 * time.Second):
		ln.Close()
		writeReply(conn, RepGeneralFailure, nil)
		conn.Close()
		return
	}

	writeReply(conn, RepSuccess, inbound.RemoteAddr())
	biCopy(conn, inbound)
}

func (f *DynamicForwarder) Stop(ctx context.Context) error {
	f.mu.Lock()

	if f.state != "listening" {
		f.mu.Unlock()
		return nil
	}

	f.listener.Close()
	f.mu.Unlock()

	select {
	case <-f.done:
	case <-ctx.Done():
	}

	waitDone := make(chan struct{})
	go func() {
		f.active.Wait()
		close(waitDone)
	}()

	select {
	case <-waitDone:
	case <-ctx.Done():
	}

	f.mu.Lock()
	f.state = "stopped"
	f.mu.Unlock()
	return nil
}

var _ Forwarder = (*DynamicForwarder)(nil)
