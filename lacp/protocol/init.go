// init
package lacp

import ()

var LaSystemIdDefault [6]uint8

func init() {

	// TODO write some logic to read the system sysId from a config file
	// hard coding for now
	LaSystemIdDefault = [6]uint8{0x00, 0x00, 0x01, 0x02, 0x03, 0x04}

	LacpSysGlobalInfoInit(LaSystemIdDefault)
}
