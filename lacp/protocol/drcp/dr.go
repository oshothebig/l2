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
	"bytes"
	"crypto/md5"
	"encoding/binary"
	"fmt"
	"github.com/google/gopacket/layers"
	"l2/lacp/protocol/lacp"
	"l2/lacp/protocol/utils"
	"math"
	"net"
	"strconv"
	"strings"
	"sync"
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
	DrniAggregatorId        [6]uint8
	DrniAggregatorPriority  uint16
	DrniPortalAddr          net.HardwareAddr
	DrniPortalPriority      uint16
	DrniThreeSystemPortal   bool
	DrniPortConversation    [MAX_CONVERSATION_IDS][4]uint16
	DrniGatewayConversation [MAX_CONVERSATION_IDS][]uint8
	// End also defined in 9.4.7

	// save the origional values from the aggregator
	PrevAggregatorId       [6]uint8
	PrevAggregatorPriority uint16

	DrniPortalSystemNumber  uint8                 // 1-3
	DrniIntraPortalLinkList [MAX_IPP_LINKS]uint32 // ifindex
	DrniAggregator          int32
	DrniConvAdminGateway    [MAX_CONVERSATION_IDS][]uint8
	// conversation id -> gateway
	DrniNeighborAdminConvGatewayListDigest Md5Digest
	DrniNeighborAdminConvPortListDigest    Md5Digest
	DrniGatewayAlgorithm                   GatewayAlgorithm
	DrniNeighborAdminGatewayAlgorithm      GatewayAlgorithm
	DrniNeighborAdminPortAlgorithm         GatewayAlgorithm
	DrniNeighborAdminDRCPState             uint8
	DrniEncapMethod                        EncapMethod
	DrniIPLEncapMap                        map[uint32]uint32
	DrniNetEncapMap                        map[uint32]uint32
	DrniPSI                                bool
	DrniPortConversationControl            bool
	DrniPortalPortProtocolIDA              net.HardwareAddr

	// TODO This should be removed
	GatewayVectorDatabase []GatewayVectorEntry

	// 9.4.10
	PortConversationUpdate     bool
	IppAllPortUpdate           bool
	GatewayConversationUpdate  bool
	IppAllGatewayUpdate        bool
	HomeGatewayVectorTransmit  bool
	OtherGatewayVectorTransmit bool

	// channel used to wait on response from distributed event send
	drEvtResponseChan chan string

	a *lacp.LaAggregator

	// Local list to keep track of distributed port list
	// Server will only indicate that a change has occured
	// updateDRFHomeState will determine what actions
	// to perform based on the differences between
	// what DR and Aggregator distributed port list
	DRAggregatorDistributedList []int32

	// sync creation and deletion
	wg sync.WaitGroup

	// state machines
	PsMachineFsm *PsMachine
	GMachineFsm  *GMachine
	AMachineFsm  *AMachine

	Ipplinks []*DRCPIpp
}

// 802.1ax-2014 Section 9.4.8 Per-DR Function variables
type DistributedRelayFunction struct {
	ChangeDRFPorts                                bool
	ChangePortal                                  bool
	DrniCommonMethods                             bool
	DrniConversationGatewayList                   [MAX_CONVERSATION_IDS]uint32
	DrniPortalSystemState                         [4]StateVectorInfo
	DRFHomeAdminAggregatorKey                     uint16
	DRFHomeConversationGatewayListDigest          Md5Digest
	DRFHomeConversationPortListDigest             Md5Digest
	DRFHomeGatewayAlgorithm                       [4]uint8
	DRFHomeGatewayConversationMask                [MAX_CONVERSATION_IDS]bool
	DRFHomeGatewaySequence                        uint16
	DRFHomePortAlgorithm                          [4]uint8
	DRFHomeOperAggregatorKey                      uint16
	DRFHomeOperPartnerAggregatorKey               uint16
	DRFHomeState                                  StateVectorInfo
	DRFNeighborAdminConversationGatewayListDigest Md5Digest
	DRFNeighborAdminConversationPortListDigest    Md5Digest
	DRFNeighborAdminDRCPState                     layers.DRCPState
	DRFNeighborAdminGatewayAlgorithm              [4]uint8
	DRFNeighborAdminPortAlgorithm                 [4]uint8
	// range 1..3
	DRFPortalSystemNumber uint8
	DRFHomeOperDRCPState  layers.DRCPState
	PSI                   bool

	// 9.3.3.2
	DrniPortalSystemGatewayConversation [MAX_CONVERSATION_IDS]bool
	DrniPortalSystemPortConversation    [MAX_CONVERSATION_IDS]bool
}

