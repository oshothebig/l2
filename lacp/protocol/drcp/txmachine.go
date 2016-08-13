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

// TX MACHINE, this is not really a State machine but going to create a sort of
// State machine to processes events
// TX Machine is described in 802.1ax-2014 6.4.16
package drcp

import (
	"fmt"
	"github.com/google/gopacket/layers"
	"l2/lacp/protocol/utils"
	"strconv"
	"strings"
	"utils/fsm"
)

const TxMachineModuleStr = "DRCP Tx Machine"

const (
	TxmStateNone = iota + 1
	TxmStateOn
	TxmStateOff
)

var TxmStateStrMap map[fsm.State]string

func TxMachineStrStateMapCreate() {

	TxmStateStrMap = make(map[fsm.State]string)
	TxmStateStrMap[TxmStateNone] = "None"
	TxmStateStrMap[TxmStateOn] = "On"
	TxmStateStrMap[TxmStateOff] = "Off"

}

const (
	TxmEventBegin = iota + 1
	TxmEventUnconditionalFallThrough
	TxmEventNtt
	TxmEventDrcpDisabled
	TxmEventDrcpEnabled
)

// TxMachine holds FSM and current State
// and event channels for State transitions
type TxMachine struct {
	// for debugging
	PreviousState fsm.State

	Machine *fsm.Machine

	// Port this Machine is associated with
	p *DRCPIpp

	// machine specific events
	TxmEvents chan utils.MachineEvent
}

// PrevState will get the previous State from the State transitions
func (txm *TxMachine) PrevState() fsm.State { return txm.PreviousState }

// PrevStateSet will set the previous State
func (txm *TxMachine) PrevStateSet(s fsm.State) { txm.PreviousState = s }

// Stop will stop all timers and close all channels
func (txm *TxMachine) Stop() {

	close(txm.TxmEvents)
}

// NewDrcpTxMachine will create a new instance of the TxMachine
func NewDrcpTxMachine(port *DRCPIpp) *TxMachine {
	txm := &TxMachine{
		p:             port,
		PreviousState: TxmStateNone,
		TxmEvents:     make(chan utils.MachineEvent, 1000)}

	port.TxMachineFsm = txm

	return txm
}

// A helpful function that lets us apply arbitrary rulesets to this
// instances State machine without reallocating the machine.
func (txm *TxMachine) Apply(r *fsm.Ruleset) *fsm.Machine {
	if txm.Machine == nil {
		txm.Machine = &fsm.Machine{}
	}

	// Assign the ruleset to be used for this machine
	txm.Machine.Rules = r
	txm.Machine.Curr = &utils.StateEvent{
		StrStateMap: TxmStateStrMap,
		LogEna:      false,
		Logger:      txm.DrcpTxmLog,
		Owner:       TxMachineModuleStr,
	}

	return txm.Machine
}

