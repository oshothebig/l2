// MUX MACHINE 802.1ax-2014 Section 6.4.15
package lacp

import (
//	"fmt"
//	"utils/fsm"
)

const (
	LacpMuxStateNone = iota
	LacpMuxStateDetached
	LacpMuxStateWaiting
	LacpMuxStateAttached
	LacpMuxStateCollecting
	LacpMuxStateDistributing
)
