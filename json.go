package golifx

import (
	"encoding/json"
)

func (b Bulb) MarshalJSON() ([]byte, error) {
	index := map[string]interface{}{
		"mac":         b.MacAddress(),
		"ip":          b.ipAddress,
		"label":       b.label,
		"power_state": b.powerState,
	}

	if b.stateHostInfo != nil {
		index["host_info"] = map[string]interface{}{
			"signal": b.stateHostInfo.Signal,
			"rx":     b.stateHostInfo.Rx,
			"tx":     b.stateHostInfo.Tx,
		}
	}

	if b.wifiInfo != nil {
		index["wifi_info"] = map[string]interface{}{
			"signal": b.wifiInfo.Signal,
			"rx":     b.wifiInfo.Rx,
			"tx":     b.wifiInfo.Tx,
		}
	}

	if b.version != nil {
		index["version"] = map[string]interface{}{
			"version":    b.version.Version,
			"product_id": b.version.ProductId,
			"vendor_id":  b.version.VendorId,
		}
	}

	if b.hostFirmware != nil {
		index["host_firmware"] = map[string]interface{}{
			"build":   b.hostFirmware.Build,
			"version": b.hostFirmware.Version,
		}
	}

	if b.wifiFirmware != nil {
		index["wifi_firmware"] = map[string]interface{}{
			"build":   b.wifiFirmware.Build,
			"version": b.wifiFirmware.Version,
		}
	}

	if b.info != nil {
		index["info"] = map[string]interface{}{
			"downtime": b.info.Downtime.Nanoseconds(),
			"time":     durationToStrDate(b.info.Time),
			"uptime":   b.info.UpTime.Seconds(),
		}
	}

	if b.location != nil {
		index["location"] = map[string]interface{}{
			"label":     b.location.Label,
			"updatedat": durationToStrDate(b.location.UpdatedAt),
		}
	}

	if b.group != nil {
		index["group"] = map[string]interface{}{
			"label":     b.group.Label,
			"updatedat": durationToStrDate(b.group.UpdatedAt),
		}
	}

	if b.color != nil {
		index["color"] = map[string]interface{}{
			"hue":        b.color.Hue,
			"kelvin":     b.color.Kelvin,
			"saturation": b.color.Saturation,
			"brightness": b.color.Brightness,
		}
	}

	return json.Marshal(index)
}
