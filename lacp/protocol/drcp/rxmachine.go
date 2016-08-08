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
// 802.1ax-2014 Section 9.4.14 DRCPDU Receive machine
// rxmachine.go
package drcp

import (
	//"fmt"
	"github.com/google/gopacket/layers"
	"l2/lacp/protocol/utils"
	"sort"
	"strconv"
	"strings"
	"time"
	"utils/fsm"
)

const RxMachineModuleStr = "DRCP Rx Machine"

// drxm States
const (
	RxmStateNone = iota + 1
	RxmStateInitialize
	RxmStateExpired
	RxmStatePortalCheck
	RxmStateCompatibilityCheck
	RxmStateDefaulted
	RxmStateDiscard
	RxmStateCurrent
)

var RxmStateStrMap map[fsm.State]string

func RxMachineStrStateMapCreate() {
	RxmStateStrMap = make(map[fsm.State]string)
	RxmStateStrMap[RxmStateNone] = "None"
	RxmStateStrMap[RxmStateInitialize] = "Initialize"
	RxmStateStrMap[RxmStateExpired] = "Expired"
	RxmStateStrMap[RxmStatePortalCheck] = "Portal Check"
	RxmStateStrMap[RxmStateCompatibilityCheck] = "Compatibility Check"
	RxmStateStrMap[RxmStateDefaulted] = "Defaulted"
	RxmStateStrMap[RxmStateDiscard] = "Discard"
	RxmStateStrMap[RxmStateCurrent] = "Current"
}

// rxm events
const (
	RxmEventBegin = iota + 1
	RxmEventNotIPPPortEnabled
	RxmEventNotDRCPEnabled
	RxmEventIPPPortEnabledAndDRCPEnabled
	RxmEventDRCPDURx
	RxmEventDRCPCurrentWhileTimerExpired
	RxmEventNotDifferPortal
	RxmEventDifferPortal
	RxmEventNotDifferConfPortal
	RxmEventDifferConfPortal
)

type RxDrcpPdu struct {
	pdu          *layers.DRCP
	src          string
	responseChan chan string
}

// DrcpRxMachine holds FSM and current State
// and event channels for State transitions
type RxMachine struct {
	// for debugging
	PreviousState fsm.State

	Machine *fsm.Machine

	p *DRCPIpp

	// debug log
	//log chan string

	// timer interval
	currentWhileTimerTimeout time.Duration

	// timers
	currentWhileTimer *time.Timer

	// machine specific events
	RxmEvents     chan utils.MachineEvent
	RxmPktRxEvent chan RxDrcpPdu
}

func (rxm *RxMachine) PrevState() fsm.State { return rxm.PreviousState }

// PrevStateSet will set the previous State
func (rxm *RxMachine) PrevStateSet(s fsm.State) { rxm.PreviousState = s }

// Stop should clean up all resources
func (rxm *RxMachine) Stop() {
	rxm.CurrentWhileTimerStop()

	close(rxm.RxmEvents)
	close(rxm.RxmPktRxEvent)
}

// NewDrcpRxMachine will create a new instance of the RxMachine
func NewDrcpRxMachine(port *DRCPIpp) *RxMachine {
	rxm := &RxMachine{
		p:             port,
		PreviousState: RxmStateNone,
		RxmEvents:     make(chan utils.MachineEvent, 10),
		RxmPktRxEvent: make(chan RxDrcpPdu, 100)}

	port.RxMachineFsm = rxm

	// create then stop
	rxm.CurrentWhileTimerStart()
	rxm.CurrentWhileTimerStop()

	return rxm
}

// A helpful function that lets us apply arbitrary rulesets to this
// instances State machine without reallocating the machine.
func (rxm *RxMachine) Apply(r *fsm.Ruleset) *fsm.Machine {
	if rxm.Machine == nil {
		rxm.Machine = &fsm.Machine{}
	}

	// Assign the ruleset to be used for this machine
	rxm.Machine.Rules = r
	rxm.Machine.Curr = &utils.StateEvent{
		StrStateMap: RxmStateStrMap,
		LogEna:      true,
		Logger:      rxm.DrcpRxmLog,
		Owner:       RxMachineModuleStr,
	}

	return rxm.Machine
}

// DrcpRxMachineInitialize function to be called after
// State transition to INITIALIZE
func (rxm *RxMachine) DrcpRxMachineInitialize(m fsm.Machine, data interface{}) fsm.State {
	p := rxm.p
	dr := p.dr

	// Record default params
	rxm.recordDefaultDRCPDU()

	// should not be set but lets be complete according to definition of ChangePortal
	isset := p.DRFNeighborOperDRCPState.GetState(layers.DRCPStateIPPActivity)
	if isset {
		defer rxm.NotifyChangePortalChanged(dr.ChangePortal, true)
		dr.ChangePortal = true
	}
	p.DRFNeighborOperDRCPState.ClearState(layers.DRCPStateIPPActivity)
	// next State
	return RxmStateInitialize
}

// DrcpRxMachineExpired function to be called after
// State transition to EXPIRED
func (rxm *RxMachine) DrcpRxMachineExpired(m fsm.Machine, data interface{}) fsm.State {
	p := rxm.p
	dr := p.dr

	// should not be set but lets be complete according to definition of ChangePortal
	isset := p.DRFNeighborOperDRCPState.GetState(layers.DRCPStateIPPActivity)
	if isset {
		defer rxm.NotifyChangePortalChanged(dr.ChangePortal, true)
		dr.ChangePortal = true
	}
	p.DRFNeighborOperDRCPState.ClearState(layers.DRCPStateIPPActivity)
	defer rxm.NotifyDRCPStateTimeoutChange(p.DRFNeighborOperDRCPState.GetState(layers.DRCPStateDRCPTimeout), true)
	// short timeout
	p.DRFNeighborOperDRCPState.SetState(layers.DRCPStateDRCPTimeout)
	// start the timer
	rxm.CurrentWhileTimerTimeoutSet(DrniShortTimeoutTime)
	rxm.CurrentWhileTimerStart()
	return RxmStateExpired
}

// DrcpRxMachinePortalCheck function to be called after
// State transition to PORTAL CHECK
func (rxm *RxMachine) DrcpRxMachinePortalCheck(m fsm.Machine, data interface{}) fsm.State {

	drcpPduInfo := data.(*layers.DRCP)

	rxm.recordPortalValues(drcpPduInfo)
	return RxmStatePortalCheck
}

// DrcpRxMachineCompatibilityCheck function to be called after
// State transition to COMPATIBILITY CHECK
func (rxm *RxMachine) DrcpRxMachineCompatibilityCheck(m fsm.Machine, data interface{}) fsm.State {

	drcpPduInfo := data.(*layers.DRCP)

	rxm.recordPortalConfValues(drcpPduInfo)
	return RxmStateCompatibilityCheck
}

// DrcpRxMachineDiscard function to be called after
// State transition to DISCARD
func (rxm *RxMachine) DrcpRxMachineDiscard(m fsm.Machine, data interface{}) fsm.State {

	// NO ACTIONS ACCORDING TO STANDARD
	return RxmStateDiscard
}

// DrcpRxMachineDefaulted function to be called after
// State transition to DEFAULTED
func (rxm *RxMachine) DrcpRxMachineDefaulted(m fsm.Machine, data interface{}) fsm.State {
	p := rxm.p
	dr := p.dr

	dr.DRFHomeOperDRCPState.SetState(layers.DRCPStateExpired)
	rxm.recordDefaultDRCPDU()
	p.reportToManagement()

	return RxmStateDefaulted
}

