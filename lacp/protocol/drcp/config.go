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
	DrniName                               string
	DrniPortalAddress                      string
	DrniPortalPriority                     uint16
	DrniThreePortalSystem                  bool
	DrniPortalSystemNumber                 uint8
	DrniIntraPortalLinkList                [3]uint32
	DrniAggregator                         uint32
	DrniConvAdminGateway                   [4096][3]uint8
	DrniNeighborAdminConvGatewayListDigest [16]uint8
	DrniNeighborAdminConvPortListDigest    [16]uint8
	DrniGatewayAlgorithm                   string
	DrniNeighborAdminGatewayAlgorithm      string
	DrniNeighborAdminPortAlgorithm         string
	DrniNeighborAdminDRCPState             string
	DrniEncapMethod                        string
	DrniIPLEncapMap                        [16]uint32
	DrniNetEncapMap                        [16]uint32
	DrniPortConversationControl            bool
	DrniIntraPortalPortProtocolDA          string
}

// Conversations are typically related to the various service types to which
// traffic is associated with.  If portList is empty is is assumed to be Gateway
// Algorithm, otherwise it is a Port Algorithm
// 802.1 AX-2014 8.1
// Therefore, a Conversation Identifier (or Conversation ID) is defined as a value in the range 0 through 4095.
// By administrative means, every possible conversation is assigned to a single Conversation ID value for each
// supported Conversation ID type. More than one conversation can be assigned to a Conversation ID. It is not
// necessary that every Conversation ID value have any conversations assigned to it. In this standard, several
// types of Conversation ID are specified for different uses.
type DRConversationConfig struct {
	DrniName   string
	Idtype     GatewayAlgorithm
	Isid       uint32
	Cvlan      uint16
	Svlan      uint16
	Bvid       uint16
	Psuedowire uint32
	PortList   []int32
}

type DRAggregatorPortListConfig struct {
	DrniAggregator uint32
	PortList       []int32
}

func (d *DistrubtedRelayConfig) GetKey() string {
	return d.DrniName
}

// DistrubtedRelayConfigParamCheck will validate the config from the user after it has
// been translated to something the Lacp module expects.  Thus if translation
// layer fails it should produce an invalid value.  The error returned
// will be translated to model values
func DistrubtedRelayConfigParamCheck(mlag *DistrubtedRelayConfig) error {

	_, err := net.ParseMAC(mlag.DrniPortalAddress)
	if err != nil {
		return errors.New(fmt.Sprintln("ERROR Portal System MAC Supplied must be in the format of 00:00:00:00:00:00 rcvd:", mlag.DrniPortalAddress))
	}

	invalidlinkcnt := 0
	for _, ippid := range mlag.DrniIntraPortalLinkList {
		portid := ippid & 0xffff
		if portid > 0 {
			if _, ok := utils.PortConfigMap[int32(portid)]; !ok {
				return errors.New(fmt.Sprintln("ERROR Invalid Intra Portal Link Port Id supplied", portid, utils.PortConfigMap))
			}
		} else {
			invalidlinkcnt++
		}
	}
	if invalidlinkcnt == 3 {
		return errors.New("ERROR Invalid Intra Portal Link, Must contain Port within system")
	}

	if mlag.DrniThreePortalSystem {
		return errors.New(fmt.Sprintln("ERROR Only support a 2 Portal System"))
	}

	if mlag.DrniPortalSystemNumber < DRNI_PORTAL_SYSTEM_ID_MIN ||
		mlag.DrniPortalSystemNumber > DRNI_PORTAL_SYSTEM_ID_MAX {
		return errors.New(fmt.Sprintln("ERROR Invalid Portal System Number must be between 1 and ", DRNI_PORTAL_SYSTEM_ID_MAX))
	}

	validPortGatewayAlgorithms := map[string]bool{
		"00:80:C2:01": true,
		"00:80:C2:02": true,
		"00:80:C2:03": true,
		"00:80:C2:04": true,
		"00:80:C2:05": true,
	}

	if _, ok := validPortGatewayAlgorithms[mlag.DrniGatewayAlgorithm]; !ok {
		return errors.New(fmt.Sprintln("ERROR Invalid Gateway Algorithm supplied must be in the format 00:80:C2:XX where XX is 1-5 the value of the algorithm ", mlag.DrniGatewayAlgorithm))
	}

	if _, ok := validPortGatewayAlgorithms[mlag.DrniNeighborAdminGatewayAlgorithm]; !ok {
		return errors.New(fmt.Sprintln("ERROR Invalid Neighbor Gateway Algorithm supplied must be in the format 00:80:C2:XX where XX is 1-5 the value of the algorithm ", mlag.DrniNeighborAdminGatewayAlgorithm))
	}

	if _, ok := validPortGatewayAlgorithms[mlag.DrniNeighborAdminPortAlgorithm]; !ok {
		return errors.New(fmt.Sprintln("ERROR Invalid Neighbor Port Algorithm supplied must be in the format 00:80:C2:XX where XX is 1-5 the value of the algorithm ", mlag.DrniNeighborAdminPortAlgorithm))
	}

	validEncapStrings := map[string]bool{
		"00:80:C2:00": true, // seperate physical or lag link
		"00:80:C2:01": true, // shared by time
		"00:80:C2:02": true, // shared by tag
	}

	if _, ok := validEncapStrings[mlag.DrniEncapMethod]; !ok {
		return errors.New(fmt.Sprintln("ERROR Invalid Encap Method supplied must be in the format 00:80:C2:XX where XX is 0-2 the value of the encap method ", mlag.DrniEncapMethod))
	}

	if mlag.DrniPortConversationControl {
		return errors.New(fmt.Sprintln("ERROR Invalid Port Conversation Control is always false as the Home Gateway Vector is controlled by protocol ", mlag.DrniPortConversationControl))
	}

	_, err = net.ParseMAC(mlag.DrniIntraPortalPortProtocolDA)
	if err != nil {
		return errors.New(fmt.Sprintln("ERROR Invalid Port Protocol DA invalid format must be 00:00:00:00:00:00 rcvd: ", mlag.DrniIntraPortalPortProtocolDA))
	}

	// only going to support this address
	if mlag.DrniIntraPortalPortProtocolDA != "01:80:C2:00:00:03" {
		return errors.New(fmt.Sprintln("ERROR Invalid Port Protocol DA only support 01:80:C2:00:00:03 rcvd: ", mlag.DrniIntraPortalPortProtocolDA))
	}

	return nil
}