// DrcpTxMachineOn will either send a packet out
func (txm *TxMachine) DrcpTxMachineOn(m fsm.Machine, data interface{}) fsm.State {

	p := txm.p
	dr := p.dr

	// NTT must be set to tx
	if p.NTTDRCPDU &&
		p.DRCPEnabled {

		numPkts := 1
		if p.GatewayConversationTransmit &&
			p.PortConversationTransmit &&
			!dr.DrniCommonMethods {
			numPkts = 2
		}

		for pkt := 0; pkt < numPkts; pkt++ {

			// increment tx counter
			p.DRCPDUsTX++

			// Ethernet + ethertype
			pktLength := uint32(14)

			drcp := layers.DRCP{
				SubType: layers.DRCPSubProtocolDRCP,
				Version: layers.DRCPVersion1,
			}

			// Lets fill the in the various TLV's
			drcp.PortalInfo = layers.DRCPPortalInfoTlv{
				AggPriority:    dr.DrniAggregatorPriority,
				AggId:          dr.DrniAggregatorId,
				PortalPriority: dr.DrniPortalPriority,
				PortalAddr: [6]uint8{
					dr.DrniPortalAddr[0],
					dr.DrniPortalAddr[1],
					dr.DrniPortalAddr[2],
					dr.DrniPortalAddr[3],
					dr.DrniPortalAddr[4],
					dr.DrniPortalAddr[5],
				},
			}
			drcp.PortalInfo.TlvTypeLength.SetTlv(uint16(layers.DRCPTLVTypePortalInfo))
			drcp.PortalInfo.TlvTypeLength.SetLength(uint16(layers.DRCPTLVPortalInfoLength))
			pktLength += uint32(layers.DRCPTLVPortalInfoLength) + uint32(layers.DRCPTlvAndLengthSize)

			drcp.PortalConfigInfo = layers.DRCPPortalConfigurationInfoTlv{
				OperAggKey:       dr.DRFHomeOperAggregatorKey,
				PortAlgorithm:    dr.DRFHomePortAlgorithm,
				GatewayAlgorithm: dr.DRFHomeGatewayAlgorithm,
			}
			drcp.PortalConfigInfo.TopologyState.SetState(layers.DRCPTopologyStatePortalSystemNum, dr.DrniPortalSystemNumber)
			drcp.PortalConfigInfo.TlvTypeLength.SetTlv(uint16(layers.DRCPTLVTypePortalConfigInfo))
			drcp.PortalConfigInfo.TlvTypeLength.SetLength(uint16(layers.DRCPTLVPortalConfigurationInfoLength))
			// TODO set Port Digest and Gateway Digest
			pktLength += uint32(layers.DRCPTLVPortalConfigurationInfoLength) + uint32(layers.DRCPTlvAndLengthSize)

			if p.GatewayConversationTransmit &&
				p.PortConversationTransmit &&
				dr.DrniCommonMethods {
				if !dr.DrniThreeSystemPortal {
					drcp.TwoPortalPortConversationVector.TlvTypeLength.SetTlv(uint16(layers.DRCPTLV2PPortConversationVector))
					drcp.TwoPortalPortConversationVector.TlvTypeLength.SetLength(uint16(layers.DRCPTLV2PPortConversationVectorLength))
					pktLength += uint32(layers.DRCPTLV2PPortConversationVectorLength) + uint32(layers.DRCPTlvAndLengthSize)

					// lets only send the port conversation vector
					for i, j := 0, 0; i < 512; i, j = i+1, j+8 {

						if dr.DrniPortalSystemPortConversation[j] {
							drcp.TwoPortalPortConversationVector.Vector[i] = 1 << 7
						}
						if dr.DrniPortalSystemPortConversation[j+1] {
							drcp.TwoPortalPortConversationVector.Vector[i] = 1 << 6
						}
						if dr.DrniPortalSystemPortConversation[j+2] {
							drcp.TwoPortalPortConversationVector.Vector[i] = 1 << 5
						}
						if dr.DrniPortalSystemPortConversation[j+3] {
							drcp.TwoPortalPortConversationVector.Vector[i] = 1 << 4
						}
						if dr.DrniPortalSystemPortConversation[j+4] {
							drcp.TwoPortalPortConversationVector.Vector[i] = 1 << 3
						}
						if dr.DrniPortalSystemPortConversation[j+5] {
							drcp.TwoPortalPortConversationVector.Vector[i] = 1 << 2
						}
						if dr.DrniPortalSystemPortConversation[j+6] {
							drcp.TwoPortalPortConversationVector.Vector[i] = 1 << 1
						}
						if dr.DrniPortalSystemPortConversation[j+7] {
							drcp.TwoPortalPortConversationVector.Vector[i] = 1 << 0
						}
					}
				} else {
					// TODO 3P Not supported
				}
			} else if p.GatewayConversationTransmit &&
				p.PortConversationTransmit &&
				!dr.DrniCommonMethods {
				if pkt == 0 {
					if !dr.DrniThreeSystemPortal {
						drcp.TwoPortalGatewayConversationVector.TlvTypeLength.SetTlv(uint16(layers.DRCPTLV2PGatewayConversationVector))
						drcp.TwoPortalGatewayConversationVector.TlvTypeLength.SetLength(uint16(layers.DRCPTLV2PGatewayConversationVectorLength))
						pktLength += uint32(layers.DRCPTLV2PGatewayConversationVectorLength) + uint32(layers.DRCPTlvAndLengthSize)
						for i, j := 0, 0; i < 512; i, j = i+1, j+8 {

							if dr.DrniPortalSystemGatewayConversation[j] {
								drcp.TwoPortalGatewayConversationVector.Vector[i] = 1 << 7
							}
							if dr.DrniPortalSystemGatewayConversation[j+1] {
								drcp.TwoPortalGatewayConversationVector.Vector[i] = 1 << 6
							}
							if dr.DrniPortalSystemGatewayConversation[j+2] {
								drcp.TwoPortalGatewayConversationVector.Vector[i] = 1 << 5
							}
							if dr.DrniPortalSystemGatewayConversation[j+3] {
								drcp.TwoPortalGatewayConversationVector.Vector[i] = 1 << 4
							}
							if dr.DrniPortalSystemGatewayConversation[j+4] {
								drcp.TwoPortalGatewayConversationVector.Vector[i] = 1 << 3
							}
							if dr.DrniPortalSystemGatewayConversation[j+5] {
								drcp.TwoPortalGatewayConversationVector.Vector[i] = 1 << 2
							}
							if dr.DrniPortalSystemGatewayConversation[j+6] {
								drcp.TwoPortalGatewayConversationVector.Vector[i] = 1 << 1
							}
							if dr.DrniPortalSystemGatewayConversation[j+7] {
								drcp.TwoPortalGatewayConversationVector.Vector[i] = 1 << 0
							}
						}
					} else {
						// TODO 3P Not supported
					}
				} else {
					drcp.TwoPortalPortConversationVector.TlvTypeLength.SetTlv(uint16(layers.DRCPTLV2PPortConversationVector))
					drcp.TwoPortalPortConversationVector.TlvTypeLength.SetLength(uint16(layers.DRCPTLV2PPortConversationVectorLength))
					pktLength += uint32(layers.DRCPTLV2PPortConversationVectorLength) + uint32(layers.DRCPTlvAndLengthSize)
					// lets only send the port conversation vector
					if !dr.DrniThreeSystemPortal {
						for i, j := 0, 0; i < 512; i, j = i+1, j+8 {

							if dr.DrniPortalSystemPortConversation[j] {
								drcp.TwoPortalPortConversationVector.Vector[i] = 1 << 7
							}
							if dr.DrniPortalSystemPortConversation[j+1] {
								drcp.TwoPortalPortConversationVector.Vector[i] = 1 << 6
							}
							if dr.DrniPortalSystemPortConversation[j+2] {
								drcp.TwoPortalPortConversationVector.Vector[i] = 1 << 5
							}
							if dr.DrniPortalSystemPortConversation[j+3] {
								drcp.TwoPortalPortConversationVector.Vector[i] = 1 << 4
							}
							if dr.DrniPortalSystemPortConversation[j+4] {
								drcp.TwoPortalPortConversationVector.Vector[i] = 1 << 3
							}
							if dr.DrniPortalSystemPortConversation[j+5] {
								drcp.TwoPortalPortConversationVector.Vector[i] = 1 << 2
							}
							if dr.DrniPortalSystemPortConversation[j+6] {
								drcp.TwoPortalPortConversationVector.Vector[i] = 1 << 1
							}
							if dr.DrniPortalSystemPortConversation[j+7] {
								drcp.TwoPortalPortConversationVector.Vector[i] = 1 << 0
							}
						}
					} else {
						// TODO 3P Not supported
					}
				}
			} else if p.GatewayConversationTransmit {
				if !dr.DrniThreeSystemPortal {
					drcp.TwoPortalGatewayConversationVector.TlvTypeLength.SetTlv(uint16(layers.DRCPTLV2PGatewayConversationVector))
					drcp.TwoPortalGatewayConversationVector.TlvTypeLength.SetLength(uint16(layers.DRCPTLV2PGatewayConversationVectorLength))
					pktLength += uint32(layers.DRCPTLV2PGatewayConversationVectorLength) + uint32(layers.DRCPTlvAndLengthSize)
					for i, j := 0, 0; i < 512; i, j = i+1, j+8 {

						if dr.DrniPortalSystemGatewayConversation[j] {
							drcp.TwoPortalGatewayConversationVector.Vector[i] = 1 << 7
						}
						if dr.DrniPortalSystemGatewayConversation[j+1] {
							drcp.TwoPortalGatewayConversationVector.Vector[i] = 1 << 6
						}
						if dr.DrniPortalSystemGatewayConversation[j+2] {
							drcp.TwoPortalGatewayConversationVector.Vector[i] = 1 << 5
						}
						if dr.DrniPortalSystemGatewayConversation[j+3] {
							drcp.TwoPortalGatewayConversationVector.Vector[i] = 1 << 4
						}
						if dr.DrniPortalSystemGatewayConversation[j+4] {
							drcp.TwoPortalGatewayConversationVector.Vector[i] = 1 << 3
						}
						if dr.DrniPortalSystemGatewayConversation[j+5] {
							drcp.TwoPortalGatewayConversationVector.Vector[i] = 1 << 2
						}
						if dr.DrniPortalSystemGatewayConversation[j+6] {
							drcp.TwoPortalGatewayConversationVector.Vector[i] = 1 << 1
						}
						if dr.DrniPortalSystemGatewayConversation[j+7] {
							drcp.TwoPortalGatewayConversationVector.Vector[i] = 1 << 0
						}
					}
				} else {
					// TODO 3P Not supported
				}
			} else if p.PortConversationTransmit {
				if !dr.DrniThreeSystemPortal {
					drcp.TwoPortalGatewayConversationVector.TlvTypeLength.SetTlv(uint16(layers.DRCPTLV2PPortConversationVector))
					drcp.TwoPortalGatewayConversationVector.TlvTypeLength.SetLength(uint16(layers.DRCPTLV2PPortConversationVectorLength))
					pktLength += uint32(layers.DRCPTLV2PPortConversationVectorLength) + uint32(layers.DRCPTlvAndLengthSize)
					for i, j := 0, 0; i < 512; i, j = i+1, j+8 {

						if dr.DrniPortalSystemPortConversation[j] {
							drcp.TwoPortalGatewayConversationVector.Vector[i] = 1 << 7
						}
						if dr.DrniPortalSystemPortConversation[j+1] {
							drcp.TwoPortalGatewayConversationVector.Vector[i] = 1 << 6
						}
						if dr.DrniPortalSystemPortConversation[j+2] {
							drcp.TwoPortalGatewayConversationVector.Vector[i] = 1 << 5
						}
						if dr.DrniPortalSystemPortConversation[j+3] {
							drcp.TwoPortalGatewayConversationVector.Vector[i] = 1 << 4
						}
						if dr.DrniPortalSystemPortConversation[j+4] {
							drcp.TwoPortalGatewayConversationVector.Vector[i] = 1 << 3
						}
						if dr.DrniPortalSystemPortConversation[j+5] {
							drcp.TwoPortalGatewayConversationVector.Vector[i] = 1 << 2
						}
						if dr.DrniPortalSystemPortConversation[j+6] {
							drcp.TwoPortalGatewayConversationVector.Vector[i] = 1 << 1
						}
						if dr.DrniPortalSystemPortConversation[j+7] {
							drcp.TwoPortalGatewayConversationVector.Vector[i] = 1 << 0
						}

					}
				} else {
					// TODO 3P Not supported
				}
			}

			drcp.State.TlvTypeLength.SetTlv(uint16(layers.DRCPTLVTypeDRCPState))
			drcp.State.TlvTypeLength.SetLength(uint16(layers.DRCPTLVStateLength))
			drcp.State.State = dr.DRFHomeOperDRCPState
			pktLength += uint32(layers.DRCPTLVStateLength) + uint32(layers.DRCPTlvAndLengthSize)

			drcp.HomePortsInfo = layers.DRCPHomePortsInfoTlv{
				AdminAggKey:       dr.DRFHomeAdminAggregatorKey,
				OperPartnerAggKey: dr.DRFHomeOperAggregatorKey,
			}
			drcp.HomePortsInfo.TlvTypeLength.SetTlv(uint16(layers.DRCPTLVTypeHomePortsInfo))
			// length is determine by the number of active ports + 6
			homePortsInfoLength := uint32(4)
			if dr.a != nil && dr.DRAggregatorDistributedList != nil {
				for _, ifindex := range dr.DRAggregatorDistributedList {
					if ifindex > 0 {
						drcp.HomePortsInfo.ActiveHomePorts = append(drcp.HomePortsInfo.ActiveHomePorts, uint32(ifindex))
					}
				}
			}
			homePortsInfoLength += uint32(len(drcp.HomePortsInfo.ActiveHomePorts))
			drcp.HomePortsInfo.TlvTypeLength.SetLength(uint16(homePortsInfoLength))
			pktLength += uint32(homePortsInfoLength) + uint32(layers.DRCPTlvAndLengthSize)

			portMtu := uint32(utils.PortConfigMap[int32(p.Id)].Mtu)
			if (p.HomeGatewayVectorTransmit ||
				p.OtherGatewayVectorTransmit) &&
				pktLength < portMtu {
				// TODO WTF is the standard trying to say is supposed to happen here
				// Only include it if it does not make the packet exceed the MTU?
				/*HomeGatewayVector                     DRCPHomeGatewayVectorTlv
				NeighborGatewayVector                 DRCPGatewaNeighborGatewayVector		OtherGatewayVector                    DRCPOtherGatewayVectorTlv
				*/

			} else if (p.HomeGatewayVectorTransmit ||
				p.OtherGatewayVectorTransmit) &&
				pktLength > portMtu {
				txm.DrcpTxmLog(fmt.Sprintf("Unable to send packet pkt size %d exceeds MTU of IPP %d", pktLength, portMtu))
			}

			if p.DRFHomeNetworkIPLSharingMethod != [4]uint8{} {

				drcp.NetworkIPLMethod.TlvTypeLength.SetTlv(uint16(layers.DRCPTLVNetworkIPLSharingMethod))
				drcp.NetworkIPLMethod.TlvTypeLength.SetLength(uint16(layers.DRCPTLVNetworkIPLSharingMethodLength))
				drcp.NetworkIPLMethod.Method = p.DRFHomeNetworkIPLSharingMethod
				pktLength += uint32(layers.DRCPTLVNetworkIPLSharingMethodLength) + uint32(layers.DRCPTlvAndLengthSize)

				drcp.NetworkIPLEncapsulation.TlvTypeLength.SetTlv(uint16(layers.DRCPTLVNetworkIPLSharingEncapsulation))
				drcp.NetworkIPLEncapsulation.TlvTypeLength.SetLength(uint16(layers.DRCPTLVNetworkIPLSharingEncapsulationLength))
				for i := 0; i < 16; i++ {
					drcp.NetworkIPLEncapsulation.IplEncapDigest[i] = p.DRFHomeNetworkIPLIPLEncapDigest[i]
					drcp.NetworkIPLEncapsulation.NetEncapDigest[i] = p.DRFHomeNetworkIPLIPLNetEncapDigest[i]
				}
				pktLength += uint32(layers.DRCPTLVNetworkIPLSharingEncapsulationLength) + uint32(layers.DRCPTlvAndLengthSize)

			}

			drcp.Terminator.TlvTypeLength.SetTlv(uint16(layers.DRCPTLVTypeTerminator))
			drcp.Terminator.TlvTypeLength.SetLength(uint16(layers.DRCPTLVTerminatorLength))

			key := IppDbKey{
				Name:   p.Name,
				DrName: p.dr.DrniName,
			}
			// transmit the packet(s)
			for _, txfunc := range DRGlobalSystem.TxCallbacks[key] {
				txfunc(key, p.dr.DrniPortalPortProtocolIDA, drcp)
			}
		}
	}
	// Clear NTT
	p.NTTDRCPDU = false
	return TxmStateOn
}