// DrFindByPortalAddr each portal address is unique within the system
func DrFindByPortalAddr(portaladdr string, dr **DistributedRelay) bool {
	for _, d := range DistributedRelayDBList {
		if d.DrniPortalAddr.String() == portaladdr {
			*dr = d
			return true
		}
	}
	return false
}

// DrFindByAggregator will find the DR based on the Aggregator that it is
// associated with
func DrFindByAggregator(DrniAggregator int32, dr **DistributedRelay) bool {
	for _, d := range DistributedRelayDBList {
		if d.DrniAggregator == DrniAggregator {
			*dr = d
			return true
		}
	}
	return false
}

// isPortInConversation will check of the provided portList intersected with
// the aggregator port list is greater than zero
func (dr *DistributedRelay) isAggPortInConverstaion(portList []int32) bool {
	a := dr.a

	if a.PortNumList != nil {
		for _, ifindex := range a.PortNumList {
			for _, pifindex := range portList {
				if int32(ifindex) == pifindex {
					return true
				}
			}
		}
	}
	return false
}

// setTimeSharingGatwewayDigest, when the port and gateway algorithm
// is set to time sharing then it should be noted that the gateway
// and port algorithm digest
// currently we only support Vlan based
// to start each
// algorithm is as follows:
// Conversations are not bound to a lag link but rather a portal system,
// thus all down traffic will either go to the local aggregator ports
// or IPL if the destination is a remote portal network port (which is not
// an aggregator port).  All up traffic is only destined to another
// aggregator or other network links either in hte local system or accross
// the IPL to the neighbor system.
// If all local aggregator ports are down then the neighbor system must
// forward frames out the aggregator as well as any network links to
// which the frame is destined for
func (dr *DistributedRelay) SetTimeSharingPortAndGatwewayDigest() {
	// algorithm assumes 2P system only
	if dr.DrniGatewayAlgorithm == GATEWAY_ALGORITHM_CVID {
		if !dr.DrniThreeSystemPortal {
			dr.setAdminConvGatewayAndNeighborGatewayListDigest()
			dr.setAdminConvPortAndNeighborPortListDigest()
		}
	}
}

