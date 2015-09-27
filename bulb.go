package golifx

import (
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
)

type (
	Bulb struct {
		hardwareAddress uint64
		ipAddress       net.Addr
		port            uint32
		label           string
	}

	BulbSignalInfo struct {
		Signal float32
		Tx     uint32
		Rx     uint32
	}

	BulbVersion struct {
		VendorId  uint32
		ProductId uint32
		Version   uint32
	}

	BulbFirmware struct {
		Build   uint64
		Version uint32
	}

	BulbStateInfo struct {
		Time     time.Duration
		UpTime   time.Duration
		Downtime time.Duration
	}

	BulbLocation struct {
		Location  []byte
		Label     string
		UpdatedAt time.Duration
	}

	HSBK struct {
		Hue        uint16
		Saturation uint16
		Brightness uint16
		Kelvin     uint16
	}

	BulbState struct {
		Color *HSBK
		Power bool
		Label string
	}
)

var (
	noResponse            = errors.New("No acknowledgement response")
	incorrectResponseType = errors.New("Incorrect response type")
)

func (b *Bulb) sendAndReceive(msg *message) (*message, error) {
	return b.sendAndReceiveDead(msg, _DEFAULT_MAX_DEAD_LINE)
}

func (b *Bulb) sendAndReceiveDead(msg *message, deadLine time.Duration) (*message, error) {
	msg.target = b.hardwareAddress
	messages, err := conn.sendAndReceiveDead(msg, deadLine)

	if err != nil {
		return nil, err
	}

	for _, m := range messages {
		if m.target != b.hardwareAddress {
			continue
		}
		return m, nil
	}

	return nil, io.EOF
}

func (b *Bulb) sendWithAcknowledgement(msg *message, deadLine time.Duration) error {
	msg.target = b.hardwareAddress
	msg.ack_required = true

	msg, err := b.sendAndReceiveDead(msg, deadLine)

	if err != nil {
		return err
	}

	if msg._type != _ACKNOWLEDGEMENT {
		return noResponse
	}
	return nil
}

func (b *Bulb) MacAddress() string {
	mac := make([]byte, 8)
	writeUInt64(mac, b.hardwareAddress)
	return strings.Replace(fmt.Sprintf("% x", mac[0:6]), " ", ":", -1)
}

func (b *Bulb) IP() net.Addr {
	return b.ipAddress
}

func (b *Bulb) GetPowerState() (bool, error) {
	msg, err := b.sendAndReceive(makeMessageWithType(_GET_POWER))

	if err != nil {
		return false, err
	}

	if msg._type != _STATE_POWER {
		return false, incorrectResponseType
	}

	var state uint16

	readUint16(msg.payout, &state)
	return state != 0, nil
}

func (b *Bulb) SetPowerState(state bool) error {
	msg := makeMessageWithType(_SET_POWER)
	msg.payout = []byte{0, 0}

	if state {
		msg.payout = []byte{0xFF, 0xFF}
	}

	return b.sendWithAcknowledgement(msg, time.Millisecond*500)
}

func (b *Bulb) GetLabel() (string, error) {
	msg, err := b.sendAndReceive(makeMessageWithType(_GET_LABEL))

	if err != nil {
		return "", err
	}

	if msg._type != _STATE_LABEL {
		return "", incorrectResponseType
	}

	b.label = strings.TrimSpace(string(msg.payout))

	return b.label, nil
}

func (b *Bulb) SetLabel(label string) error {
	msg := makeMessageWithType(_SET_LABEL)

	msg.payout = []byte(label)

	if len(msg.payout) > 32 {
		msg.payout = msg.payout[:32]
	} else {
		msg.payout = append(msg.payout, make([]byte, 32-len(msg.payout))...)
	}

	return b.sendWithAcknowledgement(msg, time.Millisecond*100)
}