// DrcpRxMachineCurrent function to be called after
// State transition to CURRENT
func (rxm *RxMachine) DrcpRxMachineCurrent(m fsm.Machine, data interface{}) fsm.State {
	p := rxm.p
	dr := p.dr

	drcpPduInfo := data.(*layers.DRCP)

	rxm.recordNeighborState(drcpPduInfo)
	rxm.updateNTT()

	// 1 short , 0 long
	if dr.DRFHomeOperDRCPState.GetState(layers.DRCPStateDRCPTimeout) {
		rxm.CurrentWhileTimerTimeoutSet(DrniShortTimeoutTime)
	} else {
		rxm.CurrentWhileTimerTimeoutSet(DrniLongTimeoutTime)
	}

	rxm.CurrentWhileTimerStart()
	dr.DRFHomeOperDRCPState.ClearState(layers.DRCPStateExpired)

	return RxmStateCurrent
}

func DrcpRxMachineFSMBuild(p *DRCPIpp) *RxMachine {

	RxMachineStrStateMapCreate()

	rules := fsm.Ruleset{}

	// Instantiate a new RxMachine
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

	// NOT DIFFER PORTAL -> COMPATIBILITY CHECK
	rules.AddRule(RxmStatePortalCheck, RxmEventNotDifferPortal, rxm.DrcpRxMachineCompatibilityCheck)

	// DIFFER PORTAL -> DISCARD
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
	go func(m *RxMachine) {
		m.DrcpRxmLog("Machine Start")
		defer m.p.wg.Done()
		for {
			select {
			case <-m.currentWhileTimer.C:
				// special case if we have pending packets in the queue
				// by the time this expires we want to ensure the packet
				// gets processed first as this will clear/restart the timer
				if len(m.RxmPktRxEvent) == 0 {
					m.DrcpRxmLog("Current While Timer Expired")
					m.Machine.ProcessEvent(RxMachineModuleStr, RxmEventDRCPCurrentWhileTimerExpired, nil)
				}

			case event, ok := <-m.RxmEvents:
				if ok {
					rv := m.Machine.ProcessEvent(event.Src, event.E, nil)
					if rv == nil {
						/* continue State transition */
						m.processPostStates(nil)
					}

					if rv != nil {
						m.DrcpRxmLog(strings.Join([]string{error.Error(rv), event.Src, RxmStateStrMap[m.Machine.Curr.CurrentState()], strconv.Itoa(int(event.E))}, ":"))
					}
				}

				// respond to caller if necessary so that we don't have a deadlock
				if event.ResponseChan != nil {
					utils.SendResponse(RxMachineModuleStr, event.ResponseChan)
				}
				if !ok {
					m.DrcpRxmLog("Machine End")
					return
				}
			case rx, ok := <-m.RxmPktRxEvent:
				if ok {
					rv := m.Machine.ProcessEvent(RxMachineModuleStr, RxmEventDRCPDURx, rx.pdu)
					if rv == nil {
						/* continue State transition */
						m.processPostStates(rx.pdu)
					}

					// respond to caller if necessary so that we don't have a deadlock
					if rx.responseChan != nil {
						utils.SendResponse(RxMachineModuleStr, rx.responseChan)
					}
				}
			}
		}
	}(rxm)
}

// NotifyDRCPStateTimeoutChange notify the Periodic Transmit Machine of a neighbor state
// timeout change
func (rxm *RxMachine) NotifyDRCPStateTimeoutChange(oldval, newval bool) {
	p := rxm.p
	if oldval != newval {
		//layers.DRCPShortTimeout
		if newval {
			p.PtxMachineFsm.PtxmEvents <- utils.MachineEvent{
				E:   PtxmEventDRFNeighborOPerDRCPStateTimeoutEqualShortTimeout,
				Src: RxMachineModuleStr,
			}
		} else {
			//layers.DRCPLongTimeout
			p.PtxMachineFsm.PtxmEvents <- utils.MachineEvent{
				E:   PtxmEventDRFNeighborOPerDRCPStateTimeoutEqualLongTimeout,
				Src: RxMachineModuleStr,
			}
		}
	}
}

// processPostStates will check local params to see if any conditions
// are met in order to transition to next state
func (rxm *RxMachine) processPostStates(drcpPduInfo *layers.DRCP) {
	rxm.processPostStateInitialize()
	rxm.processPostStateExpired()
	rxm.processPostStatePortalCheck(drcpPduInfo)
	rxm.processPostStateCompatibilityCheck(drcpPduInfo)
	rxm.processPostStateDiscard()
	rxm.processPostStateCurrent()
	rxm.processPostStateDefaulted()
}

// processPostStateInitialize will check local params to see if any conditions
// are met in order to transition to next state
func (rxm *RxMachine) processPostStateInitialize() {
	p := rxm.p
	if rxm.Machine.Curr.CurrentState() == RxmStateInitialize &&
		p.IppPortEnabled && p.DRCPEnabled {
		rv := rxm.Machine.ProcessEvent(RxMachineModuleStr, RxmEventIPPPortEnabledAndDRCPEnabled, nil)
		if rv != nil {
			rxm.DrcpRxmLog(strings.Join([]string{error.Error(rv), RxMachineModuleStr, RxmStateStrMap[rxm.Machine.Curr.CurrentState()], strconv.Itoa(int(RxmEventIPPPortEnabledAndDRCPEnabled))}, ":"))
		} else {
			rxm.processPostStates(nil)
		}
	}
}

// processPostStateExpired will check local params to see if any conditions
// are met in order to transition to next state
func (rxm *RxMachine) processPostStateExpired() {
	// nothin to do events are triggered by rx packet or current while timer expired
}

// processPostStatePortalCheck will check local params to see if any conditions
// are met in order to transition to next state
func (rxm *RxMachine) processPostStatePortalCheck(drcpPduInfo *layers.DRCP) {
	p := rxm.p
	if rxm.Machine.Curr.CurrentState() == RxmStatePortalCheck {
		if p.DifferPortal {
			rv := rxm.Machine.ProcessEvent(RxMachineModuleStr, RxmEventDifferPortal, drcpPduInfo)
			if rv != nil {
				rxm.DrcpRxmLog(strings.Join([]string{error.Error(rv), RxMachineModuleStr, RxmStateStrMap[rxm.Machine.Curr.CurrentState()], strconv.Itoa(int(RxmEventDifferPortal))}, ":"))
			} else {
				rxm.processPostStates(drcpPduInfo)
			}
		} else {
			rv := rxm.Machine.ProcessEvent(RxMachineModuleStr, RxmEventNotDifferPortal, drcpPduInfo)
			if rv != nil {
				rxm.DrcpRxmLog(strings.Join([]string{error.Error(rv), RxMachineModuleStr, RxmStateStrMap[rxm.Machine.Curr.CurrentState()], strconv.Itoa(int(RxmEventNotDifferPortal))}, ":"))
			} else {
				rxm.processPostStates(drcpPduInfo)
			}
		}
	}
}

// processPostStateCompatibilityCheck will check local params to see if any conditions
// are met in order to transition to next state
func (rxm *RxMachine) processPostStateCompatibilityCheck(drcpPduInfo *layers.DRCP) {
	p := rxm.p
	if rxm.Machine.Curr.CurrentState() == RxmStateCompatibilityCheck {
		if p.DifferConfPortal {
			rv := rxm.Machine.ProcessEvent(RxMachineModuleStr, RxmEventDifferConfPortal, drcpPduInfo)
			if rv != nil {
				rxm.DrcpRxmLog(strings.Join([]string{error.Error(rv), RxMachineModuleStr, RxmStateStrMap[rxm.Machine.Curr.CurrentState()], strconv.Itoa(int(RxmEventDifferConfPortal))}, ":"))
			} else {
				rxm.processPostStates(drcpPduInfo)
			}
		} else {
			rv := rxm.Machine.ProcessEvent(RxMachineModuleStr, RxmEventNotDifferConfPortal, drcpPduInfo)
			if rv != nil {
				rxm.DrcpRxmLog(strings.Join([]string{error.Error(rv), RxMachineModuleStr, RxmStateStrMap[rxm.Machine.Curr.CurrentState()], strconv.Itoa(int(RxmEventNotDifferConfPortal))}, ":"))
			} else {
				rxm.processPostStates(drcpPduInfo)
			}
		}
	}
}