// setAdminConvGatewayAndNeighborGatewayListDigest will set the predetermined
// algorithm as the gateway.  Every even vlan will have its gateway in system
// 2 and every odd vlan will have its gateway in system 1
func (dr *DistributedRelay) setAdminConvGatewayAndNeighborGatewayListDigest() {
	isNewConversation := false
	ghash := md5.New()
	for cid, conv := range ConversationIdMap {
		if cid == 100 {
			fmt.Printf("conv %+v   isAggPortInConversation %t  portList %+v\n", conv, dr.isAggPortInConverstaion(conv.PortList), dr.a.PortNumList)
		}
		if conv.Valid && dr.isAggPortInConverstaion(conv.PortList) {
			fmt.Printf("Conversation Admin Gateway %+v\n", dr.DrniConvAdminGateway[cid])

			// mark this call as new so that we can update the state machines
			if dr.DrniConvAdminGateway[cid] == nil {
				dr.DrniConvAdminGateway[cid] = make([]uint8, 0)
				isNewConversation = true
				fmt.Printf("Adding New Gateway Conversation %d\n", cid)
				dr.LaDrLog(fmt.Sprintf("Adding New Gateway Conversation %d", cid))

				// Fixed algorithm for 2P system
				// Because we only support sharing by time we don't really care which
				// system is the "gateway" of the conversation because all conversations
				// are free to be delivered on both systems based on bridging rules.
				// Annex G:
				//  A frame received over the IPL shall never be forwarded over the Aggregator Port.
				//  A frame received over the IPL with a DA that was learned from the Aggregator Port shall be discarded.
				//
				// NOTE when other sharing methods are supported then this algorithm will
				// need to be changed
				if math.Mod(float64(conv.Cvlan), 2) == 0 {
					dr.DrniConvAdminGateway[cid] = append(dr.DrniConvAdminGateway[cid], 2)
					dr.DrniConvAdminGateway[cid] = append(dr.DrniConvAdminGateway[cid], 1)
				} else {
					dr.DrniConvAdminGateway[cid] = append(dr.DrniConvAdminGateway[cid], 1)
					dr.DrniConvAdminGateway[cid] = append(dr.DrniConvAdminGateway[cid], 2)
				}
			}
			buf := new(bytes.Buffer)
			utils.GlobalLogger.Info("Adding to Gateway Digest:", conv.Cvlan, math.Mod(float64(conv.Cvlan), 2), []uint8{dr.DrniConvAdminGateway[cid][0], dr.DrniConvAdminGateway[cid][1], uint8(cid >> 8 & 0xff), uint8(cid & 0xff)})
			// network byte order
			binary.Write(buf, binary.BigEndian, []uint8{dr.DrniConvAdminGateway[cid][0], dr.DrniConvAdminGateway[cid][1], uint8(cid >> 8 & 0xff), uint8(cid & 0xff)})
			ghash.Write(buf.Bytes())
		} else {
			buf := new(bytes.Buffer)
			// network byte order
			binary.Write(buf, binary.BigEndian, []uint16{uint16(cid)})
			ghash.Write(buf.Bytes())

			if dr.DrniConvAdminGateway[cid] != nil {
				isNewConversation = true
			}

			dr.DrniConvAdminGateway[cid] = nil
		}
	}
	for i, val := range ghash.Sum(nil) {
		dr.DrniNeighborAdminConvGatewayListDigest[i] = val
		dr.DRFNeighborAdminConversationGatewayListDigest[i] = val
		dr.DRFHomeConversationGatewayListDigest[i] = val
	}

	// always send regardless of state because all states expect this event
	if isNewConversation &&
		dr.PsMachineFsm != nil {
		dr.ChangePortal = true
		dr.PsMachineFsm.PsmEvents <- utils.MachineEvent{
			E:   PsmEventChangePortal,
			Src: DRCPConfigModuleStr,
		}
	}
}

// setAdminConvGatewayAndNeighborGatewayListDigest will set the predetermined
// algorithm as the gateway.  Port Digest is not used as the port conversation
// is determined by hw hashing algorithm, thus setting no priority port list
// against the digest.
func (dr *DistributedRelay) setAdminConvPortAndNeighborPortListDigest() {
	phash := md5.New()
	for cid, _ := range ConversationIdMap {
		buf := new(bytes.Buffer)
		// network byte order
		binary.Write(buf, binary.BigEndian, []uint16{uint16(cid)})
		phash.Write(buf.Bytes())
	}

	for i, val := range phash.Sum(nil) {
		dr.DrniNeighborAdminConvPortListDigest[i] = val
		dr.DRFNeighborAdminConversationPortListDigest[i] = val
		dr.DRFHomeConversationPortListDigest[i] = val
	}
}

