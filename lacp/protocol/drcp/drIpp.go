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
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"l2/lacp/protocol/utils"
	"strings"
	"sync"
	"time"
)

// DRNI - Distributed Resilient Network Interconnect

var DRCPIppDB map[IppDbKey]*DRCPIpp
var DRCPIppDBList []*DRCPIpp

type IppDbKey struct {
	Name   string
	DrName string
}

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
	LastRXTime         time.Time
	DifferPortalReason string
}

type GatewayVectorEntry struct {
	Sequence uint32
	// MAX_CONVERSATION_IDS
	Vector []bool
}

type StateVectorInfo struct {
	OpState bool
	// indexed by the received Home_Gateway_Sequence in
	// increasing sequence number order
	GatewayVector []GatewayVectorEntry
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
	DRFNeighborAdminAggregatorKey            uint16
	DRFNeighborAggregatorId                  [6]uint8
	DRFNeighborAggregatorPriority            uint16
	DRFNeighborConversationGatewayListDigest Md5Digest
	DRFNeighborConversationPortListDigest    Md5Digest
	DRFNeighborGatewayAlgorithm              [4]uint8
	DRFNeighborGatewayConversationMask       [MAX_CONVERSATION_IDS]bool
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
	DRFNeighborState                         StateVectorInfo
	DRFOtherNeighborAdminAggregatorKey       uint16
	DRFOtherNeighborGatewayConversationMask  [MAX_CONVERSATION_IDS]bool
	DRFOtherNeighborGatewaySequence          uint16
	DRFOtherNeighborOperPartnerAggregatorKey uint16
	DRFOtherNeighborState                    StateVectorInfo
	DRFRcvHomeGatewayConversationMask        [MAX_CONVERSATION_IDS]bool
	DRFRcvHomeGatewaySequence                uint32
	DRFRcvNeighborGatewayConversationMask    [MAX_CONVERSATION_IDS]bool
	DRFRcvNeighborGatewaySequence            uint16
	DRFRcvOtherGatewayConversationMask       [MAX_CONVERSATION_IDS]bool
	DRFRcvOtherGatewaySequence               uint16
	DrniNeighborCommonMethods                bool
	DrniNeighborGatewayConversation          [1024]uint8
	DrniNeighborPortConversation             [1024]uint8
	DrniNeighborONN                          bool
	DrniNeighborPortalAddr                   [6]uint8
	DrniNeighborPortalPriority               uint16
	DrniNeighborState                        [4]StateVectorInfo
	// This should always be false as we will not support 3 portal system initially
	DrniNeighborThreeSystemPortal        bool
	EnabledTimeShared                    bool
	EnabledEncTagShared                  bool
	IppOtherGatewayConversation          [MAX_CONVERSATION_IDS]uint32
	IppOtherPortConversationPortalSystem [MAX_CONVERSATION_IDS]uint8
	IppPortEnabled                       bool
	IppPortalSystemState                 []StateVectorInfo // this is probably wrong
	MissingRcvGatewayConVector           bool
	MissingRcvPortConVector              bool
	NTTDRCPDU                            bool
	ONN                                  bool

	// 9.4.10
	Begin                       bool
	DRCPEnabled                 bool
	HomeGatewayVectorTransmit   bool
	GatewayConversationTransmit bool
	IppAllUpdate                bool
	IppGatewayUpdate            bool
	IppPortUpdate               bool
	OtherGatewayVectorTransmit  bool
	PortConversationTransmit    bool
}

type DRCPIpp struct {
	DistributedRelayIPP
	DRCPIntraPortal
	DistributedRelayIPPCounters
	DistributedRelayIPPDebug

	// reference to the distributed relay object
	dr *DistributedRelay

	// sync creation and deletion
	wg sync.WaitGroup

	// handle used to tx packets to linux if
	handle *pcap.Handle

	// channel used to wait on response from distributed event send
	ippEvtResponseChan chan string

	// FSMs
	RxMachineFsm          *RxMachine
	PtxMachineFsm         *PtxMachine
	TxMachineFsm          *TxMachine
	NetIplShareMachineFsm *NetIplShareMachine
	IAMachineFsm          *IAMachine
	IGMachineFsm          *IGMachine
}

