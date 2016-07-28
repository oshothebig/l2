//
//Copyright [2016] [SnapRoute Inc]
//
//Licensed under the Apache License, Version 2.0 (the "License");
//you may not use this file except in compliance with the License.
//You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
//	 Unless required by applicable law or agreed to in writing, software
//	 distributed under the License is distributed on an "AS IS" BASIS,
//	 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//	 See the License for the specific language governing permissions and
//	 limitations under the License.
//
// _______  __       __________   ___      _______.____    __    ____  __  .___________.  ______  __    __
// |   ____||  |     |   ____\  \ /  /     /       |\   \  /  \  /   / |  | |           | /      ||  |  |  |
// |  |__   |  |     |  |__   \  V  /     |   (----` \   \/    \/   /  |  | `---|  |----`|  ,----'|  |__|  |
// |   __|  |  |     |   __|   >   <       \   \      \            /   |  |     |  |     |  |     |   __   |
// |  |     |  `----.|  |____ /  .  \  .----)   |      \    /\    /    |  |     |  |     |  `----.|  |  |  |
// |__|     |_______||_______/__/ \__\ |_______/        \__/  \__/     |__|     |__|      \______||__|  |__|
//
// 802.1ax-2014 Section 9.4.15 DRCPDU Periodic Transmission machine
// rxmachine.go
package drcp

import (
	"github.com/google/gopacket/layers"
	"l2/lacp/protocol/utils"
	"sort"
	"time"
)

const PsMachineModuleStr = "Portal System Machine"

// drxm States
const (
	PsmStateNone = iota + 1
	PsmStatePortalSystemInitialize
	PsmStatePortalSystemUpdate
)

var PsmStateStrMap map[fsm.State]string

func PsMachineStrStateMapCreate() {
	PsmStateStrMap = make(map[fsm.State]string)
	PsmStateStrMap[PsmStateNone] = "None"
	PsmStateStrMap[PsmStatePortalSystemInitialize] = "No Periodic"
	PsmStateStrMap[PsmStatePortalSystemUpdate] = "Fast Periodic"
}

// rxm events
const (
	PsmEventBegin = iota + 1
	PsmEventChangePortal
	PsmEventChangeDRFPorts
)

// PsMachine holds FSM and current State
// and event channels for State transitions
type PsMachine struct {
	// for debugging
	PreviousState fsm.State

	Machine *fsm.Machine

	p *DRCPIpp

	// machine specific events
	PsmEvents chan utils.MachineEvent
}

func (psm *PsMachine) PrevState() fsm.State { return psm.PreviousState }

// PrevStateSet will set the previous State
func (psm *PsMachine) PrevStateSet(s fsm.State) { psm.PreviousState = s }

// Stop should clean up all resources
func (psm *PsMachine) Stop() {
	close(psm.PsmEvents)
}

// NewDrcpRxMachine will create a new instance of the LacpRxMachine
func NewDrcpPsMachine(port *DRCPIpp) *PsMachine {
	psm := &PsMachine{
		p:             port,
		PreviousState: PsmStateNone,
		PsmEvents:     make(chan MachineEvent, 10),
	}

	port.PsMachineFsm = ptxm

	return psm
}

// A helpful function that lets us apply arbitrary rulesets to this
// instances State machine without reallocating the machine.
func (psm *PsMachine) Apply(r *fsm.Ruleset) *fsm.Machine {
	if psm.Machine == nil {
		psm.Machine = &fsm.Machine{}
	}

	// Assign the ruleset to be used for this machine
	psm.Machine.Rules = r
	psm.Machine.Curr = &utils.StateEvent{
		StrStateMap: PsmStateStrMap,
		LogEna:      true,
		Logger:      psm.DrcpPsmLog,
		Owner:       PsMachineModuleStr,
	}

	return psm.Machine
}

// DrcpPsMachinePortalSystemInitialize function to be called after
// State transition to PORTAL_SYSTEM_INITIALIZE
func (psm *PsMachine) DrcpPsMachinePortalSystemInitialize(m fsm.Machine, data interface{}) fsm.State {
	psm.setDefaultPortalSystemParameters()
	psm.updateKey()
	return PsmStatePortalSystemInitialize
}

// DrcpPsMachineFastPeriodic function to be called after
// State transition to FAST_PERIODIC
func (psm *PsMachine) DrcpPsMachinePortalSystemUpdate(m fsm.Machine, data interface{}) fsm.State {
	p := psm.p

	p.ChangePortal = false
	p.ChangeDRFPorts = false
	psm.updateDRFHomeState()
	psm.updateKey()

	// next State
	return PsmStatePortalSystemUpdate
}