func (b *Bulb) GetStateHostInfo() (*BulbSignalInfo, error) {
	msg, err := b.sendAndReceive(makeMessageWithType(_GET_HOST_INFO))

	if err != nil {
		return nil, err
	}

	if msg._type != _STATE_HOST_INFO {
		return nil, incorrectResponseType
	}

	return parseSignal(msg.payout), nil
}

func (b *Bulb) GetWifiInfo() (*BulbSignalInfo, error) {
	msg, err := b.sendAndReceive(makeMessageWithType(_GET_WIFI_INFO))

	if err != nil {
		return nil, err
	}

	if msg._type != _STATE_WIFI_INFO {
		return nil, incorrectResponseType
	}

	return parseSignal(msg.payout), nil
}

func parseSignal(payout []byte) *BulbSignalInfo {
	signalInfo := &BulbSignalInfo{}

	readFloat32(payout[:4], &signalInfo.Signal)
	readUint32(payout[4:8], &signalInfo.Tx)
	readUint32(payout[8:12], &signalInfo.Rx)
	return signalInfo
}

func (b *Bulb) GetVersion() (*BulbVersion, error) {
	msg, err := b.sendAndReceive(makeMessageWithType(_GET_VERSION))

	if err != nil {
		return nil, err
	}

	if msg._type != _STATE_VERSION {
		return nil, incorrectResponseType
	}

	version := &BulbVersion{}

	readUint32(msg.payout[:4], &version.VendorId)
	readUint32(msg.payout[4:8], &version.ProductId)
	readUint32(msg.payout[8:], &version.Version)

	return version, nil
}

func (b *Bulb) GetHostFirmware() (*BulbFirmware, error) {
	msg, err := b.sendAndReceive(makeMessageWithType(_GET_HOST_FIRMWARE))

	if err != nil {
		return nil, err
	}

	if msg._type != _STATE_HOST_FIRMWARE {
		return nil, incorrectResponseType
	}

	return parseFirmware(msg.payout), nil
}

func (b *Bulb) GetWifiFirmware() (*BulbFirmware, error) {
	msg, err := b.sendAndReceive(makeMessageWithType(_GET_WIFI_FIRMWARE))

	if err != nil {
		return nil, err
	}

	if msg._type != _STATE_WIFI_FIRMWARE {
		return nil, incorrectResponseType
	}

	return parseFirmware(msg.payout), nil
}

func parseFirmware(payout []byte) *BulbFirmware {
	firmware := &BulbFirmware{}

	readUint64(payout[:8], &firmware.Build)
	readUint32(payout[16:], &firmware.Version)

	return firmware
}

func (b *Bulb) GetInfo() (*BulbStateInfo, error) {
	msg, err := b.sendAndReceive(makeMessageWithType(_GET_INFO))

	if err != nil {
		return nil, err
	}

	if msg._type != _STATE_INFO {
		return nil, incorrectResponseType
	}

	info := &BulbStateInfo{}

	var i uint64

	readUint64(msg.payout[:8], &i)
	info.Time = time.Duration(i)
	readUint64(msg.payout[8:16], &i)
	info.UpTime = time.Duration(i)
	readUint64(msg.payout[16:], &i)
	info.Downtime = time.Duration(i)

	return info, nil
}

func (b *Bulb) GetLocation() (*BulbLocation, error) {
	msg, err := b.sendAndReceive(makeMessageWithType(_GET_LOCATION))

	if err != nil {
		return nil, err
	}

	if msg._type != _STATE_LOCATION {
		return nil, incorrectResponseType
	}

	return parseLocation(msg.payout), nil
}

func (b *Bulb) GetGroup() (*BulbLocation, error) {
	msg, err := b.sendAndReceive(makeMessageWithType(_GET_GROUP))

	if err != nil {
		return nil, err
	}

	if msg._type != _STATE_GROUP {
		return nil, incorrectResponseType
	}

	return parseLocation(msg.payout), nil
}