// processPostStateDiscard will check local params to see if any conditions
// are met in order to transition to next state
func (rxm *RxMachine) processPostStateDiscard() {
	// nothin to do events are triggered by rx packet or current while timer expired
}

// processPostStateCurrent will check local params to see if any conditions
// are met in order to transition to next state
func (rxm *RxMachine) processPostStateCurrent() {
	// nothin to do events are triggered by rx packet or current while timer expired
}

// processPostStateCurrent will check local params to see if any conditions
// are met in order to transition to next state
func (rxm *RxMachine) processPostStateDefaulted() {
	// nothin to do events are triggered by rx packet or current while timer expired
}

func (rxm *RxMachine) NotifyChangePortalChanged(oldval, newval bool) {
	p := rxm.p
	dr := p.dr
	if newval {
		dr.PsMachineFsm.PsmEvents <- utils.MachineEvent{
			E:   PsmEventChangePortal,
			Src: RxMachineModuleStr,
		}
	}
}

func (rxm *RxMachine) NotifyGatewayConversationUpdate(oldval, newval bool) {
	p := rxm.p
	dr := p.dr
	if (dr.GMachineFsm.Machine.Curr.CurrentState() == GmStateDRNIGatewayInitialize ||
		dr.GMachineFsm.Machine.Curr.CurrentState() == GmStateDRNIGatewayUpdate ||
		dr.GMachineFsm.Machine.Curr.CurrentState() == GmStatePsGatewayUpdate) &&
		newval {
		dr.GMachineFsm.GmEvents <- utils.MachineEvent{
			E:   GmEventGatewayConversationUpdate,
			Src: RxMachineModuleStr,
		}
	}
}

func (rxm *RxMachine) NotifyPortConversationUpdate(oldval, newval bool) {
	p := rxm.p
	dr := p.dr
	if (dr.AMachineFsm.Machine.Curr.CurrentState() == AmStateDRNIPortInitialize ||
		dr.AMachineFsm.Machine.Curr.CurrentState() == AmStateDRNIPortUpdate ||
		dr.AMachineFsm.Machine.Curr.CurrentState() == AmStatePsPortUpdate) &&
		newval {
		dr.AMachineFsm.AmEvents <- utils.MachineEvent{
			E:   AmEventPortConversationUpdate,
			Src: RxMachineModuleStr,
		}
	}
}

// updateNTT This function sets NTTDRCPDU to TRUE, if any of:
func (rxm *RxMachine) updateNTT() {
	p := rxm.p
	dr := p.dr
	if !dr.DRFHomeOperDRCPState.GetState(layers.DRCPStateGatewaySync) ||
		!dr.DRFHomeOperDRCPState.GetState(layers.DRCPStatePortSync) ||
		!p.DRFNeighborOperDRCPState.GetState(layers.DRCPStateGatewaySync) ||
		!p.DRFNeighborOperDRCPState.GetState(layers.DRCPStatePortSync) {
		defer p.NotifyNTTDRCPUDChange(RxMachineModuleStr, p.NTTDRCPDU, true)
		p.NTTDRCPDU = true
	}
}

// recordDefaultDRCPDU: 802.1ax Section 9.4.1.1
//
// This function sets the current Neighbor Portal System's operational
// parameter values to the default parameter values provided by the
// administrator as follows
func (rxm *RxMachine) recordDefaultDRCPDU() {
	p := rxm.p
	dr := p.dr
	a := dr.a

	p.DRFNeighborPortAlgorithm = dr.DrniNeighborAdminPortAlgorithm
	p.DRFNeighborGatewayAlgorithm = dr.DrniNeighborAdminGatewayAlgorithm
	p.DRFNeighborConversationPortListDigest = dr.DRFNeighborAdminConversationPortListDigest
	p.DRFNeighborConversationGatewayListDigest = dr.DRFNeighborAdminConversationGatewayListDigest
	p.DRFNeighborOperDRCPState = dr.DRFNeighborAdminDRCPState
	p.DRFNeighborAggregatorPriority = uint16(a.PartnerSystemPriority)
	p.DRFNeighborAggregatorId = a.PartnerSystemId
	p.DrniNeighborPortalPriority = uint16(a.PartnerSystemPriority)
	p.DrniNeighborPortalAddr = a.PartnerSystemId
	p.DRFNeighborPortalSystemNumber = p.DRFHomeConfNeighborPortalSystemNumber
	p.DRFNeighborConfPortalSystemNumber = dr.DRFPortalSystemNumber
	p.DRFNeighborState.OpState = false
	p.DRFNeighborState.GatewayVector = nil
	p.DRFNeighborState.PortIdList = nil
	p.DRFOtherNeighborState.OpState = false
	p.DRFOtherNeighborState.GatewayVector = nil
	p.DRFOtherNeighborState.PortIdList = nil
	p.DRFNeighborAdminAggregatorKey = 0
	p.DRFOtherNeighborAdminAggregatorKey = 0
	p.DRFNeighborOperPartnerAggregatorKey = 0
	p.DRFOtherNeighborOperPartnerAggregatorKey = 0
	if dr.DrniThreeSystemPortal {
		// TODO may need to chage this logic when 3P system supported
	} else {

		for i := 0; i < 1024; i++ {
			// Boolean vector set to 1
			p.DrniNeighborGatewayConversation[i] = 1
			if dr.ChangePortal {
				p.DrniNeighborPortConversation[i] = 1
			}
		}
	}
	p.CCTimeShared = false
	p.CCEncTagShared = false

	defer rxm.NotifyChangePortalChanged(dr.ChangePortal, true)
	dr.ChangePortal = true

}

// recordPortalValues: 802.1ax-2014 Section 9.4.11 Functions
//
// This function records the Neighbor’s Portal parameter values carried
// in a received DRCPDU (9.4.3.2) from an IPP, as the current operational
// parameter values for the immediate Neighbor Portal System on this IPP,
// as follows
func (rxm *RxMachine) recordPortalValues(drcpPduInfo *layers.DRCP) {
	p := rxm.p

	rxm.recordPortalValuesSavePortalInfo(drcpPduInfo)

	rxm.recordPortalValuesCompareNeighborToPortal()

	if p.DifferPortal {
		// send the error to mgmt to correct the provisioning
		p.reportToManagement()
	}
}

// recordPortalValuesSavePortalInfo records the Neighbor’s Portal parameter
// values carried in a received DRCPDU (9.4.3.2) from an IPP, as the current
// operational parameter values for the immediate Neighbor Portal System on
// this IPP,
func (rxm *RxMachine) recordPortalValuesSavePortalInfo(drcpPduInfo *layers.DRCP) {
	p := rxm.p

	p.DRFNeighborAggregatorPriority = drcpPduInfo.PortalInfo.AggPriority
	p.DRFNeighborAggregatorId = drcpPduInfo.PortalInfo.AggId
	p.DrniNeighborPortalPriority = drcpPduInfo.PortalInfo.PortalPriority
	p.DrniNeighborPortalAddr = drcpPduInfo.PortalInfo.PortalAddr

}

