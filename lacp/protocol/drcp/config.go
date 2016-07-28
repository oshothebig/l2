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

// config
package drcp

import (
	"fmt"
	//"sync"
	"errors"
	"l2/lacp/protocol/utils"
	"net"
	"strings"
	"time"
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
	aDrniIntraPortalLinkList                []uint32
	aDrniAggregator                         uint32
	aDrniConvAdminGateway                   []uint32
	aDrniNeighborAdminConvGatewayListDigest [16]uint8
	aDrniNeighborAdminConvPortListDigest    [16]uint8
	aDrniGatewayAlgorithm                   string
	aDrniNeighborAdminGatewayAlgorithm      string
	aDrniNeighborAdminPortAlgorithm         string
	aDrniNeighborAdminDRCPState             string
	aDrniEncapMethod                        string
	aDrniIPLEncapMap                        []uint32
	aDrniNetEncapMap                        []uint32
	aDrniPortConversationControl            bool
	aDrniIntraPortalPortProtocolDA          string
}

// DistrubtedRelayConfigParamCheck will validate the config from the user after it has
// been translated to something the Lacp module expects.  Thus if translation
// layer fails it should produce an invalid value.  The error returned
// will be translated to model values
func DistrubtedRelayConfigParamCheck(mlag *DistrubtedRelayConfig) error {

	mac, ok := net.ParseMAC(mlag.aDrniPortalAddress)
	if !ok {
		return errors.New(fmt.Sprintln("ERROR Portal System MAC Supplied must be in the format of 00:00:00:00:00:00 rcvd:", mlag.aDrniPortalAddress))
	}

	for _, link := range mlag.aDrniIntraPortalLinkList {
		if _, ok := utils.PortConfigMap[int32(link)]; !ok {
			return errors.New(fmt.Sprintln("ERROR Invalid Intra Portal Link Port Id supplied", link))
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
		return errors.New(fmt.Sprintln("ERROR Invalid Neighbor Port Algorithm supplied must be in the format 00:80:C2:XX where XX is 1-5 the value of the algorithm ", mlag.aDrniNeighborPortAlgorithm))
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

	netMac, ok := net.ParseMAC(mlag.aDrniIntraPortalPortProtocolDA)
	if !ok {
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
	if dr != nil {
		// start the links
		for _, ipp := range dr.Ipplinks {
			ipp.BEGIN()
		}
	}
}

func DeleteDistributedRelay(drId int) {
}
