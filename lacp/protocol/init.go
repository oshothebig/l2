// init
package lacp

import (
	"net"
)

var LaSystemIdDefault net.HardwareAddr

func init() {

	// TODO write some logic to read the system sysId from a config file
	// hard coding for now
	LaSystemIdDefault = make(net.HardwareAddr, 6)
	LaSystemIdDefault[0] = 0x00
	LaSystemIdDefault[0] = 0x00
	LaSystemIdDefault[0] = 0x01
	LaSystemIdDefault[0] = 0x02
	LaSystemIdDefault[0] = 0x03
	LaSystemIdDefault[0] = 0x04

	LacpSysGlobalInfoInit(LaSystemIdDefault)
}