// recordPortalValuesCompareNeighborToPortal compares the newly updated values
// of the Neighbor Portal System to this Portal System’s expectations and if
func (rxm *RxMachine) recordPortalValuesCompareNeighborToPortal() {
	p := rxm.p
	dr := p.dr

	aggPrioEqual := p.DRFNeighborAggregatorPriority == dr.DrniAggregatorPriority
	aggIdEqual := p.DRFNeighborAggregatorId == dr.DrniAggregatorId
	portalPrioEqual := p.DrniNeighborPortalPriority == dr.DrniPortalPriority
	portalAddrEqual := (p.DrniNeighborPortalAddr[0] == dr.DrniPortalAddr[0] &&
		p.DrniNeighborPortalAddr[1] == dr.DrniPortalAddr[1] &&
		p.DrniNeighborPortalAddr[2] == dr.DrniPortalAddr[2] &&
		p.DrniNeighborPortalAddr[3] == dr.DrniPortalAddr[3] &&
		p.DrniNeighborPortalAddr[4] == dr.DrniPortalAddr[4] &&
		p.DrniNeighborPortalAddr[5] == dr.DrniPortalAddr[5])

	if aggPrioEqual &&
		aggIdEqual &&
		portalPrioEqual &&
		portalAddrEqual {
		p.DifferPortal = false
	} else {
		p.DifferPortal = true
		p.DifferPortalReason = ""
		if !aggPrioEqual {
			//fmt.Println(p.DRFNeighborAggregatorPriority, dr.DrniAggregatorPriority)
			p.DifferPortalReason += "Neighbor Aggregator Priority, "
		}
		if !aggIdEqual {
			//fmt.Println(p.DRFNeighborAggregatorId, dr.DrniAggregatorId)
			p.DifferPortalReason += "Neighbor Aggregator Id, "
		}
		if !portalPrioEqual {
			p.DifferPortalReason += "Neighbor Portal Priority, "
		}
		if !portalAddrEqual {
			p.DifferPortalReason += "Neighbor Portal Addr, "
		}
	}
}

