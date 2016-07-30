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
// dr.go
package drcp

import (
	"fmt"
	"github.com/google/gopacket/layers"
	"l2/lacp/protocol/lacp"
	"net"
)

var DistributedRelayDB map[string]*DistributedRelay
var DistributedRelayDBList []*DistributedRelay

// 802.1ax-2014 7.4.1.1
type DistributedRelay struct {
	DistributedRelayFunction
	DrniId          uint32
	DrniDescription string
	DrniName        string

	// Also defined in 9.4.7
	DrniAggregatorId              [6]uint8
	DrniAggregatorPriority        uint16
	DrniPortalAddr                net.HwAddress
	DrniPortalPriority            uint16
	DrniThreeSystemPortal         bool
	DrniPortConversationPasses    [MAX_CONVERSATION_IDS]bool
	DrniGatewayConversationPasses [MAX_CONVERSATION_IDS]bool
	// End also defined in 9.4.7

	DrniPortalSystemNumber  uint8                // 1-3
	DrniIntraPortalLinkList [MAX_IPP_LINKS]int32 // ifindex
	DrniAggregator          int32
	DrniConvAdminGateway    [MAX_GATEWAY_CONVERSATIONS]int32
	// conversation id -> gateway
	DrniNeighborAdminConvGatewayListDigest []Md5Digest
	DrniNeighborAdminConvPortListDigest    []Md5Digest
	DrniGatwayAlg                          GatewayAlgorithm
	DrniNeighborAdminGatewayAlgorithm      GatewayAlgorithm
	DrniNeighborAdminPortAlgorithm         GatewayAlgorithm
	DrniNeighborAdminDRCPState             uint8
	DrniEncapMethod                        EncapMethod
	DrniIPLEncapMap                        map[uint32]uint32
	DrniNetEncapMap                        map[uint32]uint32
	DrniPSI                                bool
	DrniPortConversationControl            bool
	DrniPortalPortProtocolIDA              net.HwAddress

	// 9.4.10
	PortConversationUpdate    bool
	IppAllPortUpdate          bool
	GatewayConversationUpdate bool

	// channel used to wait on response from distributed event send
	drEvtResponseChan chan string

	a *lacp.LaAggregator

	// state machines
	PsMachineFsm *PsMachine
	GMachineFsm  [2]*GMachine
	AMachineFsm  [2]*AMachine

	Ipplinks []*DRCPIpp
}

// 802.1ax-2014 Section 9.4.8 Per-DR Function variables
type DistributedRelayFunction struct {
	ChangeDRFPorts                                bool
	ChangePortal                                  bool
	DrniCommonMethods                             bool
	DrniConversationGatewayList                   [MAX_CONVERSATION_IDS]uint32
	DrniPortalSystemState                         [3]NeighborStateInfo
	DRFHomeAdminAggregatorKey                     uint16
	DRFHomeConversationGatewayListDigest          Md5Digest
	DRFHomeConversationPortListDigest             Md5Digest
	DRFHomeGatewayAlgorithm                       [4]uint8
	DRFHomeGatewayConversationMask                [MAX_CONVERSATION_IDS]bool
	DRFHomeGatewaySequence                        uint16
	DRFHomePortAlgorithm                          [4]uint8
	DRFHomeOperAggregatorKey                      uint16
	DRFHomeOperPartnerAggregatorKey               uint16
	DRFHomeState                                  NeighborStateInfo
	DRFNeighborAdminConversationGatewayListDigest Md5Digest
	DRFNeighborAdminConversationPortListDigest    Md5Digest
	DRFNeighborAdminDRCPState                     layers.DRCPState
	DRFNeighborAdminGatewayAlgorithm              [4]uint8
	DRFNeighborAdminPortAlgorithm                 [4]uint8
	// range 1..3
	DRFPortalSystemNumber uint8
	PSI                   bool

	// 9.3.3.2
	DrniPortalSystemGatewayConversation [4096]bool
	DrniPortalSystemPortConversation    [4096]bool
}

