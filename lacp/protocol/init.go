// init
package lacp

import ()

var LaSystemIdDefault LacpSystem

func init() {

	// Default system Id is all zero's
	// this will be used by all static lags, as well as initial
	// aggregation configs.
	LaSystemIdDefault = LacpSystem{
		actor_system_priority: 0,
		actor_system:          [6]uint8{0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
	}
	LacpSysGlobalInfoInit(LaSystemIdDefault)
}