func NewDRCPIpp(id uint32, dr *DistributedRelay) *DRCPIpp {

	ipp := &DRCPIpp{
		DistributedRelayIPP: DistributedRelayIPP{
			Name:       utils.PortConfigMap[int32(id)].Name,
			Id:         id,
			AdminState: true,
		},
		DRCPIntraPortal: DRCPIntraPortal{
			DRCPEnabled:          true,
			IppPortalSystemState: make([]StateVectorInfo, 0),
			// neighbor system id contained in the port id
			DRFHomeConfNeighborPortalSystemNumber: uint8(id >> 16 & 0x3),
		},
		dr:                 dr,
		ippEvtResponseChan: make(chan string),
	}

	key := IppDbKey{
		Name:   ipp.Name,
		DrName: ipp.dr.DrniName,
	}

	// add port to port db
	DRCPIppDB[key] = ipp
	DRCPIppDBList = append(DRCPIppDBList, ipp)

	// check the link status
	for _, client := range utils.GetAsicDPluginList() {
		ipp.OperState = client.GetPortLinkStatus(int32(ipp.Id))
		ipp.IppPortEnabled = ipp.OperState
		fmt.Println("Initial IPP Link State", ipp.Name, ipp.IppPortEnabled)
	}

	handle, err := pcap.OpenLive(ipp.Name, 65536, false, 50*time.Millisecond)
	if err != nil {
		// failure here may be ok as this may be SIM
		if !strings.Contains(ipp.Name, "SIM") {
			ipp.LaIppLog(fmt.Sprintf("Error creating pcap OpenLive handle for port", ipp.Id, ipp.Name, err))
		}
		return ipp
	}
	fmt.Println("Creating Listener for intf", ipp.Name)
	ipp.handle = handle
	src := gopacket.NewPacketSource(ipp.handle, layers.LayerTypeEthernet)
	in := src.Packets()
	// start rx routine
	DrRxMain(uint16(ipp.Id), ipp.dr.DrniPortalAddr.String(), in)
	ipp.LaIppLog(fmt.Sprintf("Rx Main Started for ipp link port", ipp.Id))

	// register the tx func
	DRGlobalSystem.DRSystemGlobalRegisterTxCallback(key, TxViaLinuxIf)

	return ipp
}

//
func (p *DRCPIpp) DeleteDRCPIpp() {
	// stop all state machines
	p.Stop()

	// cleanup the global tables hosting the port
	key := IppDbKey{
		Name:   p.Name,
		DrName: p.dr.DrniName,
	}
	// cleanup the tables
	if _, ok := DRCPIppDB[key]; ok {
		delete(DRCPIppDB, key)
		for i, delipp := range DRCPIppDBList {
			if delipp == p {
				DRCPIppDBList = append(DRCPIppDBList[:i], DRCPIppDBList[i+1:]...)
			}
		}
	}
}

