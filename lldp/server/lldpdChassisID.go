package lldpServer

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
