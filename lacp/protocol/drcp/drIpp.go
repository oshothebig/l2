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
// drIpp.go
package drcp

import (
	"fmt"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"l2/lacp/protocol/lacp"
	"l2/lacp/protocol/utils"
	"time"
)

// DRNI - Distributed Resilient Network Interconnect

var DRCPIppDB map[string]*DRCPIpp
var DRCPIppDBList []*DRCPIpp

const (
	MAX_CONVERSATION_IDS = 4096
)

// 802.1ax-2014 7.4.2.1.1
type DistributedRelayIPP struct {
	Name                         string
	Id                           uint32
	PortConversationPasses       [MAX_CONVERSATION_IDS]bool
	GatewayConversationDirection [MAX_CONVERSATION_IDS]bool
	AdminState                   bool
	OperState                    bool
	TimeOfLstOperChange          time.Time
}

// 802.1ax-2014 7.4.3.1.1
type DistributedRelayIPPCounters struct {
	StatId    uint32
	DRCPDUsRX uint32
	IllegalRX uint32
	DRCPDUsTX uint32
}

// 802.1ax-2014 7.4.4.1.1
type DistributedRelayIPPDebug struct {
	InfoId             uint32
	DRCPRXState        string
	LastRXTime         time.time
	DifferPortalReason string
}

type NeighborGatewayVector struct {
	Sequence uint32
	Vector   [MAX_CONVERSATION_IDS]bool
}

type NeighborStateInfo struct {
	OpState bool
	// indexed by the received Home_Gateway_Sequence in
	// increasing sequence number order
	GatewayVector []NeighborGatewayVector
	PortIdList    []uint32
}

// 802.1ax-2014 9.4.9 Per IPP Intra-Portal Variables
type DRCPIntraPortal struct {
	CCTimeShared                 bool
	CCEncTagShared               bool
	DifferConfPortal             bool
	DifferConfPortalSystemNumber bool
	DifferGatewayDigest          bool
	DifferPortDigest             bool
	DifferPortal                 bool
	// range 1..3
	DRFHomeConfNeighborPortalSystemNumber uint8
	DRFHomeNetworkIPLIPLEncapDigest       Md5Digest
	DRFHomeNetworkIPLIPLNetEncapDigest    Md5Digest
	DRFHomeNetworkIPLSharingMethod        [4]uint8
	// defines for state can be found in "github.com/google/gopacket/layers"
	DRFHomeOperDRCPState                     layers.DRCPState
	DRFNeighborAdminAggregatorKey            uint16
	DRFNeighborAggregatorId                  [6]uint8
	DRFNeighborAggregatorPriority            uint16
	DRFNeighborConversationGatewayListDigest Md5Digest
	DRFNeighborConversationPortList          Md5Digest
	DRFNeighborGatewayAlgorithm              [4]uint8
	DRFNeighborGatewayConversationMask       [MAX_GATEWAY_CONVERSATIONS]bool
	DRFNeighborGatewaySequence               uint16
	DRFNeighborNetworkIPLIPLEncapDigest      Md5Digest
	DRFNeighborNetworkIPLNetEncapDigest      Md5Digest
	DRFNeighborNetworkIPLSharingMethod       [4]uint8
	DRFNeighborOperAggregatorKey             uint16
	DRFNeighborOperPartnerAggregatorKey      uint16
	// defines for state can be found in "github.com/google/gopacket/layers"
	DRFNeighborOperDRCPState layers.DRCPState
	// range 1..3
	DRFNeighborConfPortalSystemNumber uint8
	DRFNeighborPortAlgorithm          [4]uint8
	// range 1..3
	DRFNeighborPortalSystemNumber            uint8
	DRFNeighborState                         NeighborStateInfo
	DRFOtherNeighborAdminAggregatorKey       uint16
	DRFOtherNeighborGatewayConversationMask  [MAX_GATEWAY_CONVERSATIONS]bool
	DRFOtherNeighborGatewaySequence          uint16
	DRFOtherNeighborOperPartnerAggregatorKey uint16
	DRFOtherNeighborState                    NeighborStateInfo
	DRFRcvHomeGatewayConversationMask        [MAX_GATEWAY_CONVERSATIONS]bool
	DRFRcvHomeGatewaySequence                uint16
	DRFRcvNeighborGatewayConversationMask    [MAX_GATEWAY_CONVERSATIONS]bool
	DRFRcvNeighborGatewaySequence            uint16
	DRFRcvOtherGatewayConversationMask       [MAX_GATEWAY_CONVERSATIONS]bool
	DRFRcvOtherGatewaySequence               uint16
	DrniNeighborCommonMethods                bool
	DrniNeighborGatewayConversation          [1024]uint8
	DrniNeighborPortConversation             [1024]uint8
	DrniNeighborONN                          bool
	DrniNeighborPortalAddr                   [6]uint8
	DrniNeighborPortalPriority               uint16
	DrniNeighborState                        [3]NeighborStateInfo
	// This should always be false as we will not support 3 portal system initially
	DrniNeighborThreeSystemPortal        bool
	EnabledTimeShared                    bool
	EnabledEncTagShared                  bool
	IppOtherGatewayConversation          [MAX_GATEWAY_CONVERSATIONS]uint32
	IppOtherPortConversationPortalSystem [MAX_GATEWAY_CONVERSATIONS]uint32
	IppPortEnabled                       bool
	IppPortalSystemState                 [3]NeighborStateInfo // this is probably wrong
	MissingRcvGatewayConVector           bool
	MissingRcvPortConVector              bool
	NTTDRCPDU                            bool
	ONN                                  bool

	// 9.4.10
	Begin                       bool
	DRCPEnabled                 bool
	HomeGatewayVectorTransmit   bool
	GatewayConversationTransmit bool
	IppAllGatewayUpdate         bool
	IppAllUpdate                bool
	IppGatewayUpdate            bool
	IppPortUpdate               bool
	OtherGatewayVectorTransmit  bool
	PortConversationTransmit    bool
}