// CreateDistributedRelay will create the distributed relay then attach
// the Aggregator to the Distributed Relay
func CreateDistributedRelay(cfg *DistrubtedRelayConfig) {

	dr := NewDistributedRelay(cfg)
	if dr != nil {
		AttachAggregatorToDistributedRelay(int(dr.DrniAggregator))
	}
}

// DeleteDistributedRelay will detach the distributed relay from the aggregator
// and delete the distributed relay instance
func DeleteDistributedRelay(name string) {

	dr, ok := DistributedRelayDB[name]
	if ok {
		DetachAggregatorFromDistributedRelay(int(dr.DrniAggregator))
		dr.DeleteDistributedRelay()
	}
}

// UpdateAggregatorPortList to be called when aggregator port list has
// been updated
func UpdateAggregatorPortList(aggId int) {
	for _, dr := range DistributedRelayDBList {
		fmt.Printf("Looking for aggId %d found %d portList %+v\n", aggId, dr.DrniAggregator, dr.a.PortNumList)
		if dr.DrniAggregator == int32(aggId) &&
			dr.a != nil &&
			len(dr.a.PortNumList) > 0 {
			fmt.Println("UpdateAggregatorPortList: found aggregator")
			// set this to allow for portal system machine to fall through
			// after initialization
			dr.ChangeDRFPorts = true
			if dr.PsMachineFsm != nil {
				dr.PsMachineFsm.PsmEvents <- utils.MachineEvent{
					E:   PsmEventChangeDRFPorts,
					Src: DRCPConfigModuleStr,
				}
			}
		}
	}
}

