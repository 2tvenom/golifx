package golifx

import (
	"encoding/binary"
	"io"
	"math"
	"net"
)

type (
	header struct {
		//frame
		tagged      bool
		addressable bool
		source      uint32

		//frame address
		target       uint64
		ack_required bool
		res_required bool
		sequence     uint8

		//header
		_type uint16
	}

	message struct {
		*header
		payout []byte
		addr   net.Addr
	}
)

const (
	_DEFAULT_SOURCE_VALUE  = 7
	_DEFAULT_HEADER_LENGTH = 36
)

func makeMessage() *message {
	return &message{
		header: &header{
			addressable: true,
			source:      _DEFAULT_SOURCE_VALUE,
		},
		payout: nil,
	}
}

func makeMessageWithType(tp uint16) *message {
	msg := makeMessage()
	msg._type = tp
	return msg
}

func (m *message) Write(data []byte) (n int, err error) {

	m.tagged = (data[3] >> 5 & 1) == 1
	m.addressable = (data[3] >> 4 & 1) == 1

	readUint32(data[4:8], &m.source)
	readUint64(data[8:16], &m.target)

	m.ack_required = (data[22] >> 1 & 1) == 1
	m.res_required = (data[22] & 1) == 1
	m.sequence = data[23]

	readUint16(data[32:34], &m._type)

	if len(data) > _DEFAULT_HEADER_LENGTH {
		m.payout = data[_DEFAULT_HEADER_LENGTH:]
	}

	return len(data), nil
}

func (m *message) ReadRaw() []byte {
	buff := make([]byte, 512)
	n, _ := m.Read(buff)
	return buff[:n]
}

func (m *message) Read(p []byte) (n int, err error) {
	length := _DEFAULT_HEADER_LENGTH

	if m.payout != nil {
		length = _DEFAULT_HEADER_LENGTH + len(m.payout)
	}

	data := make([]byte, _DEFAULT_HEADER_LENGTH)

	writeUInt16(data[0:2], uint16(length))

	data[3] = (boolToUInt8(m.tagged) << 5) | (boolToUInt8(m.addressable) << 4) | (uint8(1) << 2)
	writeUInt32(data[4:8], m.source)
	writeUInt64(data[8:16], m.target)
	data[22] = (boolToUInt8(m.ack_required))<<1 | boolToUInt8(m.res_required)
	data[23] = m.sequence
	writeUInt16(data[32:34], m._type)

	if m.payout != nil {
		data = append(data, m.payout...)
	}

	copy(p, data)

	return length, io.EOF
}

func boolToUInt8(b bool) uint8 {
	if b {
		return 1
	}
	return 0
}

func writeUInt16(buff []byte, data uint16) {
	buff[0] = byte(data)
	buff[1] = byte(data >> 8)
}

func writeUInt32(buff []byte, data uint32) {
	for i := 0; i < 4; i++ {
		buff[i] = byte(data >> uint(i*8))
	}
}

func writeUInt64(buff []byte, data uint64) {
	for i := 0; i < 8; i++ {
		buff[i] = byte(data >> uint(i*8))
	}
}

func writeFloat32(buff []byte, float float32) {
	binary.LittleEndian.PutUint32(buff, math.Float32bits(float))
}

func readUint16(buff []byte, dest *uint16) error {
	*dest = uint16(buff[0] & 0xFF)
	*dest += uint16(buff[1]&0xFF) << 8
	return nil
}

func readUint32(buff []byte, dest *uint32) error {
	*dest = 0
	for i := 0; i < 4; i++ {
		*dest += uint32(buff[i]&0xFF) << uint(i*8)
	}

	return nil
}

func readUint64(buff []byte, dest *uint64) error {
	*dest = 0
	for i := 0; i < 8; i++ {
		*dest += uint64(buff[i]&0xFF) << uint(i*8)
	}

	return nil
}

func readFloat32(buff []byte, float *float32) {
	*float = math.Float32frombits(binary.LittleEndian.Uint32(buff))
}