// NewDistributedRelay create a new instance of Distributed Relay and
// the associated objects for the IPP ports
func NewDistributedRelay(cfg *DistrubtedRelayConfig) *DistributedRelay {

	dr := &DistributedRelay{
		DrniId:                      uint32(cfg.DrniPortalSystemNumber),
		DrniName:                    cfg.DrniName,
		DrniPortalPriority:          cfg.DrniPortalPriority,
		DrniThreeSystemPortal:       cfg.DrniThreePortalSystem,
		DrniPortalSystemNumber:      cfg.DrniPortalSystemNumber,
		DrniIntraPortalLinkList:     cfg.DrniIntraPortalLinkList,
		DrniAggregator:              int32(cfg.DrniAggregator),
		DrniPortConversationControl: cfg.DrniPortConversationControl,
		drEvtResponseChan:           make(chan string),
		DrniIPLEncapMap:             make(map[uint32]uint32),
		DrniNetEncapMap:             make(map[uint32]uint32),
		DistributedRelayFunction: DistributedRelayFunction{
			DRFHomeState: StateVectorInfo{mutex: &sync.Mutex{}},
		},
	}

	neighborPortalSystemNumber := uint32(2)
	if cfg.DrniPortalSystemNumber == 1 {
		neighborPortalSystemNumber = 1

	}
	// Only support two portal system so we need to adjust
	// the ipp port id.  This should ideally come from the user
	// but lets make provisioning as simple as possible
	for i, ippPortId := range cfg.DrniIntraPortalLinkList {
		if ippPortId>>16&0x3 == 0 {
			dr.DrniIntraPortalLinkList[i] = ippPortId | (neighborPortalSystemNumber << 16)
		}
	}

	for i, _ := range dr.DrniPortalSystemState {
		dr.DrniPortalSystemState[i].mutex = &sync.Mutex{}
	}

	/*
		Not allowing user to set we are goign to fill this in via
		setTimeSharingPortAndGatwewayDigest
		for cid, data := range cfg.DrniConvAdminGateway {
			if data != [3]uint8{} {
				dr.DrniConvAdminGateway[cid] = make([]uint8, 0)
				for _, sysnum := range data {
					if sysnum != 0 {
						dr.DrniConvAdminGateway[cid] = append(dr.DrniConvAdminGateway[cid], sysnum)
					}
				}
			}
		}
	*/
	dr.DrniPortalAddr, _ = net.ParseMAC(cfg.DrniPortalAddress)
	for i, macbyte := range dr.DrniPortalAddr {
		dr.DrniAggregatorId[i] = macbyte
	}

	// string format in bits "00000000"
	for i, j := 0, uint32(7); i < 8; i, j = i+1, j-1 {
		val, _ := strconv.Atoi(cfg.DrniNeighborAdminDRCPState[i : i+1])
		dr.DrniNeighborAdminDRCPState |= uint8(val << j)
		dr.DRFNeighborAdminDRCPState |= layers.DRCPState(val << j)
	}

	/*
		Not allowing user to set we are goign to fill this in via
		setTimeSharingPortAndGatwewayDigest
		for i := 0; i < 16; i++ {
			dr.DrniNeighborAdminConvPortListDigest[i] = cfg.DrniNeighborAdminConvPortListDigest[i]
		}
	*/

	// format "00:00:00:00"
	encapmethod := strings.Split(cfg.DrniEncapMethod, ":")
	gatewayalgorithm := strings.Split(cfg.DrniGatewayAlgorithm, ":")
	neighborgatewayalgorithm := strings.Split(cfg.DrniNeighborAdminGatewayAlgorithm, ":")
	//neighborportalgorithm := strings.Split(cfg.DrniNeighborAdminPortAlgorithm, ":")
	var val1, val2, val3, val4 int64
	val1, _ = strconv.ParseInt(encapmethod[0], 16, 16)
	val2, _ = strconv.ParseInt(encapmethod[1], 16, 16)
	val3, _ = strconv.ParseInt(encapmethod[2], 16, 16)
	val4, _ = strconv.ParseInt(encapmethod[3], 16, 16)
	dr.DrniEncapMethod = EncapMethod{uint8(val1), uint8(val2), uint8(val3), uint8(val4)}
	val1, _ = strconv.ParseInt(gatewayalgorithm[0], 16, 16)
	val2, _ = strconv.ParseInt(gatewayalgorithm[1], 16, 16)
	val3, _ = strconv.ParseInt(gatewayalgorithm[2], 16, 16)
	val4, _ = strconv.ParseInt(gatewayalgorithm[3], 16, 16)
	dr.DrniGatewayAlgorithm = [4]uint8{uint8(val1), uint8(val2), uint8(val3), uint8(val4)}
	val1, _ = strconv.ParseInt(neighborgatewayalgorithm[0], 16, 16)
	val2, _ = strconv.ParseInt(neighborgatewayalgorithm[1], 16, 16)
	val3, _ = strconv.ParseInt(neighborgatewayalgorithm[2], 16, 16)
	val4, _ = strconv.ParseInt(neighborgatewayalgorithm[3], 16, 16)
	dr.DrniNeighborAdminGatewayAlgorithm = [4]uint8{uint8(val1), uint8(val2), uint8(val3), uint8(val4)}
	dr.DRFNeighborAdminGatewayAlgorithm = [4]uint8{uint8(val1), uint8(val2), uint8(val3), uint8(val4)}

	for i, data := range cfg.DrniIPLEncapMap {
		dr.DrniIPLEncapMap[uint32(i)] = data
	}
	for i, data := range cfg.DrniNetEncapMap {
		dr.DrniNetEncapMap[uint32(i)] = data
	}

	netMac, _ := net.ParseMAC(cfg.DrniIntraPortalPortProtocolDA)
	dr.DrniPortalPortProtocolIDA = netMac

	// add to the global db's
	DistributedRelayDB[dr.DrniName] = dr
	DistributedRelayDBList = append(DistributedRelayDBList, dr)

	dr.LaDrLog(fmt.Sprintf("Created Distributed Relay %+v\n", dr))

	for _, ippid := range dr.DrniIntraPortalLinkList {
		portid := ippid & 0xffff
		if portid > 0 {
			ipp := NewDRCPIpp(ippid, dr)
			// disabled until an aggregator has been attached
			ipp.DRCPEnabled = false
			dr.Ipplinks = append(dr.Ipplinks, ipp)
		}
	}
	return dr
}