// recordPortalConfValues: 802.1ax-2014 Section 9.4.11 Functions
//
// This function records the Neighbor Portal System’s values carried in
// the Portal Configuration Information TLV of a received DRCPDU (9.4.3.2)
// from an IPP, as the current operational parameter values for the immediate
// Neighbor Portal System on this IPP as follows
func (rxm *RxMachine) recordPortalConfValues(drcpPduInfo *layers.DRCP) {
	p := rxm.p
	dr := p.dr

	// Save config from pkt
	rxm.recordPortalConfValuesSavePortalConfInfo(drcpPduInfo)

	// which conversation vector is preset
	TwoPGatewayConverationVectorPresent := drcpPduInfo.TwoPortalGatewayConversationVector.TlvTypeLength.GetTlv() == layers.DRCPTLV2PGatewayConversationVector
	ThreePGatewayConversationVectorPresent := drcpPduInfo.ThreePortalGatewayConversationVector1.TlvTypeLength.GetTlv() == layers.DRCPTLV3PGatewayConversationVector1 &&
		drcpPduInfo.ThreePortalGatewayConversationVector2.TlvTypeLength.GetTlv() == layers.DRCPTLV3PGatewayConversationVector2
	TwoPPortConversationVectorPresent := drcpPduInfo.TwoPortalPortConversationVector.TlvTypeLength.GetTlv() == layers.DRCPTLV2PPortConversationVector
	ThreePPortConversationVectorPresent := drcpPduInfo.ThreePortalPortConversationVector1.TlvTypeLength.GetTlv() == layers.DRCPTLV3PPortConversationVector1 &&
		drcpPduInfo.ThreePortalPortConversationVector2.TlvTypeLength.GetTlv() == layers.DRCPTLV3PPortConversationVector2

	// It then compares the newly updated values of the Neighbor Portal System to this Portal
	// System’s expectations and if the comparison of
	portalSystemNumEqual := p.DRFNeighborPortalSystemNumber == p.DRFHomeConfNeighborPortalSystemNumber
	confPortalSystemNumEqual := p.DRFNeighborConfPortalSystemNumber == dr.DRFPortalSystemNumber
	threeSystemPortalEqual := p.DrniNeighborThreeSystemPortal == dr.DrniThreeSystemPortal
	commonMethodsEqual := p.DrniNeighborCommonMethods == dr.DrniCommonMethods
	operAggKeyEqual := p.DRFNeighborOperAggregatorKey&0x3fff == dr.DRFHomeOperAggregatorKey&0x3fff
	operAggKeyFullEqual := p.DRFNeighborOperAggregatorKey == dr.DRFHomeOperAggregatorKey
	portAlgorithmEqual := p.DRFNeighborPortAlgorithm == dr.DRFHomePortAlgorithm
	conversationPortListDigestEqual := p.DRFNeighborConversationPortListDigest == dr.DRFHomeConversationPortListDigest
	gatewayAlgorithmEqual := p.DRFNeighborGatewayAlgorithm == dr.DRFHomeGatewayAlgorithm
	conversationGatewayListDigestEqual := p.DRFNeighborConversationGatewayListDigest == dr.DRFHomeConversationGatewayListDigest

	// lets set this as it will be cleared later if the fields differ
	p.DifferConfPortalSystemNumber = false
	dr.ChangePortal = false
	p.MissingRcvGatewayConVector = true
	// The event post state procesing should catch this event change if it does not get
	// changed below
	p.DifferConfPortal = false

	p.DifferPortalReason = ""
	if !portalSystemNumEqual {
		p.DifferConfPortalSystemNumber = true
		p.DifferPortalReason += "Portal System Number, "
	}
	if !confPortalSystemNumEqual {
		p.DifferConfPortalSystemNumber = true
		p.DifferPortalReason += "Conf Portal System Number, "
	}
	if operAggKeyEqual {
		p.DifferConfPortal = false
		if !operAggKeyFullEqual {
			defer rxm.NotifyChangePortalChanged(dr.ChangePortal, true)
			dr.ChangePortal = true
		}
	} else {
		//fmt.Println("OperAggKey:", p.DRFNeighborOperAggregatorKey&0x3fff, dr.DRFHomeOperAggregatorKey&0x3fff)
		// The event post state procesing should catch this event change
		p.DifferConfPortal = true
		p.DifferPortalReason += "Oper Aggregator Key, "
	}

	if !threeSystemPortalEqual {
		p.DifferPortalReason += "Three System Portal, "
	}

	if !gatewayAlgorithmEqual {
		//fmt.Println("GatewayAlg:", p.DRFNeighborGatewayAlgorithm, dr.DRFHomeGatewayAlgorithm)
		p.DifferPortalReason += "Gateway Algorithm, "
	}

	if !p.DifferConfPortal &&
		(!threeSystemPortalEqual || !gatewayAlgorithmEqual) {
		for i := 0; i < 1024; i++ {
			// boolean vector
			p.DrniNeighborGatewayConversation[i] = 0xff
		}
		p.DifferGatewayDigest = true

	} else if !p.DifferConfPortal &&
		(threeSystemPortalEqual && gatewayAlgorithmEqual) {

		if conversationGatewayListDigestEqual {
			p.DifferGatewayDigest = false
			p.GatewayConversationTransmit = false
			p.MissingRcvGatewayConVector = false
		} else {
			p.DifferGatewayDigest = true
			p.GatewayConversationTransmit = true
			if TwoPGatewayConverationVectorPresent &&
				!p.dr.DrniThreeSystemPortal {
				for i := 0; i < 512; i++ {
					p.DrniNeighborGatewayConversation[i] = drcpPduInfo.TwoPortalGatewayConversationVector.Vector[i]
				}
				p.MissingRcvGatewayConVector = false
			} else if ThreePGatewayConversationVectorPresent &&
				p.dr.DrniThreeSystemPortal {
				// TODO need to change the logic to concatinate the two 3P vector
			} else if !TwoPGatewayConverationVectorPresent &&
				!ThreePGatewayConversationVectorPresent &&
				commonMethodsEqual &&
				p.dr.DrniCommonMethods &&
				(TwoPPortConversationVectorPresent ||
					ThreePPortConversationVectorPresent) {
				if TwoPPortConversationVectorPresent &&
					!p.dr.DrniThreeSystemPortal {
					for i := 0; i < 512; i++ {
						p.DrniNeighborGatewayConversation[i] = drcpPduInfo.TwoPortalPortConversationVector.Vector[i]
					}
					p.MissingRcvGatewayConVector = false
				} else if ThreePPortConversationVectorPresent &&
					p.dr.DrniThreeSystemPortal {
					// TODO need to change logic to concatinate the two 3P vector
				}
			} else {
				p.MissingRcvGatewayConVector = true
			}
		}
	}

	if !p.DifferConfPortal &&
		(!threeSystemPortalEqual || !portAlgorithmEqual) {
		if p.dr.DrniThreeSystemPortal {
			// TODO when 3P system supported
		} else {
			for i := 0; i < 1024; i++ {
				// boolean vector
				p.DrniNeighborGatewayConversation[i] = 0xff
			}
		}
	} else if !p.DifferConfPortal &&
		threeSystemPortalEqual &&
		portAlgorithmEqual {
		if conversationPortListDigestEqual {
			p.DifferPortDigest = false
			p.PortConversationTransmit = false
			p.MissingRcvPortConVector = false
		} else {
			p.DifferPortDigest = true
			p.PortConversationTransmit = true
			if TwoPPortConversationVectorPresent &&
				!p.dr.DrniThreeSystemPortal {
				for i := 0; i < 512; i++ {
					p.DrniNeighborPortConversation[i] = drcpPduInfo.TwoPortalPortConversationVector.Vector[i]
				}
				p.MissingRcvPortConVector = false
			} else if ThreePPortConversationVectorPresent &&
				p.dr.DrniThreeSystemPortal {
				// TODO when support 3P system
			} else if !TwoPPortConversationVectorPresent &&
				!ThreePPortConversationVectorPresent &&
				p.DrniNeighborCommonMethods == p.dr.DrniCommonMethods &&
				p.dr.DrniCommonMethods {
				/*
					The above if statement is meant to satisyf the following
					statement, however logicaly it does not make sense
					so will have to debug this case later to figure out what
					the logic should be

					LOGIC for this section does not make sense below based on the actions that are performed as they
					are covered above
					!TwoPPortConversationVectorPresent &&
					!ThreePPortConversationVectorPresent &&
					commonMethodsEqual &&
					p.dr.DrniCommonMethods {
					 Otherwise if no Port Conversation Vector TLVs (9.4.3.3.4, 9.4.3.3.5, 9.4.3.3.6) are
					present in the received DRCPDU, Drni_Neighbor_Common_Methods ==
					Drni_Common_Methods == TRUE OR the Port Conversation Vector TLVs
					(9.4.3.3.4, 9.4.3.3.5, 9.4.3.3.6) associated with the expected number of Portal
					Systems in the Portal are present in the received DRCPDU then;
				*/
				if !p.dr.DrniThreeSystemPortal {
					for i := 0; i < 512; i++ {
						// boolean vector
						p.DrniNeighborPortConversation[i] = drcpPduInfo.TwoPortalPortConversationVector.Vector[i]
					}
				} else {
					// TODO when 3P system supported
				}
				p.MissingRcvPortConVector = false
			}
		}
	} else {
		p.MissingRcvPortConVector = true
	}

	if p.DifferConfPortalSystemNumber {
		if !p.MissingRcvGatewayConVector {
			for i := 0; i < 1023; i += 2 {
				p.DrniNeighborGatewayConversation[i] = p.DRFHomeConfNeighborPortalSystemNumber << 6
				p.DrniNeighborGatewayConversation[i] |= p.DRFHomeConfNeighborPortalSystemNumber << 4
				p.DrniNeighborGatewayConversation[i] |= p.DRFHomeConfNeighborPortalSystemNumber << 2
				p.DrniNeighborGatewayConversation[i] |= p.DRFHomeConfNeighborPortalSystemNumber << 0
				p.DrniNeighborGatewayConversation[i+1] = p.DRFHomeConfNeighborPortalSystemNumber << 6
				p.DrniNeighborGatewayConversation[i+1] |= p.DRFHomeConfNeighborPortalSystemNumber << 4
				p.DrniNeighborGatewayConversation[i+1] |= p.DRFHomeConfNeighborPortalSystemNumber << 2
				p.DrniNeighborGatewayConversation[i+1] |= p.DRFHomeConfNeighborPortalSystemNumber << 0
			}
		} else if p.MissingRcvPortConVector {
			for i := 0; i < 1023; i += 2 {
				p.DrniNeighborPortConversation[i] = p.DRFHomeConfNeighborPortalSystemNumber << 6
				p.DrniNeighborPortConversation[i] |= p.DRFHomeConfNeighborPortalSystemNumber << 4
				p.DrniNeighborPortConversation[i] |= p.DRFHomeConfNeighborPortalSystemNumber << 2
				p.DrniNeighborPortConversation[i] |= p.DRFHomeConfNeighborPortalSystemNumber << 0
				p.DrniNeighborPortConversation[i+1] = p.DRFHomeConfNeighborPortalSystemNumber << 6
				p.DrniNeighborPortConversation[i+1] |= p.DRFHomeConfNeighborPortalSystemNumber << 4
				p.DrniNeighborPortConversation[i+1] |= p.DRFHomeConfNeighborPortalSystemNumber << 2
				p.DrniNeighborPortConversation[i+1] |= p.DRFHomeConfNeighborPortalSystemNumber << 0
			}
		}
	}
	if !commonMethodsEqual {
		p.DifferPortalReason += "Common Methods, "
	}
	if !portAlgorithmEqual {
		p.DifferPortalReason += "Port Algorithm, "
	}

	// Sharing by time would mean according to Annex G:
	// There is no agreement on symmetric Port Conversation IDs across the DRNI
	// So, lets ignore the check in this case for the ports
	// Instead of ignoring lets just always send nil digest as it is not
	// being used, thus if someone sends anything different it will fail
	//if dr.DrniEncapMethod != ENCAP_METHOD_SHARING_BY_TIME {
	if !conversationPortListDigestEqual {
		//fmt.Println("PortListDigest:", p.DRFNeighborConversationPortListDigest, dr.DRFHomeConversationPortListDigest)
		p.DifferPortalReason += "Converstaion Port List Digest, "
	}
	//}

	// There is an agreement for ownership of Gateway
	// CVID:
	// Odd VID owned by portal 1
	// Even VID owned by portal 2
	if !conversationGatewayListDigestEqual {
		//fmt.Println("GatewayListDigest:", p.DRFNeighborConversationGatewayListDigest, dr.DRFHomeConversationGatewayListDigest)
		p.DifferPortalReason += "Conversation Gateway List Digest, "
	}

}

