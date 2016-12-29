package supernova

import (
	"bufio"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// HandshakeError describes an error with the handshake from the peer.

type HandshakeError struct {
	message string
}

func (e HandshakeError) Error() string { return e.message }

type Upgrader struct {

	// HandshakeTimeout specifies the duration for the handshake to complete.
	HandshakeTimeout time.Duration

	// ReadBufferSize and WriteBufferSize specify I/O buffer sizes. If a buffer
	// size is zero, then a default value of 4096 is used. The I/O buffer sizes
	// do not limit the size of the messages that can be sent or received.
	ReadBufferSize, WriteBufferSize int

	// Subprotocols specifies the server's supported protocols in order of
	// preference. If this field is set, then the Upgrade method negotiates a
	// subprotocol by selecting the first match in this list with a protocol
	// requested by the client.
	Subprotocols []string

	// Error specifies the function for generating HTTP error responses. If Error
	// is nil, then http.Error is used to generate the HTTP response.
	Error func(req *Request, status int, reason error)

	// CheckOrigin returns true if the request Origin header is acceptable. If
	// CheckOrigin is nil, the host in the Origin header must not be set or
	// must match the host of the request.
	CheckOrigin func(ctx *Request) bool

	// EnableCompression specify if the server should attempt to negotiate per
	// message compression (RFC 7692). Setting this value to true does not
	// guarantee that compression will be supported. Currently only "no context
	// takeover" modes are supported.
	EnableCompression bool
}

func (u *Upgrader) returnError(req *Request, status int, reason string) (*Conn, error) {

	err := HandshakeError{reason}

	if u.Error != nil {
		u.Error(req, status, err)

	} else {

		req.Response.Header.Add("Sec-Websocket-Version", "13")
		req.Response.Header.Add("Content-Type", "text/plain; charset=utf-8")
		req.Response.Header.Add("X-Content-Type-Options", "nosniff")

		req.Response.SetStatusCode(status)
		req.Write([]byte(http.StatusText(status)))

	}

	return nil, err

}

// checkSameOrigin returns true if the origin is not set or is equal to the request host.

func checkSameOrigin(r *Request) bool {

	origin := string(r.Request.Header.Peek("Origin"))

	if len(origin) == 0 {

		return true

	}

	u, err := url.Parse(origin)

	if err != nil {

		return false

	}

	return u.Host == string(r.Host())

}

func (u *Upgrader) selectSubprotocol(r *Request) string {

	if u.Subprotocols != nil {
		clientProtocols := Subprotocols(r)
		for _, serverProtocol := range u.Subprotocols {
			for _, clientProtocol := range clientProtocols {
				if clientProtocol == serverProtocol {
					return clientProtocol
				}
			}
		}
	}

	return ""
}

func (u *Upgrader) Upgrade(req *Request, responseHeader http.Header) (*Conn, error) {
	if req.GetMethod() != "GET" {
		return u.returnError(req, http.StatusMethodNotAllowed, "websocket: method not GET")
	}

	println(string(req.Request.Header.Peek("Sec-Websocket-Extensions")))
	if string(req.Request.Header.Peek("Sec-Websocket-Extensions")) != "" {
		return u.returnError(req, http.StatusInternalServerError, "websocket: application specific Sec-Websocket-Extensions headers are unsupported")
	}

	if !tokenListContainsValue(req.Request.Header, "Sec-Websocket-Version", "13") {
		return u.returnError(req, http.StatusBadRequest, "websocket: version != 13")
	}

	if !tokenListContainsValue(req.Request.Header, "Connection", "upgrade") {
		return u.returnError(req, http.StatusBadRequest, "websocket: could not find connection header with token 'upgrade'")
	}

	if !tokenListContainsValue(req.Request.Header, "Upgrade", "websocket") {
		return u.returnError(req, http.StatusBadRequest, "websocket: could not find upgrade header with token 'websocket'")
	}

	checkOrigin := u.CheckOrigin

	if checkOrigin == nil {
		checkOrigin = checkSameOrigin
	}

	if !checkOrigin(req) {
		return u.returnError(req, http.StatusForbidden, "websocket: origin not allowed")
	}

	challengeKey := string(req.Request.Header.Peek("Sec-Websocket-Key"))

	if challengeKey == "" {
		return u.returnError(req, http.StatusBadRequest, "websocket: key missing or blank")
	}

	subprotocol := u.selectSubprotocol(req)

	// Negotiate PMCE
	var compress bool

	if u.EnableCompression {
		for _, ext := range parseExtensions(req) {
			if ext[""] != "permessage-deflate" {
				continue
			}

			compress = true
			break
		}
	}

	var (
		netConn net.Conn
		br      *bufio.Reader
		err     error
	)

	h, ok := req.Ctx.(http.Hijacker)
	if !ok {
		return u.returnError(req, http.StatusInternalServerError, "websocket: response does not implement http.Hijacker")
	}

	var rw *bufio.ReadWriter

	netConn, rw, err = h.Hijack()

	if err != nil {
		return u.returnError(req, http.StatusInternalServerError, err.Error())
	}

	br = rw.Reader

	if br.Buffered() > 0 {
		netConn.Close()
		return nil, errors.New("websocket: client sent data before handshake is complete")
	}

	c := newConn(netConn, true, u.ReadBufferSize, u.WriteBufferSize)

	c.subprotocol = subprotocol

	if compress {
		c.newCompressionWriter = compressNoContextTakeover
		c.newDecompressionReader = decompressNoContextTakeover
	}

	p := c.writeBuf[:0]
	p = append(p, "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: "...)
	p = append(p, computeAcceptKey(challengeKey)...)
	p = append(p, "\r\n"...)

	if c.subprotocol != "" {
		p = append(p, "Sec-Websocket-Protocol: "...)
		p = append(p, c.subprotocol...)
		p = append(p, "\r\n"...)
	}

	if compress {
		p = append(p, "Sec-Websocket-Extensions: permessage-deflate; server_no_context_takeover; client_no_context_takeover\r\n"...)
	}

	for k, vs := range responseHeader {
		if k == "Sec-Websocket-Protocol" {
			continue
		}

		for _, v := range vs {
			p = append(p, k...)
			p = append(p, ": "...)
			for i := 0; i < len(v); i++ {
				b := v[i]
				if b <= 31 {
					// prevent response splitting.
					b = ' '
				}
				p = append(p, b)
			}
			p = append(p, "\r\n"...)
		}
	}

	p = append(p, "\r\n"...)

	// Clear deadlines set by HTTP server.
	netConn.SetDeadline(time.Time{})

	if u.HandshakeTimeout > 0 {
		netConn.SetWriteDeadline(time.Now().Add(u.HandshakeTimeout))
	}

	if _, err = netConn.Write(p); err != nil {
		netConn.Close()
		return nil, err
	}

	if u.HandshakeTimeout > 0 {
		netConn.SetWriteDeadline(time.Time{})
	}

	return c, nil
}

// Subprotocols returns the subprotocols requested by the client in the
// Sec-Websocket-Protocol header.
func Subprotocols(r *Request) []string {

	h := strings.TrimSpace(string(r.Request.Header.Peek("Sec-Websocket-Protocol")))
	if h == "" {
		return nil
	}

	protocols := strings.Split(h, ",")
	for i := range protocols {
		protocols[i] = strings.TrimSpace(protocols[i])
	}

	return protocols
}