// DrcpTxMachineOff will ensure that no packets are transmitted, typically means that
// drcp has been disabled or a packet was just transmitted
func (txm *TxMachine) DrcpTxMachineOff(m fsm.Machine, data interface{}) fsm.State {
	p := txm.p
	p.NTTDRCPDU = false
	return TxmStateOff
}

// DrcpTxMachineFSMBuild will build the State machine with callbacks
func DrcpTxMachineFSMBuild(p *DRCPIpp) *TxMachine {

	rules := fsm.Ruleset{}

	TxMachineStrStateMapCreate()

	// Instantiate a new TxMachine
	// Initial State will be a psuedo State known as "begin" so that
	// we can transition to the initalize State
	txm := NewDrcpTxMachine(p)

	//BEGIN -> TX OFF
	rules.AddRule(TxmStateNone, TxmEventBegin, txm.DrcpTxMachineOff)
	rules.AddRule(TxmStateOn, TxmEventBegin, txm.DrcpTxMachineOff)
	rules.AddRule(TxmStateOff, TxmEventBegin, txm.DrcpTxMachineOff)

	// NTT -> TX ON
	rules.AddRule(TxmStateOff, TxmEventNtt, txm.DrcpTxMachineOn)

	// UNCONDITIONAL -> TX OFF
	rules.AddRule(TxmStateOn, TxmEventUnconditionalFallThrough, txm.DrcpTxMachineOff)

	// Create a new FSM and apply the rules
	txm.Apply(&rules)

	return txm
}

