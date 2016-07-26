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
package lacp

import (
	"fmt"
	"github.com/google/gopacket/layers"
	"l2/lacp/protocol/lacp"
	"net"
)

// 802.1ax-2014 7.4.1.1
type DistributedRelay struct {
	DistributedRelayFunction

	DrniId                  uint32
	DrniDescription         string
	DrniName                string
	DrniPortalAddr          net.HwAddress
	DrniPortalPriority      uint16
	DrniThreeSystemPortal   bool
	DrniPortalSystemNumber  uint8                // 1-3
	DrniIntraPortalLinkList [MAX_IPP_LINKS]int32 // ifindex
	DrniAggregator          int32
	DrniAggregatorPriority  uint16
	DrniAggregatorId        [6]uint8
	DrniConvAdminGateway    [MAX_GATEWAY_CONVERSATIONS]int32
	// conversation id -> gateway
	DrniNeighborAdminConvGatewayListDigest []Md5Digest
	DrniGatwayAlg                          GatewayAlgorithm
	DrniNeighborAdminGatewayAlgorithm      GatewayAlgorithm
	DrniNeighborAdminPortAlgorithm         GatewayAlgorithm
	DrniNeighborAdminDRCPState             uint8
	DrniEncapMethod                        EncapMethod
	DrniIPLEncapMap                        map[uint32]uint32
	DrniNetEncapMap                        map[uint32]uint32
	DrniPortConversationPasses             [MAX_CONVERSATION_IDS]bool
	DrniGatewayConversationPasses          [MAX_CONVERSATION_IDS]bool
	DrniPSI                                bool
	DrniPortConversationControl            bool
	DrniPortalPortProtocolIDA              net.HwAddress

	a *lacp.LaAggregator
}

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
	DRFHomeOperAggergatorKey                      uint16
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
}

// 802.1ax-2014 9.3.4.4
func extractGatewayConversationID() {

}

// 802.1ax-2014 9.3.4.4
func extractPortConversationID() {

}
