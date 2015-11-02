package golifx

import (
	"net"
	"time"
)

type connection struct {
	bcastAddress net.IP
}

const (
	_DEFAULT_MAX_DEAD_LINE = time.Millisecond * 500
	_DEFAULT_PORT          = 56700
)

func (c *connection) get() (*net.UDPConn, error) {
	udpConn, err := net.ListenPacket("udp", ":0")

	if err != nil {
		return nil, err
	}

	conn := udpConn.(*net.UDPConn)
	return conn, nil
}

func (c *connection) sendAndReceive(inMessage *message) ([]*message, error) {
	return c.sendAndReceiveDead(inMessage, _DEFAULT_MAX_DEAD_LINE)
}

func (c *connection) sendAndReceiveDead(inMessage *message, deadline time.Duration) ([]*message, error) {
	conn, err := c.get()
	conn.SetReadDeadline(time.Now().Add(deadline))
	defer conn.Close()

	if err != nil {
		return nil, err
	}

	_, err = conn.WriteTo(inMessage.ReadRaw(), &net.UDPAddr{
		IP:   c.bcastAddress,
		Port: _DEFAULT_PORT,
	})

	if err != nil {
		return nil, err
	}

	messages := []*message{}

	for {
		buff := make([]byte, 512)
		n, addr, err := conn.ReadFrom(buff)
		if neterr, ok := err.(net.Error); ok && neterr.Timeout() {
			return messages, nil
		}

		msg := makeMessage()
		msg.Write(buff[:n])
		msg.addr = addr

		messages = append(messages, msg)
	}

	return messages, nil
}