func NewDistributedRelay(cfg *DistrubtedRelayConfig) {
	dr := &DistributedRelay{
		DrniName:                               cfg.aDrniName,
		DrniPortalAddr:                         cfg.aDrniPortalAddress,
		DrniPortalPriority:                     cfg.aDrniPortalPriority,
		DrniThreeSystemPortal:                  cfg.aDrniThreePortalSystem,
		DrniPortalSystemNumber:                 cfg.aDrniPortalSystemNumber,
		DrniIntraPortalLinkList:                cfg.aDrniIntraPortalLinkList,
		DrniAggregator:                         cfg.aDrniAggregator,
		DrniConvAdminGateway:                   cfg.aDrniConvAdminGateway,
		DrniNeighborAdminConvGatewayListDigest: cfg.aDrniNeighborAdminConvGatewayListDigest,
		DrniNeighborAdminConvPortListDigest:    cfg.aDrniNeighborAdminConvPortListDigest,
		DrniGatwayAlg:                          cfg.aDrniGatewayAlgorithm,
		DrniNeighborAdminGatewayAlgorithm:      cfg.aDrniNeighborGatewayAlgorithm,
		DrniNeighborAdminPortAlgorithm:         cfg.aDrniNeighborPortAlgorithm,
		DrniNeighborAdminDRCPState:             cfg.aDrniNeighborAdminDRCPState,
		DrniEncapMethod:                        cfg.aDrniEncapMethod,
		DrniPortConversationControl:            cfg.aDrniPortConversationControl,
	}

	for i, data := range cfg.aDrniIPLEncapMap {
		dr.DrniIPLEncapMap[i] = data
	}
	for i, data := range cfg.aDrniNetEncapMap {
		dr.DrniNetEncapMap[i] = data
	}

	netMac, _ := net.ParseMAC(cfg.aDrniIntraPortalPortProtocolDA)
	dr.DrniPortalPortProtocolIDA = netMac

	for _, ipp := range dr.DrniIntraPortalLinkList {
		ipplink := NewDRCPIpp(ipp)
		dr.Ipplinks = append(dr.Ipplinks, ipplink)
	}

}