func DrcpPsMachineFSMBuild(p *DRCPIpp) *LacpRxMachine {

	RxMachineStrStateMapCreate()

	rules := fsm.Ruleset{}

	// Instantiate a new LacpRxMachine
	// Initial State will be a psuedo State known as "begin" so that
	// we can transition to the initalize State
	rxm := NewDrcpRxMachine(p)

	//BEGIN -> INITITIALIZE
	rules.AddRule(RxmStateNone, RxmEventBegin, rxm.DrcpRxMachineInitialize)
	rules.AddRule(RxmStateInitialize, RxmEventBegin, rxm.DrcpRxMachineInitialize)
	rules.AddRule(RxmStateExpired, RxmEventBegin, rxm.DrcpRxMachineInitialize)
	rules.AddRule(RxmStatePortalCheck, RxmEventBegin, rxm.DrcpRxMachineInitialize)
	rules.AddRule(RxmStateCompatibilityCheck, RxmEventBegin, rxm.DrcpRxMachineInitialize)
	rules.AddRule(RxmStateDefaulted, RxmEventBegin, rxm.DrcpRxMachineInitialize)
	rules.AddRule(RxmStateDiscard, RxmEventBegin, rxm.DrcpRxMachineInitialize)
	rules.AddRule(RxmStateCurrent, RxmEventBegin, rxm.DrcpRxMachineInitialize)

	// NOT IPP PORT ENABLED  > INITIALIZE
	rules.AddRule(RxmStateNone, RxmEventNotIPPPortEnabled, rxm.DrcpRxMachineInitialize)
	rules.AddRule(RxmStateInitialize, RxmEventNotIPPPortEnabled, rxm.DrcpRxMachineInitialize)
	rules.AddRule(RxmStateExpired, RxmEventNotIPPPortEnabled, rxm.DrcpRxMachineInitialize)
	rules.AddRule(RxmStatePortalCheck, RxmEventNotIPPPortEnabled, rxm.DrcpRxMachineInitialize)
	rules.AddRule(RxmStateCompatibilityCheck, RxmEventNotIPPPortEnabled, rxm.DrcpRxMachineInitialize)
	rules.AddRule(RxmStateDefaulted, RxmEventNotIPPPortEnabled, rxm.DrcpRxMachineInitialize)
	rules.AddRule(RxmStateDiscard, RxmEventNotIPPPortEnabled, rxm.DrcpRxMachineInitialize)
	rules.AddRule(RxmStateCurrent, RxmEventNotIPPPortEnabled, rxm.DrcpRxMachineInitialize)

	// NOT DRCP ENABLED  > INITIALIZE
	rules.AddRule(RxmStateNone, RxmEventNotDRCPEnabled, rxm.DrcpRxMachineInitialize)
	rules.AddRule(RxmStateInitialize, RxmEventNotDRCPEnabled, rxm.DrcpRxMachineInitialize)
	rules.AddRule(RxmStateExpired, RxmEventNotDRCPEnabled, rxm.DrcpRxMachineInitialize)
	rules.AddRule(RxmStatePortalCheck, RxmEventNotDRCPEnabled, rxm.DrcpRxMachineInitialize)
	rules.AddRule(RxmStateCompatibilityCheck, RxmEventNotDRCPEnabled, rxm.DrcpRxMachineInitialize)
	rules.AddRule(RxmStateDefaulted, RxmEventNotDRCPEnabled, rxm.DrcpRxMachineInitialize)
	rules.AddRule(RxmStateDiscard, RxmEventNotDRCPEnabled, rxm.DrcpRxMachineInitialize)
	rules.AddRule(RxmStateCurrent, RxmEventNotDRCPEnabled, rxm.DrcpRxMachineInitialize)

	// IPP PORT ENABLED AND DRCP ENABLED -> EXPIRED
	rules.AddRule(RxmStateInitialize, RxmEventIPPPortEnabledAndDRCPEnabled, rxm.DrcpRxMachineExpired)

	// DRCPDU RX -> PORTAL CHECK
	rules.AddRule(RxmStateExpired, RxmEventDRCPDURx, rxm.DrcpRxMachinePortalCheck)
	rules.AddRule(RxmStateDiscard, RxmEventDRCPDURx, rxm.DrcpRxMachinePortalCheck)
	rules.AddRule(RxmStateCurrent, RxmEventDRCPDURx, rxm.DrcpRxMachinePortalCheck)
	rules.AddRule(RxmStateDefaulted, RxmEventDRCPDURx, rxm.DrcpRxMachinePortalCheck)

	// NOT DIFFER PORTAL -> DISCARD
	rules.AddRule(RxmStatePortalCheck, RxmEventDifferPortal, rxm.DrcpRxMachineDiscard)

	// NOT DIFFER CONF PORTAL -> CURRENT
	rules.AddRule(RxmStateCompatibilityCheck, RxmEventNotDifferConfPortal, rxm.DrcpRxMachineCurrent)

	// DIFFER CONF PORTAL -> DISCARD
	rules.AddRule(RxmStateCompatibilityCheck, RxmEventDifferConfPortal, rxm.DrcpRxMachineDiscard)

	// DRCP CURRENT WHILE TIMER EXPIRED -> DEFAULTED
	rules.AddRule(RxmStateExpired, RxmEventDRCPCurrentWhileTimerExpired, rxm.DrcpRxMachineDefaulted)
	rules.AddRule(RxmStateDiscard, RxmEventDRCPCurrentWhileTimerExpired, rxm.DrcpRxMachineDefaulted)
	rules.AddRule(RxmStateCurrent, RxmEventDRCPCurrentWhileTimerExpired, rxm.DrcpRxMachineExpired)

	// Create a new FSM and apply the rules
	rxm.Apply(&rules)

	return rxm
}

