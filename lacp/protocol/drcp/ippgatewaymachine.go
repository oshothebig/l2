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
// 802.1ax-2014 Section 9.4.18 DRNI IPP machines
// ippgatewaymachine.go
package drcp

import (
	"l2/lacp/protocol/utils"
	"strconv"
	"strings"
	"utils/fsm"
)

const IGMachineModuleStr = "IPP Gateway Machine"

// igm States
const (
	IGmStateNone = iota + 1
	IGmStateIPPGatewayInitialize
	IGmStateIPPGatewayUpdate
)

var IGmStateStrMap map[fsm.State]string

func IGMachineStrStateMapCreate() {
	IGmStateStrMap = make(map[fsm.State]string)
	IGmStateStrMap[IGmStateNone] = "None"
	IGmStateStrMap[IGmStateIPPGatewayInitialize] = "IPP Gateway Initialize"
	IGmStateStrMap[IGmStateIPPGatewayUpdate] = "IPP Gateway Update"
}

// igm events
const (
	IGmEventBegin = iota + 1
	IGmEventGatewayUpdate
)

// IGMachine holds FSM and current State
// and event channels for State transitions
type IGMachine struct {
	// for debugging
	PreviousState fsm.State

	Machine *fsm.Machine

	p *DRCPIpp

	// machine specific events
	IGmEvents chan utils.MachineEvent
}

func (igm *IGMachine) PrevState() fsm.State { return igm.PreviousState }

// PrevStateSet will set the previous State
func (igm *IGMachine) PrevStateSet(s fsm.State) { igm.PreviousState = s }

// Stop should clean up all resources
func (igm *IGMachine) Stop() {
	close(igm.IGmEvents)
}

// NewDrcpIGMachine will create a new instance of the IGMachine
func NewDrcpIGMachine(p *DRCPIpp) *IGMachine {
	igm := &IGMachine{
		p:             p,
		PreviousState: IGmStateNone,
		IGmEvents:     make(chan utils.MachineEvent, 10),
	}

	p.IGMachineFsm = igm

	return igm
}

// A helpful function that lets us apply arbitrary rulesets to this
// instances State machine without reallocating the machine.
func (igm *IGMachine) Apply(r *fsm.Ruleset) *fsm.Machine {
	if igm.Machine == nil {
		igm.Machine = &fsm.Machine{}
	}

	// Assign the ruleset to be used for this machine
	igm.Machine.Rules = r
	igm.Machine.Curr = &utils.StateEvent{
		StrStateMap: IGmStateStrMap,
		LogEna:      true,
		Logger:      igm.DrcpIGmLog,
		Owner:       IGMachineModuleStr,
	}

	return igm.Machine
}

// DrcpGMachineIPPGatewayInitialize function to be called after
// State transition to IPP_GATEWAY_INITIALIZE
func (igm *IGMachine) DrcpIGMachineIPPGatewayInitialize(m fsm.Machine, data interface{}) fsm.State {
	p := igm.p
	igm.initializeIPPGatewayConversation()
	p.IppGatewayUpdate = false

	return IGmStateIPPGatewayInitialize
}

// DrcpIGMachineIPPGatewayUpdate function to be called after
// State transition to IPP_GATEWAY_UPDATE
func (igm *IGMachine) DrcpIGMachineIPPGatewayUpdate(m fsm.Machine, data interface{}) fsm.State {
	p := igm.p

	igm.setIPPGatewayConversation()
	igm.updateIPPGatewayConversationDirection()
	p.IppGatewayUpdate = false

	// next State
	return IGmStateIPPGatewayUpdate
}

func DrcpIGMachineFSMBuild(p *DRCPIpp) *IGMachine {

	IGMachineStrStateMapCreate()

	rules := fsm.Ruleset{}

	// Instantiate a new IGMachine
	// Initial State will be a psuedo State known as "begin" so that
	// we can transition to the initalize State
	igm := NewDrcpIGMachine(p)

	//BEGIN -> IPP GATEWAY INITIALIZE
	rules.AddRule(IGmStateNone, IGmEventBegin, igm.DrcpIGMachineIPPGatewayInitialize)
	rules.AddRule(IGmStateIPPGatewayInitialize, IGmEventBegin, igm.DrcpIGMachineIPPGatewayInitialize)
	rules.AddRule(IGmStateIPPGatewayUpdate, IGmEventBegin, igm.DrcpIGMachineIPPGatewayInitialize)

	// IPP GATEWAY UPDATE  > IPP GATEWAY UPDATE
	rules.AddRule(IGmStateIPPGatewayInitialize, IGmEventGatewayUpdate, igm.DrcpIGMachineIPPGatewayUpdate)
	rules.AddRule(IGmStateIPPGatewayUpdate, IGmEventGatewayUpdate, igm.DrcpIGMachineIPPGatewayUpdate)

	// Create a new FSM and apply the rules
	igm.Apply(&rules)

	return igm
}

