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
package lacp

import (
	"fmt"
	"github.com/google/gopacket/layers"
	"time"
)

// DRNI - Distributed Resilient Network Interconnect

const (
	MAX_CONVERSATION_IDS = 4096
)

// 802.1ax-2014 7.4.2.1.1
type DistributedRelayIPP struct {
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
	GatewayConversationUpdate   bool
	IppAllGatewayUpdate         bool
	IppAllPortUpdate            bool
	IppAllUpdate                bool
	IppGatewayUpdate            bool
	IppPortUpdate               bool
	OtherGatewayVectorTransmit  bool
	PortConversationTransmit    bool
	PortConversationUpdate      bool
}

type DRCPIpp struct {
	DistributedRelayIPP
	DRCPIntraPortal
	DistributedRelayIPPDebug

	// reference to the distributed relay object
	dr *DistributedRelay

	Logger *DrcpDebug

	// FSMs
	RxMachineFsm *RxMachine
}

func NewDRCPIpp() *DRCPIpp {

	return &DRCPIpp{}
}

// ReportToManagement send events for various reason to infor management of something
// is wrong.
func (p *DRCPIpp) ReportToManagement() {

	p.Logger.Info(fmt.Sprintln("Report Failure to Management: %s", p.DifferPortalReason))
	// TODO send event
}

func (p *DRCPIpp) updateNTT() {
	if !p.DRFHomeOperDRCPState.GetState(layers.DRCPStateGatewaySync) ||
		!p.DRFHomeOperDRCPState.GetState(layers.DRCPStatePortSync) ||
		!p.DRFNeighborOperDRCPState.GetState(layers.DRCPStateGatewaySync) ||
		!p.DRFNeighborOperDRCPState.GetState(layers.DRCPStatePortSync) {
		p.NTTDRCPDU = true
	}
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