// DrcpRxMachineMain:  802.1ax-2014 Figure 9-23
// Creation of Rx State Machine State transitions and callbacks
// and create go routine to pend on events
func (p *DRCPIpp) DrcpRxMachineMain() {

	// Build the State machine for Lacp Receive Machine according to
	// 802.1ax Section 6.4.12 Receive Machine
	rxm := DrcpRxMachineFSMBuild(p)
	p.wg.Add(1)

	// set the inital State
	rxm.Machine.Start(rxm.PrevState())

	// lets create a go routing which will wait for the specific events
	// that the RxMachine should handle.
	go func(m *LacpRxMachine) {
		m.LacpRxmLog("Machine Start")
		defer m.p.wg.Done()
		for {
			select {
			case <-m.currentWhileTimer.C:

				m.Machine.ProcessEvent(PtxMachineModuleStr, PtxmEventDRCPPeriodicTimerExpired, nil)

				if m.Machine.Curr.CurrentState() == PtxmStatePeriodicTx {
					if p.DRFNeighborOperDRCPState.GetState(layers.DRCPStateDRCPTimeout) == layers.DRCPLongTimeout {
						m.Machine.ProcessEvent(PtxMachineModuleStr, PtxmEventDRFNeighborOPerDRCPStateTimeoutEqualLongTimeout, nil)
					} else {
						m.Machine.ProcessEvent(PtxMachineModuleStr, PtxmEventDRFNeighborOPerDRCPStateTimeoutEqualShortTimeout, nil)
					}
				}

			case event, ok := <-m.RxmEvents:
				if ok {
					rv := m.Machine.ProcessEvent(event.src, event.e, nil)
					if rv == nil {
						p := m.p
						/* continue State transition */
						if m.Machine.Curr.CurrentState() == PtxmStateNoPeriodic {
							rv = m.Machine.ProcessEvent(RxMachineModuleStr, LacpRxmEventUnconditionalFallthrough, nil)
						}
						// post processing
						if rv == nil {
							if m.Machine.Curr.CurrentState() == PtxmStateFastPeriodic &&
								p.DRFNeighborOperDRCPState.GetState(layers.DRCPStateDRCPTimeout) == layers.DRCPLongTimeout {
								rv = m.Machine.ProcessEvent(RxMachineModuleStr, PtxmEventDRFNeighborOPerDRCPStateTimeoutEqualLongTimeout, nil)
							}
							if rv == nil &&
								m.Machine.Curr.CurrentState() == PtxmStateSlowPeriodic &&
								p.DRFNeighborOperDRCPState.GetState(layers.DRCPStateDRCPTimeout) == layers.DRCPShortTimeout {
								rv = m.Machine.ProcessEvent(RxMachineModuleStr, PtxmEventDRFNeighborOPerDRCPStateTimeoutEqualShortTimeout, nil)
							}
							if rv == nil &&
								m.Machine.Curr.CurrentState() == PtxmStatePeriodicTx {
								if p.DRFNeighborOperDRCPState.GetState(layers.DRCPStateDRCPTimeout) == layers.DRCPLongTimeout {
									rv = m.Machine.ProcessEvent(RxMachineModuleStr, PtxmEventDRFNeighborOPerDRCPStateTimeoutEqualLongTimeout, nil)
								} else {
									rv = m.Machine.ProcessEvent(RxMachineModuleStr, PtxmEventDRFNeighborOPerDRCPStateTimeoutEqualShortTimeout, nil)
								}
							}
						}
					}

					if rv != nil {
						m.DrcpPtxmLog(strings.Join([]string{error.Error(rv), event.src, RxmStateStrMap[m.Machine.Curr.CurrentState()], strconv.Itoa(int(event.e))}, ":"))
					}
				}

				// respond to caller if necessary so that we don't have a deadlock
				if event.ResponseChan != nil {
					utils.SendResponse(PtxMachineModuleStr, event.ResponseChan)
				}
				if !ok {
					m.DrcpPtxmLog("Machine End")
					return
				}
			}
		}
	}(rxm)
}

// NotifyNTTDRCPUDChange
func (ptxm PtxMachine) NotifyNTTDRCPUDChange(oldval, newval bool) {
	p := ptxm.p
	if oldval != newval &&
		p.NTTDRCPDU {
		// TODO send event to other state machine
	}
}
