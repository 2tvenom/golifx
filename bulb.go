package golifx

import (
	"bytes"
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
		powerState      bool
		stateHostInfo   *BulbSignalInfo
		wifiInfo        *BulbSignalInfo
		version         *BulbVersion
		hostFirmware    *BulbFirmware
		wifiFirmware    *BulbFirmware
		info            *BulbStateInfo
		location        *BulbLocation
		group           *BulbLocation
		color           *HSBK
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

func (b *Bulb) SetHardwareAddress(address uint64) {
	b.hardwareAddress = address
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

	b.powerState = state != 0
	return b.powerState, nil
}

func (b *Bulb) SetPowerState(state bool) error {
	msg := makeMessageWithType(_SET_POWER)
	msg.payout = []byte{0, 0}

	if state {
		msg.payout = []byte{0xFF, 0xFF}
	}

	err := b.sendWithAcknowledgement(msg, time.Millisecond*500)

	if err != nil {
		return err
	}

	b.powerState = state
	return nil
}

func (b *Bulb) GetLabel() (string, error) {
	msg, err := b.sendAndReceive(makeMessageWithType(_GET_LABEL))

	if err != nil {
		return "", err
	}

	if msg._type != _STATE_LABEL {
		return "", incorrectResponseType
	}

	b.label = string(bytes.Trim(msg.payout, "\x00"))

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

	err := b.sendWithAcknowledgement(msg, time.Millisecond*500)

	if err != nil {
		return err
	}

	b.label = label
	return nil
}

func (b *Bulb) GetStateHostInfo() (*BulbSignalInfo, error) {
	msg, err := b.sendAndReceive(makeMessageWithType(_GET_HOST_INFO))

	if err != nil {
		return nil, err
	}

	if msg._type != _STATE_HOST_INFO {
		return nil, incorrectResponseType
	}

	b.stateHostInfo = parseSignal(msg.payout)
	return b.stateHostInfo, nil
}

func (b *Bulb) GetWifiInfo() (*BulbSignalInfo, error) {
	msg, err := b.sendAndReceive(makeMessageWithType(_GET_WIFI_INFO))

	if err != nil {
		return nil, err
	}

	if msg._type != _STATE_WIFI_INFO {
		return nil, incorrectResponseType
	}

	b.wifiInfo = parseSignal(msg.payout)
	return b.wifiInfo, nil
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

	b.version = &BulbVersion{}

	readUint32(msg.payout[:4], &b.version.VendorId)
	readUint32(msg.payout[4:8], &b.version.ProductId)
	readUint32(msg.payout[8:], &b.version.Version)

	return b.version, nil
}

func (b *Bulb) GetHostFirmware() (*BulbFirmware, error) {
	msg, err := b.sendAndReceive(makeMessageWithType(_GET_HOST_FIRMWARE))

	if err != nil {
		return nil, err
	}

	if msg._type != _STATE_HOST_FIRMWARE {
		return nil, incorrectResponseType
	}

	b.hostFirmware = parseFirmware(msg.payout)

	return b.hostFirmware, nil
}

func (b *Bulb) GetWifiFirmware() (*BulbFirmware, error) {
	msg, err := b.sendAndReceive(makeMessageWithType(_GET_WIFI_FIRMWARE))

	if err != nil {
		return nil, err
	}

	if msg._type != _STATE_WIFI_FIRMWARE {
		return nil, incorrectResponseType
	}

	b.wifiFirmware = parseFirmware(msg.payout)

	return b.wifiFirmware, nil
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

	b.info = &BulbStateInfo{}

	var i uint64

	readUint64(msg.payout[:8], &i)
	b.info.Time = time.Duration(i)
	readUint64(msg.payout[8:16], &i)
	b.info.UpTime = time.Duration(i)
	readUint64(msg.payout[16:], &i)
	b.info.Downtime = time.Duration(i)

	return b.info, nil
}

func (b *Bulb) GetLocation() (*BulbLocation, error) {
	msg, err := b.sendAndReceive(makeMessageWithType(_GET_LOCATION))

	if err != nil {
		return nil, err
	}

	if msg._type != _STATE_LOCATION {
		return nil, incorrectResponseType
	}

	b.location = parseLocation(msg.payout)

	return b.location, nil
}

func (b *Bulb) GetGroup() (*BulbLocation, error) {
	msg, err := b.sendAndReceive(makeMessageWithType(_GET_GROUP))

	if err != nil {
		return nil, err
	}

	if msg._type != _STATE_GROUP {
		return nil, incorrectResponseType
	}

	b.group = parseLocation(msg.payout)

	return b.group, nil
}

func parseLocation(payout []byte) *BulbLocation {
	location := &BulbLocation{}
	location.Location = payout[:16]
	location.Label = string(bytes.Trim(payout[16:48], "\x00"))
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

	b.powerState = state != 0
	return b.powerState, nil
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

	err := b.sendWithAcknowledgement(msg, time.Millisecond*500)

	if err != nil {
		return err
	}

	b.powerState = state
	return nil
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

	state := parseColorState(msg.payout)
	b.powerState = state.Power
	b.label = state.Label
	b.color = state.Color
	return state, nil
}

func (b *Bulb) SetColorState(hsbk *HSBK, duration uint32) error {
	msg := makeMessageWithType(_SET_COLOR)
	msg.payout = make([]byte, 13)

	hsbk.Read(msg.payout[1:9])

	if duration > 0 {
		writeUInt32(msg.payout[9:], duration)
	}

	err := b.sendWithAcknowledgement(msg, time.Millisecond*500)

	if err != nil {
		return err
	}

	b.color = hsbk

	return nil
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

	state := parseColorState(msg.payout)
	b.powerState = state.Power
	b.label = state.Label
	b.color = state.Color
	return state, nil
}

func parseColorState(payout []byte) *BulbState {
	hsbk := &HSBK{}
	hsbk.Write(payout[:8])

	state := &BulbState{Color: hsbk}

	var powerState uint16
	readUint16(payout[10:12], &powerState)

	state.Power = powerState != 0

	state.Label = string(bytes.Trim(payout[12:44], "\x00"))
	return state
}

func (b BulbSignalInfo) String() string {
	return fmt.Sprintf("Signal: %f\nRx: %d\nTx: %d\n", b.Signal, b.Rx, b.Tx)
}

func (b BulbVersion) String() string {
	return fmt.Sprintf("Vendor id: %d\nProduct id: %d\nVersion: %d\n", b.VendorId, b.ProductId, b.Version)
}

func (b BulbFirmware) String() string {
	return fmt.Sprintf("Build: %d\nVersion: %d\n", b.Build, b.Version)
}

func (b BulbStateInfo) String() string {
	return fmt.Sprintf("Time: %s\nUpTime: %s\nDowntime: %s\n", durationToStrDate(b.Time), b.UpTime, b.Downtime)
}

func (b BulbLocation) String() string {
	return fmt.Sprintf("Label: %s\nUpdatedAt: %s\n", b.Label, durationToStrDate(b.UpdatedAt))
}

func (b HSBK) String() string {
	return fmt.Sprintf("HUE: %d\nSaturation: %d\nBrightness: %d\nKelvin: %d\n", b.Hue, b.Saturation, b.Brightness, b.Kelvin)
}

func (b BulbState) String() string {
	return fmt.Sprintf("Color: %sPower: %t\nLabel: %s\n", b.Color, b.Power, b.Label)
}

func (b Bulb) String() string {
	str := fmt.Sprintf("MAC: %s\nIP: %s\n", b.MacAddress(), b.IP())

	if b.label != "" {
		str += fmt.Sprintf("Label: %s\n", b.label)
	}

	str += fmt.Sprintf("Power state: %t\n", b.powerState)

	if b.stateHostInfo != nil {
		str += fmt.Sprintf("Host info:\n%s", b.stateHostInfo)
	}

	if b.wifiInfo != nil {
		str += fmt.Sprintf("Wi-Fi info:\n%s", b.wifiInfo)
	}

	if b.version != nil {
		str += fmt.Sprintf("Version:\n%s", b.version)
	}

	if b.hostFirmware != nil {
		str += fmt.Sprintf("Host Firmware:\n%s", b.hostFirmware)
	}

	if b.wifiFirmware != nil {
		str += fmt.Sprintf("Wi-Fi Firmware:\n%s", b.wifiFirmware)
	}

	if b.info != nil {
		str += fmt.Sprintf("Info:\n%s", b.info)
	}

	if b.location != nil {
		str += fmt.Sprintf("Location:\n%s", b.location)
	}

	if b.group != nil {
		str += fmt.Sprintf("Group:\n%s", b.group)
	}

	if b.color != nil {
		str += fmt.Sprintf("Color:\n%s", b.color)
	}

	return str
}

func durationToStrDate(d time.Duration) string {
	return time.Unix(int64(d.Seconds()), 0).Format("2006-01-02 15:04:05")
}
