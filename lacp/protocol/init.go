// init
package lacp

import ()

var LaSystemIdDefault LacpSystem

func init() {

	// Default System Id is all zero's
	// this will be used by all static lags, as well as initial
	// aggregation configs.
	LaSystemIdDefault = LacpSystem{
		Actor_System_priority: 0,
		actor_System:          [6]uint8{0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
	}
	LacpSysGlobalInfoInit(LaSystemIdDefault)
}
