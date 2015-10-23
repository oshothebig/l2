// defs
package lacp

import (
	"time"
	"utils/fsm"
)

// 6.4.4 Constants
// number of seconds between periodic trasmissions using Short Timeouts
const LacpFastPeriodicTime time.Duration = (time.Second * 1)

// number of seconds etween periodic transmissions using Long timeouts
const LacpSlowPeriodicTime time.Duration = (time.Second * 30)

// number of seconds before invalidating received LACPDU info when using
// Short Timeouts (3 x LacpFastPeriodicTime)
// Lacp State Timeout == 1
const LacpShortTimeoutTime time.Duration = (time.Second * 3)

// number of seconds before invalidating received LACPDU info when using
// Long Timeouts (3 x LacpSlowPeriodicTime)
// Lacp State Timeout == 0
const LacpLongTimeoutTime time.Duration = (time.Second * 90)

// number of seconds that the Actor and Partner Churn state machines
// wait for the Actor or Partner Sync state to stabilize
const LacpChurnDetectionTime time.Duration = (time.Second * 60)

// number of seconds to delay aggregation to allow multiple links to
// aggregate simultaneously
const LacpAggregateWaitTime time.Duration = (time.Second * 2)

// the version number of the Actor LACP implementation
const LacpActorSystemLacpVersion int = 0x01

const LacpIsEnabled bool = true
const LacpIsDisabled bool = false

const (
	LacpStateActivityBit = 1 << iota
	LacpStateTimeoutBit
	LacpStateAggregationBit
	LacpStateSyncBit
	LacpStateCollectingBit
	LacpStateDistributingBit
	LacpStateDefaultedBit
	LacpStateExpiredBit
)

const (
	// also known as manual mode
	LacpModeOn = iota + 1
	// lacp state Activity == FALSE
	// considered lacp enabled
	LacpModeActive
	// lacp state Activity == TRUE
	// considered lacp enabled
	LacpModePassive
)

type LacpStateEvent struct {
	// current state
	s fsm.State
	// previous state
	ps fsm.State
	// current event
	e fsm.Event
	// previous event
	pe fsm.Event
}

func (se *LacpStateEvent) CurrentState() fsm.State { return se.s }
func (se *LacpStateEvent) CurrentEvent() fsm.Event { return se.e }
func (se *LacpStateEvent) SetState(s fsm.State)    { se.s = s }
func (se *LacpStateEvent) SetEvent(e fsm.Event)    { se.e = e }

func LacpStateSet(currState uint8, stateBits uint8) uint8 {
	return currState | stateBits
}

func LacpStateClear(currState uint8, stateBits uint8) uint8 {
	return currState & ^(stateBits)
}

func LacpStateIsSet(currState uint8, stateBits uint8) bool {
	return (currState & stateBits) == stateBits
}

func LacpModeGet(currState uint8, lacpDisabled bool) int {
	mode := LacpModeOn
	if !lacpDisabled {
		mode = LacpModePassive
		if LacpStateIsSet(currState, LacpStateActivityBit) {
			mode = LacpModeActive
		}
	}
	return mode
}
