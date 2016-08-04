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

// config.go
package drcp

import (
	"fmt"
	//"sync"
	"errors"
	"l2/lacp/protocol/lacp"
	"l2/lacp/protocol/utils"
	"net"
)

const (
	DRNI_PORTAL_SYSTEM_ID_MIN = 1
	DRNI_PORTAL_SYSTEM_ID_MAX = 2 // only support two portal system
)

const DRCPConfigModuleStr = "DRCP Config"

// 802.1.AX-2014 7.4.1.1 Distributed Relay Attributes GET-SET
type DistrubtedRelayConfig struct {
	// GET-SET
	aDrniName                               string
	aDrniPortalAddress                      string
	aDrniPortalPriority                     uint16
	aDrniThreePortalSystem                  bool
	aDrniPortalSystemNumber                 uint8
	aDrniIntraPortalLinkList                [3]uint32
	aDrniAggregator                         uint32
	aDrniConvAdminGateway                   [4096][3]uint8
	aDrniNeighborAdminConvGatewayListDigest [16]uint8
	aDrniNeighborAdminConvPortListDigest    [16]uint8
	aDrniGatewayAlgorithm                   string
	aDrniNeighborAdminGatewayAlgorithm      string
	aDrniNeighborAdminPortAlgorithm         string
	aDrniNeighborAdminDRCPState             string
	aDrniEncapMethod                        string
	aDrniIPLEncapMap                        [16]uint32
	aDrniNetEncapMap                        [16]uint32
	aDrniPortConversationControl            bool
	aDrniIntraPortalPortProtocolDA          string
}

// DistrubtedRelayConfigParamCheck will validate the config from the user after it has
// been translated to something the Lacp module expects.  Thus if translation
// layer fails it should produce an invalid value.  The error returned
// will be translated to model values
func DistrubtedRelayConfigParamCheck(mlag *DistrubtedRelayConfig) error {

	_, err := net.ParseMAC(mlag.aDrniPortalAddress)
	if err != nil {
		return errors.New(fmt.Sprintln("ERROR Portal System MAC Supplied must be in the format of 00:00:00:00:00:00 rcvd:", mlag.aDrniPortalAddress))
	}

	for _, ippid := range mlag.aDrniIntraPortalLinkList {
		if _, ok := utils.PortConfigMap[int32(ippid)]; !ok {
			return errors.New(fmt.Sprintln("ERROR Invalid Intra Portal Link Port Id supplied", ippid))
		}
	}

	if mlag.aDrniThreePortalSystem {
		return errors.New(fmt.Sprintln("ERROR Only support a 2 Portal System"))
	}

	if mlag.aDrniPortalSystemNumber < DRNI_PORTAL_SYSTEM_ID_MIN ||
		mlag.aDrniPortalSystemNumber > DRNI_PORTAL_SYSTEM_ID_MAX {
		return errors.New(fmt.Sprintln("ERROR Invalid Portal System Number must be between 1 and ", DRNI_PORTAL_SYSTEM_ID_MAX))
	}

	validPortGatewayAlgorithms := map[string]bool{
		"00:80:C2:01": true,
		"00:80:C2:02": true,
		"00:80:C2:03": true,
		"00:80:C2:04": true,
		"00:80:C2:05": true,
	}

	if _, ok := validPortGatewayAlgorithms[mlag.aDrniGatewayAlgorithm]; !ok {
		return errors.New(fmt.Sprintln("ERROR Invalid Gateway Algorithm supplied must be in the format 00:80:C2:XX where XX is 1-5 the value of the algorithm ", mlag.aDrniGatewayAlgorithm))
	}

	if _, ok := validPortGatewayAlgorithms[mlag.aDrniNeighborAdminGatewayAlgorithm]; !ok {
		return errors.New(fmt.Sprintln("ERROR Invalid Neighbor Gateway Algorithm supplied must be in the format 00:80:C2:XX where XX is 1-5 the value of the algorithm ", mlag.aDrniNeighborAdminGatewayAlgorithm))
	}

	if _, ok := validPortGatewayAlgorithms[mlag.aDrniNeighborAdminPortAlgorithm]; !ok {
		return errors.New(fmt.Sprintln("ERROR Invalid Neighbor Port Algorithm supplied must be in the format 00:80:C2:XX where XX is 1-5 the value of the algorithm ", mlag.aDrniNeighborAdminPortAlgorithm))
	}

	validEncapStrings := map[string]bool{
		"00:80:C2:00": true, // seperate physical or lag link
		"00:80:C2:01": true, // shared by time
		"00:80:C2:02": true, // shared by tag
	}

	if _, ok := validEncapStrings[mlag.aDrniEncapMethod]; !ok {
		return errors.New(fmt.Sprintln("ERROR Invalid Encap Method supplied must be in the format 00:80:C2:XX where XX is 0-2 the value of the encap method ", mlag.aDrniEncapMethod))
	}

	if mlag.aDrniPortConversationControl {
		return errors.New(fmt.Sprintln("ERROR Invalid Port Conversation Control is always false as the Home Gateway Vector is controlled by protocol ", mlag.aDrniPortConversationControl))
	}

	_, err = net.ParseMAC(mlag.aDrniIntraPortalPortProtocolDA)
	if err != nil {
		return errors.New(fmt.Sprintln("ERROR Invalid Port Protocol DA invalid format must be 00:00:00:00:00:00 rcvd: ", mlag.aDrniIntraPortalPortProtocolDA))
	}

	// only going to support this address
	if mlag.aDrniIntraPortalPortProtocolDA != "01:80:C2:00:00:03" {
		return errors.New(fmt.Sprintln("ERROR Invalid Port Protocol DA only support 01:80:C2:00:00:03 rcvd: ", mlag.aDrniIntraPortalPortProtocolDA))
	}

	return nil
}