// recordPortalConfValuesSavePortalConfInfo This function records the Neighbor
// Portal System’s values carried in the Portal Configuration Information TLV
// of a received DRCPDU (9.4.3.2) from an IPP, as the current operational
// parameter values for the immediate Neighbor Portal System on this IPP as follows
func (rxm *RxMachine) recordPortalConfValuesSavePortalConfInfo(drcpPduInfo *layers.DRCP) {
	p := rxm.p

	p.DRFNeighborPortalSystemNumber =
		uint8(drcpPduInfo.PortalConfigInfo.TopologyState.GetState(layers.DRCPTopologyStatePortalSystemNum))
	p.DRFNeighborConfPortalSystemNumber =
		uint8(drcpPduInfo.PortalConfigInfo.TopologyState.GetState(layers.DRCPTopologyStateNeighborConfPortalSystemNumber))
	p.DrniNeighborThreeSystemPortal =
		drcpPduInfo.PortalConfigInfo.TopologyState.GetState(layers.DRCPTopologyState3SystemPortal) == 1
	p.DrniNeighborCommonMethods =
		drcpPduInfo.PortalConfigInfo.TopologyState.GetState(layers.DRCPTopologyStateCommonMethods) == 1
	p.DrniNeighborONN =
		drcpPduInfo.PortalConfigInfo.TopologyState.GetState(layers.DRCPTopologyStateOtherNonNeighbor) == 1
	p.DRFNeighborOperAggregatorKey =
		drcpPduInfo.PortalConfigInfo.OperAggKey
	p.DRFNeighborPortAlgorithm =
		drcpPduInfo.PortalConfigInfo.PortAlgorithm
	p.DRFNeighborConversationPortListDigest =
		drcpPduInfo.PortalConfigInfo.PortDigest
	p.DRFNeighborGatewayAlgorithm =
		drcpPduInfo.PortalConfigInfo.GatewayAlgorithm
	p.DRFNeighborConversationGatewayListDigest =
		drcpPduInfo.PortalConfigInfo.GatewayDigest

}

// recordNeighborState: 802.1ax-2014 Section 9.4.11 Functions
//
// This function sets DRF_Neighbor_Oper_DRCP_State.IPP_Activity to TRUE and records the
//parameter values for the Drni_Portal_System_State[] and DRF_Home_Oper_DRCP_State
//carried in a received DRCPDU [item s) in 9.4.3.2] on the IPP, as the current parameter
// values for Drni_Neighbor_State[] and DRF_Neighbor_Oper_DRCP_State associated with
// this IPP respectively. In particular, the operational Boolean Gateway Vectors for
// each Portal System in the Drni_Neighbor_State[] are extracted from the received
// DRCPDU as follows
func (rxm *RxMachine) recordNeighborState(drcpPduInfo *layers.DRCP) {
	p := rxm.p
	dr := p.dr

	// lets be complete according to definition of ChangePortal
	isset := p.DRFNeighborOperDRCPState.GetState(layers.DRCPStateIPPActivity)
	if isset {
		defer rxm.NotifyChangePortalChanged(dr.ChangePortal, true)
		dr.ChangePortal = true
	}

	// Ipp Activity
	p.DRFNeighborOperDRCPState.SetState(layers.DRCPStateIPPActivity)

	// extract the neighbor system number
	neighborSystemNum := drcpPduInfo.PortalConfigInfo.TopologyState.GetState(layers.DRCPTopologyStatePortalSystemNum)

	if neighborSystemNum > 0 &&
		neighborSystemNum <= MAX_PORTAL_SYSTEM_IDS {

		homeGatewayBitSet := drcpPduInfo.State.State.GetState(layers.DRCPStateHomeGatewayBit)
		if !homeGatewayBitSet {
			// clear all entries == NULL
			p.DRFRcvNeighborGatewayConversationMask = [MAX_CONVERSATION_IDS]bool{}
		} else if homeGatewayBitSet {
			if drcpPduInfo.HomeGatewayVector.TlvTypeLength.GetTlv() == layers.DRCPTLVTypeHomeGatewayVector {
				if len(drcpPduInfo.HomeGatewayVector.Vector) == 512 {
					vector := make([]bool, MAX_CONVERSATION_IDS)
					for i, j := 0, 0; i < 512; i, j = i+1, j+8 {
						vector[j] = drcpPduInfo.HomeGatewayVector.Vector[i]>>7&0x1 == 1
						p.DRFRcvNeighborGatewayConversationMask[j] = drcpPduInfo.HomeGatewayVector.Vector[i]>>7&0x1 == 1
						vector[j+1] = drcpPduInfo.HomeGatewayVector.Vector[i]>>6&0x1 == 1
						p.DRFRcvNeighborGatewayConversationMask[j+1] = drcpPduInfo.HomeGatewayVector.Vector[i]>>6&0x1 == 1
						vector[j+2] = drcpPduInfo.HomeGatewayVector.Vector[i]>>5&0x1 == 1
						p.DRFRcvNeighborGatewayConversationMask[j+2] = drcpPduInfo.HomeGatewayVector.Vector[i]>>5&0x1 == 1
						vector[j+3] = drcpPduInfo.HomeGatewayVector.Vector[i]>>4&0x1 == 1
						p.DRFRcvNeighborGatewayConversationMask[j+3] = drcpPduInfo.HomeGatewayVector.Vector[i]>>4&0x1 == 1
						vector[j+4] = drcpPduInfo.HomeGatewayVector.Vector[i]>>3&0x1 == 1
						p.DRFRcvNeighborGatewayConversationMask[j+4] = drcpPduInfo.HomeGatewayVector.Vector[i]>>3&0x1 == 1
						vector[j+5] = drcpPduInfo.HomeGatewayVector.Vector[i]>>2&0x1 == 1
						p.DRFRcvNeighborGatewayConversationMask[j+5] = drcpPduInfo.HomeGatewayVector.Vector[i]>>2&0x1 == 1
						vector[j+6] = drcpPduInfo.HomeGatewayVector.Vector[i]>>1&0x1 == 1
						p.DRFRcvNeighborGatewayConversationMask[j+6] = drcpPduInfo.HomeGatewayVector.Vector[i]>>1&0x1 == 1
						vector[j+7] = drcpPduInfo.HomeGatewayVector.Vector[i]>>0&0x1 == 1
						p.DRFRcvNeighborGatewayConversationMask[j+7] = drcpPduInfo.HomeGatewayVector.Vector[i]>>0&0x1 == 1
					}
					dr.updateNeighborVector(drcpPduInfo.HomeGatewayVector.Sequence, vector)
					// The OtherGatewayVectorTransmit on the other IPP, if it exists and is operational, is set to
					// TRUE, which mean we need to get the other ipp
					for _, ipp := range dr.Ipplinks {
						if ipp != p && ipp.DRCPEnabled {
							ipp.OtherGatewayVectorTransmit = true
						}
					}
				}
			} else if len(drcpPduInfo.HomeGatewayVector.Vector) == 0 {
				// The OtherGatewayVectorTransmit on the other IPP, if it exists and is operational, is set to
				// TRUE
				for _, ipp := range dr.Ipplinks {
					if ipp != p && ipp.DRCPEnabled {
						ipp.OtherGatewayVectorTransmit = false
					}
					// If the tuple (Home_Gateway_Sequence, Neighbor_Gateway_Vector) is stored as the first
					// entry in the database, then
					vector := make([]bool, MAX_CONVERSATION_IDS)
					index := dr.getNeighborVectorGatwaySequenceIndex(drcpPduInfo.HomeGatewayVector.Sequence, vector)
					if index == 0 {
						for i := 0; i < MAX_CONVERSATION_IDS; i++ {
							p.DRFRcvNeighborGatewayConversationMask[i] = p.DrniNeighborState[neighborSystemNum].GatewayVector[index].Vector[i]
						}
					} else {
						for i := 0; i < MAX_CONVERSATION_IDS; i++ {
							p.DRFRcvNeighborGatewayConversationMask[i] = true
						}
					}
				}
			}
		}
		if !drcpPduInfo.State.State.GetState(layers.DRCPStateNeighborGatewayBit) {
			// clear all entries == NULL
			p.DRFRcvHomeGatewayConversationMask = [MAX_CONVERSATION_IDS]bool{}
		} else {
			vector := make([]bool, MAX_CONVERSATION_IDS)
			index := dr.getNeighborVectorGatwaySequenceIndex(drcpPduInfo.NeighborGatewayVector.Sequence, vector)
			if index != -1 {
				for i := 0; i < MAX_CONVERSATION_IDS; i++ {
					p.DRFRcvHomeGatewayConversationMask[i] = dr.GatewayVectorDatabase[index].Vector[i]
				}
				if index == 0 {
					p.HomeGatewayVectorTransmit = false
				} else {
					p.HomeGatewayVectorTransmit = true
					if drcpPduInfo.NeighborGatewayVector.Sequence > p.DRFRcvHomeGatewaySequence {
						vector := make([]bool, MAX_CONVERSATION_IDS)
						for i := 0; i < MAX_CONVERSATION_IDS; i++ {
							vector[i] = dr.GatewayVectorDatabase[index].Vector[i]
						}
						dr.updateNeighborVector(drcpPduInfo.NeighborGatewayVector.Sequence+1,
							vector)
					}
				}
			} else {
				for i := 0; i < 4096; i++ {
					p.DRFRcvNeighborGatewayConversationMask[i] = true
				}
				p.HomeGatewayVectorTransmit = true
			}
		}
		/* TODO when 3P portal system supported
		if !drcpPduInfo.State.State.GetState(layers.DRCPStateOtherGatewayBit) {
			// clear all entries == NULL
			p.DRFRcvOtherGatewayConversationMask = [MAX_CONVERSATION_IDS]bool{}
		} else {
			if drcpPduInfo.OtherGatewayVector.TlvTypeLength.GetTlv() == layers.DRCPTLVTypeOtherGatewayVector {
				if len(drcpPduInfo.OtherGatewayVector.Vector) > 0 {
					for i, j := 0, 0; i < 512; i, j = i+1, j+8 {
						p.DRFRcvOtherGatewayConversationMask[j] = drcpPduInfo.OtherGatewayVector.Vector[j]>>7&0x1 == 1
						p.DRFRcvOtherGatewayConversationMask[j+1] = drcpPduInfo.OtherGatewayVector.Vector[j]>>6&0x1 == 1
						p.DRFRcvOtherGatewayConversationMask[j+2] = drcpPduInfo.OtherGatewayVector.Vector[j]>>5&0x1 == 1
						p.DRFRcvOtherGatewayConversationMask[j+3] = drcpPduInfo.OtherGatewayVector.Vector[j]>>4&0x1 == 1
						p.DRFRcvOtherGatewayConversationMask[j+4] = drcpPduInfo.OtherGatewayVector.Vector[j]>>3&0x1 == 1
						p.DRFRcvOtherGatewayConversationMask[j+5] = drcpPduInfo.OtherGatewayVector.Vector[j]>>2&0x1 == 1
						p.DRFRcvOtherGatewayConversationMask[j+6] = drcpPduInfo.OtherGatewayVector.Vector[j]>>1&0x1 == 1
						p.DRFRcvOtherGatewayConversationMask[j+7] = drcpPduInfo.OtherGatewayVector.Vector[j]>>0&0x1 == 1
					}
					if !p.DRNINeighborONN {
						p.DrniNeighborState[neighborSystemNum].updateNeighborVector(drcpPduInfo.OtherGatewayVector.Sequence,
							drcpPduInfo.OtherGatewayVecto.Vector)
					}
				} else {
					index := dr.getNeighborVectorGatwaySequenceIndex(drcpPduInfo.OtherGatewayVector.Sequence,
						drcpPduInfo.NeighborGatewayVector.Vector)
					if index != -1 {
						p.DRFRcvOtherGatewayConversationMask = p.DRFRcvNeighborGatewayConversationMask[index]
						if index == 1 {
							p.OtherGatewayVectorTransmit = false
						} else {
							p.OtherGatewayVectorTransmit = true
						}
					} else {
						for i := 0; i < 4096; i++ {
							p.DRFRcvOtherGatewayConversationMask[i] = true
						}
						p.OtherGatewayVectorTransmit = true
					}
				}
			}
		}
		*/

		// Lets also record the following variables
		if drcpPduInfo.State.State.GetState(layers.DRCPStateHomeGatewayBit) {
			dr.DRFHomeOperDRCPState.SetState(layers.DRCPStateHomeGatewayBit)
		} else {
			dr.DRFHomeOperDRCPState.ClearState(layers.DRCPStateHomeGatewayBit)
		}
		if drcpPduInfo.HomePortsInfo.TlvTypeLength.GetTlv() == layers.DRCPTLVTypeHomePortsInfo {
			// Active_Home_Ports in the Home Ports Information TLV, carried in a
			// received DRCPDU on the IPP, are used as the current values for the DRF_Neighbor_State on
			// this IPP and are associated with the Portal System identified by DRF_Neighbor_Portal_System_Number;
			p.DrniNeighborState[neighborSystemNum].PortIdList = drcpPduInfo.HomePortsInfo.ActiveHomePorts
		}

		if drcpPduInfo.State.State.GetState(layers.DRCPStateOtherGatewayBit) {
			dr.DRFHomeOperDRCPState.SetState(layers.DRCPStateOtherGatewayBit)
		} else {
			dr.DRFHomeOperDRCPState.ClearState(layers.DRCPStateOtherGatewayBit)
		}

		rxm.compareOtherPortsInfo(drcpPduInfo)

		// Network / IPL sharing by time (9.3.2.1) is supported
		rxm.compareNetworkIPLMethod(drcpPduInfo)
		rxm.compareNetworkIPLSharingEncapsulation(drcpPduInfo)
		rxm.compareGatewayOperGatewayVector()
		rxm.comparePortIds()
	}
}

