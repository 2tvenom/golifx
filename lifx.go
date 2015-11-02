package golifx

import "net"

var (
	conn = &connection{bcastAddress: net.IPv4bcast}
)

const (
	_GET_SERVICE         = 2
	_STATE_SERVICE       = 3
	_GET_HOST_INFO       = 12
	_STATE_HOST_INFO     = 13
	_GET_HOST_FIRMWARE   = 14
	_STATE_HOST_FIRMWARE = 15
	_GET_WIFI_INFO       = 16
	_STATE_WIFI_INFO     = 17
	_GET_WIFI_FIRMWARE   = 18
	_STATE_WIFI_FIRMWARE = 19
	_GET_POWER           = 20
	_SET_POWER           = 21
	_STATE_POWER         = 22
	_GET_LABEL           = 23
	_SET_LABEL           = 24
	_STATE_LABEL         = 25
	_GET_VERSION         = 32
	_STATE_VERSION       = 33
	_GET_INFO            = 34
	_STATE_INFO          = 35
	_ACKNOWLEDGEMENT     = 45
	_GET_LOCATION        = 48
	_STATE_LOCATION      = 50
	_GET_GROUP           = 51
	_STATE_GROUP         = 53
	_ECHO_REQUEST        = 58
	_ECHO_RESPONSE       = 59

	_GET                  = 101
	_SET_COLOR            = 102
	_STATE                = 107
	_GET_POWER_DURATION   = 116
	_SET_POWER_DURATION   = 117
	_POWER_STATE_DURATION = 118
)

func LookupBulbs() ([]*Bulb, error) {
	message := makeMessage()
	message.tagged = true
	message._type = _GET_SERVICE

	messages, err := conn.sendAndReceive(message)

	if err != nil {
		return nil, err
	}

	bulbs := []*Bulb{}

	for _, message := range messages {
		if message.payout[0] != 1 {
			continue
		}

		bulb := &Bulb{}
		bulb.hardwareAddress = message.target
		bulb.ipAddress = message.addr

		var port uint32

		readUint32(message.payout[1:5], &port)
		bulb.port = port
		bulbs = append(bulbs, bulb)
	}

	return bulbs, nil
}

func SetBroadcastAddress(addr net.IP) {
	conn.bcastAddress = addr
}