// Stop the port services and state machines
func (p *DRCPIpp) Stop() {

	key := IppDbKey{
		Name:   p.Name,
		DrName: p.dr.DrniName,
	}
	// De-register the tx function
	DRGlobalSystem.DRSystemGlobalDeRegisterTxCallback(key)

	// close rx/tx processing
	if p.handle != nil {
		p.handle.Close()
		p.LaIppLog(fmt.Sprintf("RX/TX handle closed for port", p.Id))

	}

	// Stop the State Machines

	// Ptxm
	if p.PtxMachineFsm != nil {
		p.PtxMachineFsm.Stop()
		p.PtxMachineFsm = nil
	}
	// Rxm
	if p.RxMachineFsm != nil {
		p.RxMachineFsm.Stop()
		p.RxMachineFsm = nil
	}
	// Txm
	if p.TxMachineFsm != nil {
		p.TxMachineFsm.Stop()
		p.TxMachineFsm = nil
	}
	// NetIplShare
	if p.NetIplShareMachineFsm != nil {
		p.NetIplShareMachineFsm.Stop()
		p.NetIplShareMachineFsm = nil
	}
	// IAm
	if p.IAMachineFsm != nil {
		p.IAMachineFsm.Stop()
		p.IAMachineFsm = nil
	}
	// IGm
	if p.IGMachineFsm != nil {
		p.IGMachineFsm.Stop()
		p.IGMachineFsm = nil
	}
	// lets wait for all the State machines to have stopped
	p.wg.Wait()

	close(p.ippEvtResponseChan)
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
		// Periodic Tx Machine
		p.DrcpPtxMachineMain()
		// Net/IPL Sharing Machine
		p.NetIplShareMachineMain()
		// IPP Aggregator machine
		p.DrcpIAMachineMain()
		// IPP Gateway Machine
		p.DrcpIGMachineMain()
		// Tx Machine
		p.TxMachineMain()
		// Rx Machine
		p.DrcpRxMachineMain()
	}

	// wait group used when stopping all the
	// State mahines associated with this port.
	// want to ensure that all routines are stopped
	// before proceeding with cleanup thus why not
	// create the wg as part of a BEGIN process
	// 1) Rx Machine
	// 2) Tx Machine
	// 3) Periodic Tx Machine
	// 4) Net/IPL Sharing Machine
	// 5) IPP Aggregator Machine
	// 6) IPP Gateway Machine

	// Rxm
	if p.RxMachineFsm != nil {
		mEvtChan = append(mEvtChan, p.RxMachineFsm.RxmEvents)
		evt = append(evt, utils.MachineEvent{
			E:   RxmEventBegin,
			Src: DRCPConfigModuleStr})
	}
	// Txm
	if p.TxMachineFsm != nil {
		mEvtChan = append(mEvtChan, p.TxMachineFsm.TxmEvents)
		evt = append(evt, utils.MachineEvent{
			E:   TxmEventBegin,
			Src: DRCPConfigModuleStr})
	}
	// Ptxm
	if p.PtxMachineFsm != nil {
		mEvtChan = append(mEvtChan, p.PtxMachineFsm.PtxmEvents)
		evt = append(evt, utils.MachineEvent{
			E:   PtxmEventBegin,
			Src: DRCPConfigModuleStr})
	}
	// NetIplShare
	if p.NetIplShareMachineFsm != nil {
		mEvtChan = append(mEvtChan, p.NetIplShareMachineFsm.NetIplSharemEvents)
		evt = append(evt, utils.MachineEvent{
			E:   NetIplSharemEventBegin,
			Src: DRCPConfigModuleStr})
	}
	// IAm
	if p.IAMachineFsm != nil {
		mEvtChan = append(mEvtChan, p.IAMachineFsm.IAmEvents)
		evt = append(evt, utils.MachineEvent{
			E:   IAmEventBegin,
			Src: DRCPConfigModuleStr})
	}
	// IGm
	if p.IGMachineFsm != nil {
		mEvtChan = append(mEvtChan, p.IGMachineFsm.IGmEvents)
		evt = append(evt, utils.MachineEvent{
			E:   IGmEventBegin,
			Src: DRCPConfigModuleStr})
	}

	// call the begin event for each
	// distribute the port disable event to various machines
	p.DistributeMachineEvents(mEvtChan, evt, true)

}

// DrIppLinkUp distribute link up event
func (p *DRCPIpp) DrIppLinkUp() {

	mEvtChan := make([]chan utils.MachineEvent, 0)
	evt := make([]utils.MachineEvent, 0)

	p.IppPortEnabled = true

	if p.DRCPEnabled {
		mEvtChan = append(mEvtChan, p.RxMachineFsm.RxmEvents)
		evt = append(evt, utils.MachineEvent{
			E:   RxmEventNotIPPPortEnabled,
			Src: DRCPConfigModuleStr})

		mEvtChan = append(mEvtChan, p.PtxMachineFsm.PtxmEvents)
		evt = append(evt, utils.MachineEvent{
			E:   IGmEventBegin,
			Src: DRCPConfigModuleStr})

	}
	p.DistributeMachineEvents(mEvtChan, evt, false)

}

