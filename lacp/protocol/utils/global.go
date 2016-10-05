// global.go
package utils

// The following dbs are used to keep track of
// certain conditions that must exist from a config
// check perspective.
// 1) A Port can only be part of one agg group
// 2) An Agg can only be part of one distributed relay group
// holds the agg to port list
var ConfigAggMap map[string][]uint16

// holds the dr to agg list
var ConfigDrMap map[string]uint32

const (
	// init
	LACP_GLOBAL_INIT = iota + 1
	// allow all config
	LACP_GLOBAL_ENABLE
	// disallow all config
	LACP_GLOBAL_DISABLE
	// transition state to to allow deleting on
	// when global state changes to disable
	LACP_GLOBAL_DISABLE_PENDING
)

var LacpGlobalState int = LACP_GLOBAL_INIT

func LacpGlobalStateSet(state int) {
	LacpGlobalState = state
}

func LacpGlobalStateGet() int {
	return LacpGlobalState
}