// AttachAggregatorToDistributedRelay: will attach the aggregator and start the Distributed
// relay protocol for the given dr if this agg is associated with a DR
func AttachAggregatorToDistributedRelay(aggId int) {
	for _, dr := range DistributedRelayDBList {
		if dr.DrniAggregator == int32(aggId) &&
			dr.a == nil {
			var a *lacp.LaAggregator
			if lacp.LaFindAggById(aggId, &a) {
				dr.a = a
				a.DrniName = dr.DrniName
				a.ActorOperKey = uint16(dr.DRFHomeOperAggregatorKey)
				a.PartnerOperKey = a.ActorOperKey

				dr.LaDrLog(fmt.Sprintf("Attaching Agg %s %d to DR %s", a.AggName, a.AggId, dr.DrniName))

				// These values should be the same as the admin
				dr.PrevAggregatorId = a.AggMacAddr
				dr.PrevAggregatorPriority = a.AggPriority
				dr.LaDrLog(fmt.Sprintf("Saving Orig SystemId %+v Priority %d", a.AggMacAddr, a.AggPriority))

				// set the aggregator Id for the aggregator as this is used in
				// setDefaultPortalSystemParameters
				a.AggMacAddr = dr.DrniAggregatorId

				// only need to set this once the key has been negotiated.
				if dr.PsMachineFsm != nil &&
					dr.PsMachineFsm.Machine.Curr.CurrentState() == PsmStatePortalSystemUpdate {
					// lets update the aggregator parameters
					// configured ports
					for _, aggport := range a.PortNumList {
						var p *lacp.LaAggPort
						if lacp.LaFindPortById(aggport, &p) {

							dr.LaDrLog(fmt.Sprintf("Aggregator found updating system parameters moving to unselected until DR is synced"))
							// assign the new values to the aggregator
							lacp.SetLaAggPortSystemInfoFromDistributedRelay(
								uint16(aggport),
								fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x",
									dr.DrniPortalAddr[0],
									dr.DrniPortalAddr[1],
									dr.DrniPortalAddr[2],
									dr.DrniPortalAddr[3],
									dr.DrniPortalAddr[4],
									dr.DrniPortalAddr[5]),
								dr.DrniPortalPriority,
								dr.DRFHomeOperAggregatorKey,
								dr.DrniName,
								false)
						} else {
							dr.LaDrLog(fmt.Sprintf("ERROR unable update system info on port %d not found", aggport))
						}
					}
				}
				if len(a.PortNumList) == 0 {
					dr.LaDrLog(fmt.Sprintf("Aggregator found but port list is empty thus not updating system parameters"))
				}

				// add the port to the local distributed list so that the digests can be
				// calculated
				dr.DRAggregatorDistributedList = make([]int32, 0)
				for _, disport := range a.DistributedPortNumList {
					var aggp *lacp.LaAggPort
					foundPort := false
					for lacp.LaGetPortNext(&aggp) && !foundPort {
						if aggp.IntfNum == disport {
							dr.DRAggregatorDistributedList = append(dr.DRAggregatorDistributedList, int32(aggp.PortNum))
							dr.LaDrLog(fmt.Sprintf("Aggregator port in Distributing State", aggp.PortNum))
						}
					}
				}

				// set port and gateway info and digest
				//dr.SetTimeSharingPortAndGatwewayDigest()
				// TODO can we get away with just setting
				if len(a.PortNumList) > 0 {
					// set this to allow for portal system machine to fall through
					// after initialization
					dr.ChangeDRFPorts = true
					dr.ChangePortal = true
				}

				dr.BEGIN(false)
				// start the IPP links
				for _, ipp := range dr.Ipplinks {
					dr.LaDrLog(fmt.Sprintf("Starting Ipp %s", ipp.Name))
					ipp.DRCPEnabled = true
					ipp.BEGIN(false)
				}
			}
		}
	}
}

// DetachCreatedAggregatorFromDistributedRelay: will detach the aggregator and stop the Distributed
// relay protocol for the given dr if since this aggregator is no longer attached
func DetachAggregatorFromDistributedRelay(aggId int) {
	for _, dr := range DistributedRelayDBList {
		if dr.DrniAggregator == int32(aggId) &&
			dr.a != nil {
			var a *lacp.LaAggregator
			if lacp.LaFindAggById(aggId, &a) {
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
				// reset aggregator values
				a.DrniName = ""
				a.AggMacAddr = dr.PrevAggregatorId
				a.AggPriority = dr.PrevAggregatorPriority
				a.ActorOperKey = a.ActorAdminKey
			}
			dr.Stop()
			dr.a = nil
		}
	}
}

// DistributedRelayAggregatorPortListUpdate inform the Portal System Machine
// that the port list has changed.  It will then send the correct update
// to gateway and aggregator machines
func DistributedRelayAggregatorPortListUpdate(cfg *DRAggregatorPortListConfig) {
	var dr *DistributedRelay
	if DrFindByAggregator(int32(cfg.DrniAggregator), &dr) {

		a := dr.a

		if a != nil {
			// fill in current distributed port list
			dr.DRAggregatorDistributedList = make([]int32, 0)
			// add the port to the local distributed list so that the digests can be
			// calculated
			for _, disport := range a.DistributedPortNumList {
				var aggp *lacp.LaAggPort
				foundPort := false
				for lacp.LaGetPortNext(&aggp) && !foundPort {
					if aggp.IntfNum == disport {
						dr.DRAggregatorDistributedList = append(dr.DRAggregatorDistributedList, int32(aggp.PortNum))
					}
				}
			}

			dr.ChangeDRFPorts = true
			if dr.PsMachineFsm != nil {
				dr.PsMachineFsm.PsmEvents <- utils.MachineEvent{
					E:   PsmEventChangeDRFPorts,
					Src: DRCPConfigModuleStr,
				}
			}
		}
	}
}
