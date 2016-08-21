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

package server

import (
	"errors"
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
	"l2/lldp/utils"
	"time"
)

/* Go routine to recieve lldp frames. This go routine is created for all the
 * ports which are in up state.
 */
func (svr *LLDPServer) ReceiveFrames(pHandle *pcap.Handle, ifIndex int32) {
	pktSrc := gopacket.NewPacketSource(pHandle, pHandle.LinkType())
	in := pktSrc.Packets()
	// process packets
	gblInfo, exists := svr.lldpGblInfo[ifIndex]
	if !exists {
		debug.Logger.Info(fmt.Sprintln("No Entry for", ifIndex, "terminate go routine"))
		return
	}
	quit := gblInfo.RxKill
	for {
		select {
		case pkt, ok := <-in:
			//default:
			if !ok {
				debug.Logger.Info(fmt.Sprintln("Channel closed for ifIndex", ifIndex,
					"exiting go routine"))
				return
			}

			// process packets
			gblInfo, exists := svr.lldpGblInfo[ifIndex]
			if !exists {
				debug.Logger.Info(fmt.Sprintln("No Entry for", ifIndex, "terminate go routine"))
				return
			}

			// When StopRxTx is called for an interface, we will set the state to be disabled always or is
			// LLDP is disabled globally (something like no feature lldp) then we will exit out of
			// go routine and let StopRxTx worry about pcap handler
			if gblInfo.isDisabled() || !svr.Global.Enable {
				debug.Logger.Info("rx frame port " + gblInfo.Port.Name + "disabled exit go routine")
				return
			}
			// pcap will be closed only in two places
			// 1) during interface state down
			// 2) during os exit
			// Because this is read we do not need to worry about
			// doing any locks...
			if gblInfo.PcapHandle == nil {
				debug.Logger.Info("Pcap closed terminate go routine for " + gblInfo.Port.Name)
				return
			}
			// check if rx channel is still valid or not
			if svr.lldpRxPktCh == nil {
				return
			} else {
				svr.lldpRxPktCh <- InPktChannel{
					pkt:     pkt,
					ifIndex: ifIndex,
				}
			}
		case <-quit:
			debug.Logger.Info(fmt.Sprintln("quit for ifIndex", ifIndex, "rx exiting go routine"))
			return
		}
	}
}

/*  lldp server go routine to handle tx timer... once the timer fires we will
*  send the ifindex on the channel to handle send info
 */
func (svr *LLDPServer) TransmitFrames(ifIndex int32) {
	var TxTimerHandler_func func()
	TxTimerHandler_func = func() {
		svr.lldpTxPktCh <- SendPktChannel{
			ifIndex: ifIndex,
		}
		gblInfo, exists := svr.lldpGblInfo[ifIndex]
		if !exists {
			debug.Logger.Info(fmt.Sprintln("No Entry for", ifIndex, "terminate go routine"))
			return
		}
		// Wait until the packet is send out on the wire... Once done then reset the timer and
		// update global info again
		<-gblInfo.TxDone
		gblInfo.TxInfo.TxTimer.Reset(time.Duration(gblInfo.TxInfo.MessageTxInterval) * time.Second)
		svr.lldpGblInfo[ifIndex] = gblInfo
	}
	// Create an After Func and go routine for it, so that on timer stop TX is stopped automatically
	gblInfo, exists := svr.lldpGblInfo[ifIndex]
	if !exists {
		debug.Logger.Info(fmt.Sprintln("No Entry for", ifIndex, "terminate go routine"))
		return
	}

	gblInfo.TxInfo.TxTimer = time.AfterFunc(time.Duration(gblInfo.TxInfo.MessageTxInterval)*time.Second,
		TxTimerHandler_func)
	svr.lldpGblInfo[ifIndex] = gblInfo
}

/*  Write packet is helper function to send packet on wire.
 *  It will inform caller that packet was send successfully and you can go ahead
 *  and cache the pkt or else do not cache the packet as it is corrupted or there
 *  was some error
 */
func (gblInfo *LLDPGlobalInfo) WritePacket(pkt []byte) bool {
	var err error
	if len(pkt) == 0 {
		return false
	}
	if gblInfo.PcapHandle != nil {
		err = gblInfo.PcapHandle.WritePacketData(pkt)
	} else {
		err = errors.New("Pcap Handle is invalid for " + gblInfo.Port.Name)
	}
	if err != nil {
		debug.Logger.Err(fmt.Sprintln("Sending packet failed Error:",
			err, "for Port:", gblInfo.Port.Name))
		return false
	}
	return true
}