func (dr *DistributedRelay) BEGIN(restart bool) {

	mEvtChan := make([]chan utils.MachineEvent, 0)
	evt := make([]utils.MachineEvent, 0)

	// there is a case in which we have only called
	// restart and called main functions outside
	// of this scope (TEST for example)
	//prevBegin := p.begin

	// System in being initalized
	//p.begin = true

	if !restart {
		// start all the State machines
		// Order here matters as Rx machine
		// will send event to Mux machine
		// thus machine must be up and
		// running first
		// Mux Machine
		//p.LacpMuxMachineMain()
		// Periodic Tx Machine
		//p.LacpPtxMachineMain()
		// Churn Detection Machine
		//p.LacpActorCdMachineMain()
		// Partner Churn Detection Machine
		//p.LacpPartnerCdMachineMain()
		// Rx Machine
		//p.LacpRxMachineMain()
		// Tx Machine
		//p.LacpTxMachineMain()
		// Marker Responder
		//p.LampMarkerResponderMain()
	}

	// wait group used when stopping all the
	// State mahines associated with this port.
	// want to ensure that all routines are stopped
	// before proceeding with cleanup thus why not
	// create the wg as part of a BEGIN process
	// 1) Rx Machine
	// 2) Tx Machine
	// 3) Mux Machine
	// 4) Periodic Tx Machine
	// 5) Churn Detection Machine * 2
	// 6) Marker Responder
	// Rxm
	/*
		if p.RxMachineFsm != nil {
			mEvtChan = append(mEvtChan, p.RxMachineFsm.RxmEvents)
			evt = append(evt, utils.MachineEvent{
				E:   LacpRxmEventBegin,
				Src: PortConfigModuleStr})
			if !restart || !prevBegin {
				p.wg.Add(1)
			}
		}

		// Ptxm
		if p.PtxMachineFsm != nil {
			mEvtChan = append(mEvtChan, p.PtxMachineFsm.PtxmEvents)
			evt = append(evt, utils.MachineEvent{
				E:   LacpPtxmEventBegin,
				Src: PortConfigModuleStr})
			if !restart || !prevBegin {
				p.wg.Add(1)
			}
		}
		// Cdm
		if p.CdMachineFsm != nil {
			mEvtChan = append(mEvtChan, p.CdMachineFsm.CdmEvents)
			evt = append(evt, utils.MachineEvent{
				E:   LacpCdmEventBegin,
				Src: PortConfigModuleStr})
			if !restart || !prevBegin {
				p.wg.Add(1)
			}
		}
		// Cdm
		if p.PCdMachineFsm != nil {
			mEvtChan = append(mEvtChan, p.PCdMachineFsm.CdmEvents)
			evt = append(evt, utils.MachineEvent{
				E:   LacpCdmEventBegin,
				Src: PortConfigModuleStr})
			if !restart || !prevBegin {
				p.wg.Add(1)
			}
		}
		// Muxm
		if p.MuxMachineFsm != nil {
			mEvtChan = append(mEvtChan, p.MuxMachineFsm.MuxmEvents)
			evt = append(evt, utils.MachineEvent{
				E:   LacpMuxmEventBegin,
				Src: PortConfigModuleStr})
			if !restart || !prevBegin {
				p.wg.Add(1)
			}
		}
		// Txm
		if p.TxMachineFsm != nil {
			mEvtChan = append(mEvtChan, p.TxMachineFsm.TxmEvents)
			evt = append(evt, utils.MachineEvent{
				E:   LacpTxmEventBegin,
				Src: PortConfigModuleStr})
			if !restart || !prevBegin {
				p.wg.Add(1)
			}
		}
		// Marker Responder
		if p.MarkerResponderFsm != nil {
			mEvtChan = append(mEvtChan, p.MarkerResponderFsm.LampMarkerResponderEvents)
			evt = append(evt, utils.MachineEvent{
				E:   LampMarkerResponderEventBegin,
				Src: PortConfigModuleStr})
			if !restart || !prevBegin {
				p.wg.Add(1)
			}
		}
		// call the begin event for each
		// distribute the port disable event to various machines
		p.DistributeMachineEvents(mEvtChan, evt, true)
	*/
}

// DistributeMachineEvents will distribute the events in parrallel
// to each machine
func (dr *DistributedRelay) DistributeMachineEvents(mec []chan utils.MachineEvent, e []utils.MachineEvent, waitForResponse bool) {

	length := len(mec)
	if len(mec) != len(e) {
		dr.LaDrLog("LADR: Distributing of events failed")
		return
	}

	// send all begin events to each machine in parrallel
	for j := 0; j < length; j++ {
		go func(port *DRCPIpp, w bool, idx int, machineEventChannel []chan utils.MachineEvent, event []utils.MachineEvent) {
			if w {
				event[idx].ResponseChan = p.portChan
			}
			event[idx].Src = PortConfigModuleStr
			machineEventChannel[idx] <- event[idx]
		}(p, waitForResponse, j, mec, e)
	}

	if waitForResponse {
		i := 0
		// lets wait for all the machines to respond
		for {
			select {
			case mStr := <-dr.drEvtResponseChan:
				i++
				dr.LaDrLog(strings.Join([]string{"LADR:", mStr, "response received"}, " "))
				//fmt.Println("LAPORT: Waiting for response Delayed", length, "curr", i, time.Now())
				if i >= length {
					// 10/24/15 fixed hack by sending response after Machine.ProcessEvent
					// HACK, found that port is pre-empting the State machine callback return
					// lets delay for a short period to allow for event to be received
					// and other routines to process their events
					/*
						if p.logEna {
							time.Sleep(time.Millisecond * 3)
						} else {
							time.Sleep(time.Millisecond * 1)
						}
					*/
					return
				}
			}
		}
	}
}

// 802.1ax-2014 9.3.4.4
func extractGatewayConversationID() {

}

// 802.1ax-2014 9.3.4.4
func extractPortConversationID() {

}
