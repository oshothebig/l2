package lldpServer

import (
	"time"
)

// A LLDPFrame is an lldp frame or lldp Data Unit aka LLDPDU. A frame carries
// infomration related to device in a series of TLV (type-length-value) structs
type LLDPFrame struct {
	// ChassisID will indicate Chassis ID information regarding a device. This
	// field is mandatory.
	ChassisID *LLDPChassisID

	// PortID will indicate Port ID information regarding a device. This
	// field is mandatory
	PortID *LLDPPortID

	// TTL will indicate how long the info in the frame should be considered
	// valid. This field is mandatory
	TTL time.Duration

	// Optional specifies zero or more optional TLV values in raw format.
	// This is also mendatory
	Optional []*LLDPTLV
}