func (rxm *RxMachine) compareNetworkIPLSharingEncapsulation(drcpPduInfo *layers.DRCP) {
	p := rxm.p
	dr := p.dr
	if (dr.DrniEncapMethod == EncapMethod{0x00, 0x80, 0xC2, 0x02} ||
		dr.DrniEncapMethod == EncapMethod{0x00, 0x80, 0xC2, 0x03} ||
		dr.DrniEncapMethod == EncapMethod{0x00, 0x80, 0xC2, 0x04} ||
		dr.DrniEncapMethod == EncapMethod{0x00, 0x80, 0xC2, 0x05}) &&
		drcpPduInfo.NetworkIPLEncapsulation.TlvTypeLength.GetTlv() == layers.DRCPTLVNetworkIPLSharingEncapsulation &&
		drcpPduInfo.NetworkIPLMethod.TlvTypeLength.GetTlv() == layers.DRCPTLVNetworkIPLSharingMethod {
		p.DRFNeighborNetworkIPLSharingMethod[0] = drcpPduInfo.NetworkIPLMethod.Method[0]
		p.DRFNeighborNetworkIPLSharingMethod[1] = drcpPduInfo.NetworkIPLMethod.Method[1]
		p.DRFNeighborNetworkIPLSharingMethod[2] = drcpPduInfo.NetworkIPLMethod.Method[2]
		p.DRFNeighborNetworkIPLSharingMethod[3] = drcpPduInfo.NetworkIPLMethod.Method[3]

		p.DRFNeighborNetworkIPLIPLEncapDigest = drcpPduInfo.NetworkIPLEncapsulation.IplEncapDigest
		p.DRFNeighborNetworkIPLNetEncapDigest = drcpPduInfo.NetworkIPLEncapsulation.NetEncapDigest

		if p.DRFHomeNetworkIPLSharingMethod == p.DRFNeighborNetworkIPLSharingMethod &&
			p.DRFHomeNetworkIPLIPLEncapDigest == p.DRFNeighborNetworkIPLIPLEncapDigest &&
			p.DRFHomeNetworkIPLIPLNetEncapDigest == p.DRFNeighborNetworkIPLNetEncapDigest {
			p.CCEncTagShared = true
		} else {
			p.CCEncTagShared = false
		}
	}
}