// DrcpIGMachineMain:  802.1ax-2014 Figure 9-27
// Creation of DRNI IPP Gateway State Machine state transitions and callbacks
// and create go routine to pend on events
func (p *DRCPIpp) DrcpIGMachineMain() {

	// Build the State machine for  DRNI Gateway Machine according to
	// 802.1ax-2014 Section 9.4.18 DRNI IPP machines
	igm := DrcpIGMachineFSMBuild(p)
	p.wg.Add(1)

	// set the inital State
	igm.Machine.Start(igm.PrevState())

	// lets create a go routing which will wait for the specific events
	// that the IGMachine should handle.
	go func(m *IGMachine) {
		m.DrcpIGmLog("Machine Start")
		defer m.p.wg.Done()
		for {
			select {
			case event, ok := <-m.IGmEvents:
				if ok {
					rv := m.Machine.ProcessEvent(event.Src, event.E, nil)
					p := m.p
					// post state processing
					if rv == nil &&
						p.IppGatewayUpdate {
						rv = m.Machine.ProcessEvent(IGMachineModuleStr, IGmEventGatewayUpdate, nil)
					}

					if rv != nil {
						m.DrcpIGmLog(strings.Join([]string{error.Error(rv), event.Src, IGmStateStrMap[m.Machine.Curr.CurrentState()], strconv.Itoa(int(event.E))}, ":"))
					}
				}

				// respond to caller if necessary so that we don't have a deadlock
				if event.ResponseChan != nil {
					utils.SendResponse(IGMachineModuleStr, event.ResponseChan)
				}
				if !ok {
					m.DrcpIGmLog("Machine End")
					return
				}
			}
		}
	}(igm)
}

// initializeIPPGatewayConversation This function sets the Ipp_Gateway_Conversation_Direction to a sequence of zeros, indexed
// by Gateway Conversation ID.
func (igm *IGMachine) initializeIPPGatewayConversation() {
	p := igm.p
	for i := 0; i < MAX_CONVERSATION_IDS; i++ {
		p.GatewayConversationDirection[i] = false
	}
}

// setIPPGatewayConversation This function sets Ipp_Other_Gateway_Conversation as follows
func (igm *IGMachine) setIPPGatewayConversation() {
	p := igm.p

	if p.DifferGatewayDigest &&
		p.dr.DrniThreeSystemPortal {
		// TODO handle 3 portal system logic
	} else if p.DifferGatewayDigest &&
		!p.dr.DrniThreeSystemPortal {
		var neighborConversationSystemNumbers [1024]uint8
		for i, j := 0, 0; i < MAX_CONVERSATION_IDS; i, j = i+8, j+2 {
			if (p.DrniNeighborGatewayConversation[i] >> 7 & 1) == 0 {
				neighborConversationSystemNumbers[j] |= p.dr.DRFPortalSystemNumber << 6
			} else {
				neighborConversationSystemNumbers[j] |= p.DRFHomeConfNeighborPortalSystemNumber << 6
			}
			if (p.DrniNeighborGatewayConversation[i] >> 6 & 1) == 0 {
				neighborConversationSystemNumbers[j] |= p.dr.DRFPortalSystemNumber << 4
			} else {
				neighborConversationSystemNumbers[j] |= p.DRFHomeConfNeighborPortalSystemNumber << 4
			}
			if (p.DrniNeighborGatewayConversation[i] >> 5 & 1) == 0 {
				neighborConversationSystemNumbers[j] |= p.dr.DRFPortalSystemNumber << 2
			} else {
				neighborConversationSystemNumbers[j] |= p.DRFHomeConfNeighborPortalSystemNumber << 2
			}
			if (p.DrniNeighborGatewayConversation[i] >> 4 & 1) == 0 {
				neighborConversationSystemNumbers[j] |= p.dr.DRFPortalSystemNumber << 0
			} else {
				neighborConversationSystemNumbers[j] |= p.DRFHomeConfNeighborPortalSystemNumber << 0
			}
			if (p.DrniNeighborGatewayConversation[i] >> 3 & 1) == 0 {
				neighborConversationSystemNumbers[j+1] |= p.dr.DRFPortalSystemNumber << 6
			} else {
				neighborConversationSystemNumbers[j+1] |= p.DRFHomeConfNeighborPortalSystemNumber << 6
			}
			if (p.DrniNeighborGatewayConversation[i] >> 2 & 1) == 0 {
				neighborConversationSystemNumbers[j+1] |= p.dr.DRFPortalSystemNumber << 4
			} else {
				neighborConversationSystemNumbers[j+1] |= p.DRFHomeConfNeighborPortalSystemNumber << 4
			}
			if (p.DrniNeighborGatewayConversation[i] >> 1 & 1) == 0 {
				neighborConversationSystemNumbers[j+1] |= p.dr.DRFPortalSystemNumber << 2
			} else {
				neighborConversationSystemNumbers[j+1] |= p.DRFHomeConfNeighborPortalSystemNumber << 2
			}
			if (p.DrniNeighborGatewayConversation[i] >> 0 & 1) == 0 {
				neighborConversationSystemNumbers[j+1] |= p.dr.DRFPortalSystemNumber << 0
			} else {
				neighborConversationSystemNumbers[j+1] |= p.DRFHomeConfNeighborPortalSystemNumber << 0
			}
		}
		// now lets save the values
		p.DrniNeighborGatewayConversation = neighborConversationSystemNumbers
	} else if !p.DifferGatewayDigest {
		// TODO revisit logic
	}
}

// updateIPPGatewayConversationDirection This function computes a value for Ipp_Gateway_Conversation_Direction as follows
func (igm *IGMachine) updateIPPGatewayConversationDirection() {
	// TODO need to understand the logic better
}