func parseLocation(payout []byte) *BulbLocation {
	location := &BulbLocation{}
	location.Location = payout[:16]
	location.Label = strings.TrimSpace(string(payout[16:48]))
	var i uint64
	readUint64(payout[48:], &i)
	location.UpdatedAt = time.Duration(i)

	return location
}

func (b *Bulb) EchoRequest(echoRequest []byte) ([]byte, error) {
	if len(echoRequest) > 64 {
		return nil, errors.New("Echo request max length is 64")
	}

	msg := makeMessageWithType(_ECHO_REQUEST)
	msg.payout = echoRequest
	msg, err := b.sendAndReceive(msg)

	if msg._type != _ECHO_RESPONSE {
		return nil, incorrectResponseType
	}

	if err != nil {
		return nil, err
	}

	return msg.payout, nil
}

func (b *Bulb) GetPowerDurationState() (bool, error) {
	msg, err := b.sendAndReceive(makeMessageWithType(_GET_POWER_DURATION))

	if err != nil {
		return false, err
	}

	if msg._type != _POWER_STATE_DURATION {
		return false, incorrectResponseType
	}

	var state uint16

	readUint16(msg.payout, &state)
	return state != 0, nil
}

func (b *Bulb) SetPowerDurationState(state bool, duration uint32) error {
	msg := makeMessageWithType(_SET_POWER_DURATION)

	msg.payout = make([]byte, 6)

	if state {
		msg.payout[0], msg.payout[1] = 0xFF, 0xFF
	}

	if duration > 0 {
		writeUInt32(msg.payout[2:], duration)
	}

	return b.sendWithAcknowledgement(msg, time.Millisecond*500)
}

func (h *HSBK) Write(data []byte) (n int, err error) {
	readUint16(data[:2], &h.Hue)
	readUint16(data[2:4], &h.Saturation)
	readUint16(data[4:6], &h.Brightness)
	readUint16(data[6:8], &h.Kelvin)
	return 8, nil
}

func (h *HSBK) Read(p []byte) (n int, err error) {
	data := make([]byte, 8)
	writeUInt16(data[:2], h.Hue)
	writeUInt16(data[2:4], h.Saturation)
	writeUInt16(data[4:6], h.Brightness)
	writeUInt16(data[6:8], h.Kelvin)

	copy(p, data)

	return 8, io.EOF
}

func (b *Bulb) GetColorState() (*BulbState, error) {
	msg, err := b.sendAndReceive(makeMessageWithType(_GET))

	if err != nil {
		return nil, err
	}

	if msg._type != _STATE {
		return nil, incorrectResponseType
	}

	return parseColorState(msg.payout), nil
}

func (b *Bulb) SetColorState(hsbk *HSBK, duration uint32) error {
	msg := makeMessageWithType(_SET_COLOR)

	msg.payout = make([]byte, 13)

	hsbk.Read(msg.payout[1:9])

	if duration > 0 {
		writeUInt32(msg.payout[9:], duration)
	}

	return b.sendWithAcknowledgement(msg, time.Millisecond*500)
}

func (b *Bulb) SetColorStateWithResponse(hsbk *HSBK, duration uint32) (*BulbState, error) {
	msg := makeMessageWithType(_SET_COLOR)
	msg.res_required = true

	msg.payout = make([]byte, 13)

	hsbk.Read(msg.payout[1:9])

	if duration > 0 {
		writeUInt32(msg.payout[9:], duration)
	}

	msg, err := b.sendAndReceive(msg)

	if err != nil {
		return nil, err
	}
	return parseColorState(msg.payout), nil
}

func parseColorState(payout []byte) *BulbState {
	hsbk := &HSBK{}
	hsbk.Write(payout[:8])

	state := &BulbState{Color: hsbk}

	var powerState uint16
	readUint16(payout[10:12], &powerState)

	state.Power = powerState != 0
	state.Label = strings.TrimSpace(string(payout[12:44]))
	return state
}