// comapreOtehrPortsInfo function will be used to compare the Other Ports info logic
// but since we are only supporting a 2P System then this function logic is not needed
// will be a placeholder and know that it is incomplete
func (rxm *RxMachine) compareOtherPortsInfo(drcpPduInfo *layers.DRCP) {
	/*
		p := rxm.p
		if drcpPduInfo.OtherPortsInfo.TlvTypeLength.GetTlv() == layers.DRCPTLVTypeOtherPortsInfo {
			// the Other_Neighbor_Ports in the Other Ports Information TLV,
			// carried in a received DRCPDU on the IPP, are used as the current values for the
			// DRF_Other_Neighbor_State on this IPP and are associated with the Portal System identified
			// by the value assigned to the two most significant bits of the
			// DRF_Other_Neighbor_Admin_Aggregator_Key carried within the Other Ports Information
			// TLV in the received DRCPDU
			p.DRFOtherNeighborState.PortIdList = drcpPduInfo.OtherPortsInfo.NeighborPorts
		} else if p.dr.DrniThreeSystemPortal {
			p.DRFOtherNeighborState.OpState = false
			p.DRFOtherNeighborState.GatewayVector = nil
			p.DRFOtherNeighborState.PortIdList = nil
			// no Portal System state information is available on this IPP for the distant
			// Neighbor Portal System on the IPP
			// TODO what does this mean that the info is not present for now going to clear/set
			// the keys
			p.DRFNeighborAdminAggregatorKey = p.dr.DRFHomeAdminAggregatorKey
			p.DRFNeighborOperPartnerAggregatorKey = p.dr.DRFHomeOperPartnerAggregatorKey
			// TODO Document states to set to this but if we reached this if the other should be
			// set to NULL
			//p.DRFOtherNeighborAdminAggregatorKey = p.DRFOtherNeighborAdminAggregatorKey
			//p.DRFOtherNeighborOperPartnerAggregatorKey = p.DRFOtherNeighborOperPartnerAggregatorKey
			p.DRFOtherNeighborAdminAggregatorKey = 0
			p.DRFOtherNeighborOperPartnerAggregatorKey = 0
		}
	*/
}

func (rxm *RxMachine) compareNetworkIPLMethod(drcpPduInfo *layers.DRCP) {
	p := rxm.p
	if drcpPduInfo.NetworkIPLMethod.TlvTypeLength.GetTlv() == layers.DRCPTLVNetworkIPLSharingMethod {
		p.DRFNeighborNetworkIPLSharingMethod = drcpPduInfo.NetworkIPLMethod.Method
		if p.DRFNeighborNetworkIPLSharingMethod == p.DRFHomeNetworkIPLSharingMethod &&
			p.DRFHomeNetworkIPLSharingMethod == [4]uint8{0x00, 0x80, 0xC2, 0x01} {
			p.CCTimeShared = true
		} else {
			p.CCTimeShared = false
		}
	}
}

func (rxm *RxMachine) compareGatewayOperGatewayVector() {

	p := rxm.p
	dr := p.dr

	operOrVectorDiffer := false
	for i := 1; i <= MAX_PORTAL_SYSTEM_IDS && !operOrVectorDiffer; i++ {
		if dr.DrniPortalSystemState[i].OpState != p.DrniNeighborState[i].OpState {
			operOrVectorDiffer = true
		} else if dr.DrniPortalSystemState[i].OpState {
			if len(dr.DrniPortalSystemState[i].GatewayVector) != len(p.DrniNeighborState[i].GatewayVector) {
				operOrVectorDiffer = true
			} else {
				for j, val := range dr.DrniPortalSystemState[i].GatewayVector {
					for k := 0; k < MAX_CONVERSATION_IDS && !operOrVectorDiffer; k++ {
						if val.Vector[k] != p.DrniNeighborState[i].GatewayVector[j].Vector[k] {
							operOrVectorDiffer = true
						}
					}
				}
			}
		}
	}

	if operOrVectorDiffer {
		defer rxm.NotifyGatewayConversationUpdate(dr.GatewayConversationUpdate, true)
		dr.GatewayConversationUpdate = true
		dr.DRFHomeOperDRCPState.ClearState(layers.DRCPStateGatewaySync)
		if p.MissingRcvGatewayConVector {
			for i := 0; i < 1024; i++ {
				if p.dr.DrniThreeSystemPortal {
					// TOOD need to change logic
					p.DrniNeighborGatewayConversation[i] = p.DRFHomeConfNeighborPortalSystemNumber
				} else {
					p.DrniNeighborGatewayConversation[i] = 0xff
				}
			}
		}
	} else {
		if !dr.DRFHomeOperDRCPState.GetState(layers.DRCPStateGatewaySync) {
			defer rxm.NotifyGatewayConversationUpdate(dr.GatewayConversationUpdate, true)
			dr.GatewayConversationUpdate = true
			dr.DRFHomeOperDRCPState.SetState(layers.DRCPStateGatewaySync)
		} else if p.DifferGatewayDigest {
			defer rxm.NotifyGatewayConversationUpdate(dr.GatewayConversationUpdate, true)
			dr.GatewayConversationUpdate = true
		} else {
			dr.DRFHomeOperDRCPState.SetState(layers.DRCPStateGatewaySync)
		}
	}
}

type sortPortList []uint32

// Helper Functions For sort
func (s sortPortList) Len() int           { return len(s) }
func (s sortPortList) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s sortPortList) Less(i, j int) bool { return s[i] < s[j] }

func (rxm *RxMachine) comparePortIds() {
	p := rxm.p
	dr := p.dr
	portListDiffer := false

	for i := 1; i <= MAX_PORTAL_SYSTEM_IDS && !portListDiffer; i++ {
		if len(dr.DrniPortalSystemState[i].PortIdList) == len(p.DrniNeighborState[i].PortIdList) {
			// make sure the lists are sorted
			sort.Sort(sortPortList(dr.DrniPortalSystemState[i].PortIdList))
			sort.Sort(sortPortList(p.DrniNeighborState[i].PortIdList))
			for j, val := range dr.DrniPortalSystemState[i].PortIdList {
				if val != p.DrniNeighborState[i].PortIdList[j] {
					portListDiffer = true
				}
			}
		} else {
			portListDiffer = true
		}
	}

	if portListDiffer {
		defer rxm.NotifyPortConversationUpdate(p.dr.PortConversationUpdate, true)
		p.dr.PortConversationUpdate = true
		dr.DRFHomeOperDRCPState.ClearState(layers.DRCPStatePortSync)
		if p.MissingRcvPortConVector {
			for i := 0; i < 1024; i++ {
				if p.dr.DrniThreeSystemPortal {
					// TODO when 3P system supported
				} else {
					p.DrniNeighborPortConversation[i] = 0xff
				}
			}
		}
	} else {
		if p.DifferPortDigest {
			defer rxm.NotifyPortConversationUpdate(p.dr.PortConversationUpdate, true)
			p.dr.PortConversationUpdate = true
		} else if !dr.DRFHomeOperDRCPState.GetState(layers.DRCPStatePortSync) {
			defer rxm.NotifyPortConversationUpdate(p.dr.PortConversationUpdate, true)
			p.dr.PortConversationUpdate = true
			dr.DRFHomeOperDRCPState.SetState(layers.DRCPStatePortSync)
		} else {
			dr.DRFHomeOperDRCPState.SetState(layers.DRCPStatePortSync)
		}
	}
}