func CreateDistributedRelay(cfg *DistrubtedRelayConfig) {

	dr := NewDistributedRelay(cfg)
	// aggregator must exist for the protocol to really make sense
	if dr != nil {
		AttachAggregatorToDistributedRelay(int(dr.DrniAggregator))
		if dr.a != nil {
			dr.BEGIN(false)
			// start the IPP links
			for _, ipp := range dr.Ipplinks {
				ipp.BEGIN(false)
			}
		}
	}
}

func DeleteDistributedRelay(name string) {

	dr := DistributedRelayDB[name]
	DetachAggregatorFromDistributedRelay(int(dr.DrniAggregator))
	dr.DeleteDistributedRelay()
}

// AttachCreatedAggregator: will attach the aggregator and start the Distributed
// relay protocol for the given dr if this agg is associated with a DR
func AttachAggregatorToDistributedRelay(aggId int) {
	for _, dr := range DistributedRelayDBList {
		if dr.DrniAggregator == int32(aggId) &&
			dr.a == nil {
			var a *lacp.LaAggregator
			if lacp.LaFindAggById(aggId, &a) {
				dr.a = a

				// lets update the aggregator parameters
				// configured ports
				for _, pId := range a.PortNumList {
					var p *lacp.LaAggPort
					if lacp.LaFindPortById(pId, &p) {
						dr.PrevAggregatorId = p.ActorAdmin.System.Actor_System
						dr.PrevAggregatorPriority = p.ActorAdmin.System.Actor_System_priority
						// assign the new values to the aggregator
						lacp.SetLaAggPortSystemInfo(
							uint16(pId),
							fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x",
								dr.DrniPortalAddr[0],
								dr.DrniPortalAddr[1],
								dr.DrniPortalAddr[2],
								dr.DrniPortalAddr[3],
								dr.DrniPortalAddr[4],
								dr.DrniPortalAddr[5]),
							dr.DrniPortalPriority)
					}
				}

				dr.BEGIN(false)
				// start the IPP links
				for _, ipp := range dr.Ipplinks {
					ipp.BEGIN(false)
				}
			}
		}
	}
}

func DetachAggregatorFromDistributedRelay(aggId int) {
	for _, dr := range DistributedRelayDBList {
		if dr.DrniAggregator == int32(aggId) &&
			dr.a != nil {
			var a *lacp.LaAggregator
			if lacp.LaFindAggById(aggId, &a) {
				dr.a = a

				// lets update the aggregator parameters
				// configured ports
				for _, pId := range a.PortNumList {
					lacp.SetLaAggPortSystemInfo(
						uint16(pId),
						fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x",
							dr.PrevAggregatorId[0],
							dr.PrevAggregatorId[1],
							dr.PrevAggregatorId[2],
							dr.PrevAggregatorId[3],
							dr.PrevAggregatorId[4],
							dr.PrevAggregatorId[5]),
						dr.PrevAggregatorPriority)
				}
			}
			// re-init state machine
			dr.Stop()
			dr.a = nil
		}
	}
}