// DeleteDistriutedRelay will delete the distributed relay along with
// the associated IPP links and de-associate from the Aggregator
func (dr *DistributedRelay) DeleteDistributedRelay() {

	// detach was not called externally, so lets call it
	if dr.a != nil {
		DetachAggregatorFromDistributedRelay(int(dr.DrniAggregator))
	}

	for _, ipp := range dr.Ipplinks {
		ipp.DeleteDRCPIpp()
	}

	// cleanup the tables hosting the dr data
	// cleanup the tables
	if _, ok := DistributedRelayDB[dr.DrniName]; ok {
		delete(DistributedRelayDB, dr.DrniName)
		for i, deldr := range DistributedRelayDBList {
			if deldr == dr {
				DistributedRelayDBList = append(DistributedRelayDBList[:i], DistributedRelayDBList[i+1:]...)
			}
		}
	}
}

// BEGIN will start/build all the Distributed Relay State Machines and
// send the begin event
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
		// Portal System Machine
		dr.DrcpPsMachineMain()
		// Gateway Machine
		dr.DrcpGMachineMain()
		// Aggregator Machine
		dr.DrcpAMachineMain()
	}

	// wait group used when stopping all the
	// State mahines associated with this port.
	// want to ensure that all routines are stopped
	// before proceeding with cleanup thus why not
	// create the wg as part of a BEGIN process
	// 1) Portal System Machine
	// 2) Gateway Machine
	// 3) Aggregator Machine
	// Psm
	if dr.PsMachineFsm != nil {
		mEvtChan = append(mEvtChan, dr.PsMachineFsm.PsmEvents)
		evt = append(evt, utils.MachineEvent{
			E:   PsmEventBegin,
			Src: DRCPConfigModuleStr})
	}

	// Gm
	if dr.GMachineFsm != nil {
		mEvtChan = append(mEvtChan, dr.GMachineFsm.GmEvents)
		evt = append(evt, utils.MachineEvent{
			E:   GmEventBegin,
			Src: DRCPConfigModuleStr})
	}
	// Am
	if dr.AMachineFsm != nil {
		mEvtChan = append(mEvtChan, dr.AMachineFsm.AmEvents)
		evt = append(evt, utils.MachineEvent{
			E:   AmEventBegin,
			Src: DRCPConfigModuleStr})
	}
	// call the begin event for each
	// distribute the port disable event to various machines
	dr.DistributeMachineEvents(mEvtChan, evt, true)
}

func (dr *DistributedRelay) waitgroupadd(m string) {
	//fmt.Println("Calling wait group add", m)
	dr.wg.Add(1)
}

func (dr *DistributedRelay) waitgroupstop(m string) {
	//fmt.Println("Calling wait group stop", m)
	dr.wg.Done()
}

func (dr *DistributedRelay) Stop() {
	// Psm
	if dr.PsMachineFsm != nil {
		dr.PsMachineFsm.Stop()
		dr.PsMachineFsm = nil
	}
	// Gm
	if dr.GMachineFsm != nil {
		dr.GMachineFsm.Stop()
		dr.GMachineFsm = nil
	}
	// Am
	if dr.AMachineFsm != nil {
		dr.AMachineFsm.Stop()
		dr.AMachineFsm = nil
	}
	dr.wg.Wait()

	close(dr.drEvtResponseChan)
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
		go func(d *DistributedRelay, w bool, idx int, machineEventChannel []chan utils.MachineEvent, event []utils.MachineEvent) {
			if w {
				event[idx].ResponseChan = d.drEvtResponseChan
			}
			event[idx].Src = DRCPConfigModuleStr
			machineEventChannel[idx] <- event[idx]
		}(dr, waitForResponse, j, mec, e)
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