// TxMachineMain:  802.1ax-2014 Section 9.4.19 DRCPDU Transmit machine
// Creation of Tx State Machine State transitions and callbacks
// and create go routine to pend on events
func (p *DRCPIpp) TxMachineMain() {

	// Build the State machine for Transmit Machine according to
	// 802.1ax Section 9.4.19 DRCPDU Transmit machine
	txm := DrcpTxMachineFSMBuild(p)
	p.wg.Add(1)

	// set the inital State
	txm.Machine.Start(txm.PrevState())

	// lets create a go routing which will wait for the specific events
	// that the TxMachine should handle.
	go func(m *TxMachine) {
		m.DrcpTxmLog("Machine Start")
		defer m.p.wg.Done()
		for {
			select {
			case event, ok := <-m.TxmEvents:
				var rv error
				if ok {
					if event.E == TxmEventNtt {
						p.NTTDRCPDU = true
					}

					rv := m.Machine.ProcessEvent(event.Src, event.E, nil)
					if rv == nil && m.Machine.Curr.CurrentState() == TxmStateOn {
						rv = m.Machine.ProcessEvent(TxMachineModuleStr, TxmEventUnconditionalFallThrough, nil)
					}
				}
				if event.ResponseChan != nil {
					utils.SendResponse(TxMachineModuleStr, event.ResponseChan)
				}
				if rv != nil {
					m.DrcpTxmLog(strings.Join([]string{error.Error(rv), event.Src, TxmStateStrMap[m.Machine.Curr.CurrentState()], strconv.Itoa(int(event.E))}, ":"))

				}
				if !ok {
					m.DrcpTxmLog("Machine End")
					return
				}
			}
		}
	}(txm)
}
