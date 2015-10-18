// muxmachine
package lacp

import ()

const (
	LacpMuxStateDetached = iota
	LacpMuxStateWaiting
	LacpMuxStateAttached
	LacpMuxStateCollecting
	LacpMuxStateDistributing
)