// DrIppLinkDown distributelink down event
func (p *DRCPIpp) DrIppLinkDown() {
	mEvtChan := make([]chan utils.MachineEvent, 0)
	evt := make([]utils.MachineEvent, 0)

	p.IppPortEnabled = false

	mEvtChan = append(mEvtChan, p.RxMachineFsm.RxmEvents)
	evt = append(evt, utils.MachineEvent{
		E:   RxmEventNotIPPPortEnabled,
		Src: DRCPConfigModuleStr})

	mEvtChan = append(mEvtChan, p.PtxMachineFsm.PtxmEvents)
	evt = append(evt, utils.MachineEvent{
		E:   IGmEventBegin,
		Src: DRCPConfigModuleStr})

	p.DistributeMachineEvents(mEvtChan, evt, false)

}

// DistributeMachineEvents will distribute the events in parrallel
// to each machine
func (p *DRCPIpp) DistributeMachineEvents(mec []chan utils.MachineEvent, e []utils.MachineEvent, waitForResponse bool) {

	length := len(mec)
	if len(mec) != len(e) {
		p.LaIppLog("LADR: Distributing of events failed")
		return
	}

	// send all begin events to each machine in parrallel
	for j := 0; j < length; j++ {
		go func(port *DRCPIpp, w bool, idx int, machineEventChannel []chan utils.MachineEvent, event []utils.MachineEvent) {
			if w {
				event[idx].ResponseChan = p.ippEvtResponseChan
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
			case mStr := <-p.ippEvtResponseChan:
				i++
				p.LaIppLog(strings.Join([]string{"LADRIPP:", mStr, "response received"}, " "))
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

// NotifyNTTDRCPUDChange
func (p *DRCPIpp) NotifyNTTDRCPUDChange(src string, oldval, newval bool) {
	if oldval != newval &&
		newval {
		p.TxMachineFsm.TxmEvents <- utils.MachineEvent{
			E:   TxmEventNtt,
			Src: src,
		}
	}
}

// ReportToManagement send events for various reason to infor management of something
// is wrong.
func (p *DRCPIpp) reportToManagement() {

	p.LaIppLog(fmt.Sprintln("Report Failure to Management:", p.DifferPortalReason))
	// TODO send event
}

// DRFindPortByKey find ipp port by key
func DRFindPortByKey(key IppDbKey, p **DRCPIpp) bool {
	if ipp, ok := DRCPIppDB[key]; ok {
		*p = ipp
		return true
	}
	return false
}

// updateGatewayVector will update the vector, indexed by the received
// Home_Gateway_Sequence in increasing sequence number order
func (nsi *StateVectorInfo) updateGatewayVector(sequence uint32, vector []bool) {

	if len(nsi.GatewayVector) > 0 {
		for i, seqVector := range nsi.GatewayVector {
			if seqVector.Sequence == sequence {
				// overwrite the sequence
				nsi.GatewayVector[i] = GatewayVectorEntry{
					Sequence: sequence,
					Vector:   make([]bool, 4096),
				}
				for j, val := range vector {
					nsi.GatewayVector[i].Vector[j] = val
				}
			} else if seqVector.Sequence > sequence {
				// insert sequence/vecotor before between i and i -1
				nsi.GatewayVector = append(nsi.GatewayVector, GatewayVectorEntry{Vector: make([]bool, 4096)})
				copy(nsi.GatewayVector[i:], nsi.GatewayVector[i+1:])
				nsi.GatewayVector[i-1] = GatewayVectorEntry{
					Sequence: sequence,
					Vector:   make([]bool, 4096),
				}
				for j, val := range vector {
					nsi.GatewayVector[i-1].Vector[j] = val
				}
			}
		}
	} else {
		tmp := GatewayVectorEntry{
			Sequence: sequence,
			Vector:   make([]bool, 4096),
		}
		for j, val := range vector {
			tmp.Vector[j] = val
		}
		nsi.GatewayVector = append(nsi.GatewayVector, tmp)
	}
}

// getNeighborVectorGatwaySequenceIndex get the index for the entry whos
// sequence number is equal.
func (nsi *StateVectorInfo) getNeighborVectorGatwaySequenceIndex(sequence uint32, vector []bool) int32 {
	if len(nsi.GatewayVector) > 0 {
		for i, seqVector := range nsi.GatewayVector {
			if seqVector.Sequence == sequence {
				return int32(i)
			}
		}
	}
	return -1
}
