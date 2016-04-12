package lldpServer

import (
	"io"
)

// chassis id type def
type lldpChassisIDSubtype uint8

// list of all valid chassis id subtype value
const (
	ChassisIDSubtypeReserved           lldpChassisIDSubtype = 0
	ChassisIDSubtypeChassisComponenent lldpChassisIDSubtype = 1
	ChassisIDSubtypeInterfaceAlias     lldpChassisIDSubtype = 2
	ChassisIDSubtypePortComponent      lldpChassisIDSubtype = 3
	ChassisIDSubtypeMACAddress         lldpChassisIDSubtype = 4
	ChassisIDSubtypeNetworkAddress     lldpChassisIDSubtype = 5
	ChassisIDSubtypeInterfaceName      lldpChassisIDSubtype = 6
	ChassisIDSubtypeLocallyAssigned    lldpChassisIDSubtype = 7
)

// struct that defines lldp chassis id in lldp frame received on the wire.
// This will contain information pertaining to a particular chassis on a given
// network
type LLDPChassisID struct {
	// Subtype specifies the type of identification carried.
	SubType lldpChassisIDSubtype

	// ID specifies raw bytes containing identification information for
	// that ChassisID.
	//
	// ID may also carry alphanumeric data or binary data, depending upon the
	// value of Subtype, received or transmitted
	ID []byte
}

// @FIXME: do we need to return error??
// Marshall chassis id information into binary form
func (c *LLDPChassisID) LLDPChassisIDMarshall() ([]byte, error) {
	// SubType : 1 Byte
	// ID : N Bytes
	// This is the total length of the byte object
	b := make([]byte, 1+len(c.ID))
	// Copy SubType first
	b[0] = byte(c.SubType)
	// Copy id information
	copy(b[1:], c.ID)

	return b, nil
}

// UnMarshall chassis id information from binary form to LLDPChassisID
func (c *LLDPChassisID) LLDPChassisIDUnMarshall(b []byte) error {
	// Mandatory field of subtype should be specified
	if len(b) < 1 {
		return io.ErrUnexpectedEOF
	}
	c.SubType = lldpChassisIDSubtype(b[0])
	c.ID = make([]byte, len(b[1:]))
	copy(c.ID, b[1:])

	return nil
}