type DRCPIpp struct {
	DistributedRelayIPP
	DRCPIntraPortal
	DistributedRelayIPPDebug

	// reference to the distributed relay object
	dr *DistributedRelay

	// sync creation and deletion
	wg sync.WaitGroup

	// handle used to tx packets to linux if
	handle *pcap.Handle

	// FSMs
	RxMachineFsm          *RxMachine
	PtxMachineFsm         *PtxMachine
	TxMachineFsm          *TxMachine
	NetIplShareMachineFsm *NetIplShareMachine
}

func NewDRCPIpp(id uint32, dr *DistributedRelay) *DRCPIpp {

	ipp := &DRCPIpp{
		Name:        PortConfigMap[id].Name,
		Id:          id,
		AdminState:  true,
		dr:          dr,
		DRCPEnabled: true,
	}

	DRCPIppDB[ipp.Name] = ipp
	DRCPIppDBList = append(DRCPIppDBList, ipp)

	// check the link status
	for _, client := range utils.GetAsicDPluginList() {
		ipp.OperState = client.GetPortLinkStatus(int32(ipp.id))
		ipp.IppPortEnabled = ipp.OperState
	}

	handle, err := pcap.OpenLive(ipp.Name, 65536, false, 50*time.Millisecond)
	if err != nil {
		// failure here may be ok as this may be SIM
		if !strings.Contains(ipp.Name, "SIM") {
			fmt.Println("Error creating pcap OpenLive handle for port", ipp.Id, ipp.Name, err)
		}
		return p
	}
	fmt.Println("Creating Listener for intf", ipp.Name)
	ipp.handle = handle
	src := gopacket.NewPacketSource(ipp.handle, layers.LayerTypeEthernet)
	in := src.Packets()
	// start rx routine
	DrRxMain(ipp.Id, in)
	fmt.Println("Rx Main Started for ipp link port", ipp.Id)

	// register the tx func
	//sgi.LaSysGlobalRegisterTxCallback(p.IntfNum, tx.TxViaLinuxIf)

}

// BEGIN this will send the start event to the start the state machines
func (p *DRCPIpp) BEGIN(restart bool) {

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
		p.DrcpRxMachineMain()
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
	if p.RxMachineFsm != nil {
		mEvtChan = append(mEvtChan, p.RxMachineFsm.RxmEvents)
		evt = append(evt, utils.MachineEvent{
			E:   RxmEventBegin,
			Src: DRCPConfigModuleStr})
	}
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
func (dr *DRCPIpp) DistributeMachineEvents(mec []chan utils.MachineEvent, e []utils.MachineEvent, waitForResponse bool) {

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
			event[idx].Src = DRCPConfigModuleStr
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

// ReportToManagement send events for various reason to infor management of something
// is wrong.
func (p *DRCPIpp) ReportToManagement() {

	p.Logger.Info(fmt.Sprintln("Report Failure to Management: %s", p.DifferPortalReason))
	// TODO send event
}

// updateNeighborVector will update the vector, indexed by the received
// Home_Gateway_Sequence in increasing sequence number order
func (n NeighborStateInfo) updateNeighborVector(sequence uint32, vector []bool) {

	if len(p.DRFNeighborState.GatewayVector) > 0 {
		for i, seqVector := range n.GatewayVector {
			if seqVector.Sequence == sequence {
				// overwrite the sequence
				n.GatewayVector[i] = NeighborGatewayVector{
					Sequence: sequence,
					Vector:   vector,
				}
			} else if seqVector.Sequence > sequence {
				// insert sequence/vecotor before between i and i -1
				n.GatewayVector = append(n.GatewayVector, NeighborGatewayVector{})
				copy(n.GatewayVector[i:], n.GatewayVector[i+1:])
				n.GatewayVector[i-1] = NeighborGatewayVector{
					Sequence: sequence,
					Vector:   vector,
				}
			}
		}
	} else {
		n.GatewayVector = append(n.GatewayVector, NeighborGatewayVector{
			Sequence: sequence,
			Vector:   vector})
	}
}

// getNeighborVectorGatwaySequenceIndex get the index for the entry whos
// sequence number is equal.
func (n NeighborStateInfo) getNeighborVectorGatwaySequenceIndex(sequence uint32, vector []bool) int32 {
	if len(n.GatewayVector) > 0 {
		for i, seqVector := range n.GatewayVector {
			if seqVector.Sequence == sequence {
				return i
			}
		}
	}
	return -1
}
