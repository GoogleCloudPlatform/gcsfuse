package dns

import (
	"bytes"
	"context"
	"io"
	"net"
	"strings"
	"sync"
	"time"
)

type dnsConn struct {
	cancel    context.CancelFunc // +checklocks:Mutex
	deadline  time.Time          // +checklocks:Mutex
	roundTrip roundTripper

	ibuf bytes.Buffer // +checklocks:Mutex
	obuf bytes.Buffer // +checklocks:Mutex

	sync.Mutex
}

type roundTripper func(ctx context.Context, req string) (res string, err error)

func (c *dnsConn) Read(b []byte) (n int, err error) {
	imsg, n, err := c.drainBuffers(b)
	if n != 0 || err != nil {
		return n, err
	}

	ctx, cancel := c.childContext()
	omsg, err := c.roundTrip(ctx, imsg)
	cancel()
	if err != nil {
		return 0, err
	}

	return c.fillBuffer(b, omsg)
}

func (c *dnsConn) Write(b []byte) (n int, err error) {
	c.Lock()
	defer c.Unlock()
	return c.ibuf.Write(b)
}

func (c *dnsConn) Close() error {
	c.Lock()
	cancel := c.cancel
	c.Unlock()

	if cancel != nil {
		cancel()
	}
	return nil
}

func (c *dnsConn) LocalAddr() net.Addr {
	return nil
}

func (c *dnsConn) RemoteAddr() net.Addr {
	return nil
}

func (c *dnsConn) SetDeadline(t time.Time) error {
	c.SetReadDeadline(t)
	c.SetWriteDeadline(t)
	return nil
}

func (c *dnsConn) SetReadDeadline(t time.Time) error {
	c.Lock()
	defer c.Unlock()
	c.deadline = t
	return nil
}

func (c *dnsConn) SetWriteDeadline(t time.Time) error {
	// writes do not timeout
	return nil
}

func (c *dnsConn) drainBuffers(b []byte) (string, int, error) {
	c.Lock()
	defer c.Unlock()

	// drain the output buffer
	if c.obuf.Len() > 0 {
		n, err := c.obuf.Read(b)
		return "", n, err
	}

	// otherwise, get the next message from the input buffer
	sz := c.ibuf.Next(2)
	if len(sz) < 2 {
		return "", 0, io.ErrUnexpectedEOF
	}

	size := int64(sz[0])<<8 | int64(sz[1])

	var str strings.Builder
	_, err := io.CopyN(&str, &c.ibuf, size)
	if err == io.EOF {
		return "", 0, io.ErrUnexpectedEOF
	}
	if err != nil {
		return "", 0, err
	}
	return str.String(), 0, nil
}

func (c *dnsConn) fillBuffer(b []byte, str string) (int, error) {
	c.Lock()
	defer c.Unlock()
	c.obuf.Grow(len(str) + 2)
	c.obuf.WriteByte(byte(len(str) >> 8))
	c.obuf.WriteByte(byte(len(str)))
	c.obuf.WriteString(str)
	return c.obuf.Read(b)
}

func (c *dnsConn) childContext() (context.Context, context.CancelFunc) {
	c.Lock()
	defer c.Unlock()
	ctx := context.Background()
	if c.deadline.IsZero() {
		return ctx, func() {}
	}
	return context.WithDeadline(ctx, c.deadline)
}

func writeMessage(conn net.Conn, msg string) error {
	var buf []byte
	pc, udp := conn.(net.PacketConn)
	if udp {
		buf = []byte(msg)
	} else {
		buf = make([]byte, len(msg)+2)
		buf[0] = byte(len(msg) >> 8)
		buf[1] = byte(len(msg))
		copy(buf[2:], msg)
	}
	// SHOULD do a single write on TCP (RFC 7766, section 8).
	// MUST do a single write on UDP.
	_, err := conn.Write(buf)
	if err != nil && udp {
		// Write failed, maybe WriteTo can work.
		if addr := conn.RemoteAddr(); addr != nil {
			_, err := pc.WriteTo(buf, addr)
			if err == nil {
				return nil
			}
		}
	}
	return err
}

func readMessage(c net.Conn) (string, error) {
	if _, udp := c.(net.PacketConn); udp {
		// RFC 1035 specifies 512 as the maximum message size for DNS over UDP.
		// RFC 6891 OTOH suggests 4096 as the maximum payload size for EDNS.
		b := make([]byte, 4096)
		n, err := c.Read(b)
		if err != nil {
			return "", err
		}
		return string(b[:n]), nil
	} else {
		var sz [2]byte
		_, err := io.ReadFull(c, sz[:])
		if err != nil {
			return "", err
		}

		size := int64(sz[0])<<8 | int64(sz[1])

		var str strings.Builder
		_, err = io.CopyN(&str, c, size)
		if err == io.EOF {
			return "", io.ErrUnexpectedEOF
		}
		if err != nil {
			return "", err
		}
		return str.String(), nil
	}
}
