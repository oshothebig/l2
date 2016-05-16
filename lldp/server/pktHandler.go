Copyright [2016] [SnapRoute Inc]

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

	 Unless required by applicable law or agreed to in writing, software
	 distributed under the License is distributed on an "AS IS" BASIS,
	 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
	 See the License for the specific language governing permissions and
	 limitations under the License.
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
	for pkt := range pktSrc.Packets() {
		// check if rx channel is still valid or not
		if svr.lldpRxPktCh == nil {
			return
		}
		select {
		// check if we receive exit call
		case exit := <-svr.lldpExit:
			if exit {
				debug.Logger.Info(fmt.Sprintln("ifIndex:",
					ifIndex, "received lldp exit"))
			}
		default:

			// process packets
			gblInfo, exists := svr.lldpGblInfo[ifIndex]
			if !exists {
				debug.Logger.Info(fmt.Sprintln("No Entry for",
					ifIndex, "terminate go routine"))
				return
			}
			// pcap will be closed only in two places
			// 1) during interface state down
			// 2) during os exit
			// Because this is read we do not need to worry about
			// doing any locks...
			if gblInfo.PcapHandle == nil {
				debug.Logger.Info("Pcap closed terminate go " +
					"routine for " + gblInfo.Port.Name)
				return
			}
			svr.lldpRxPktCh <- InPktChannel{
				pkt:     pkt,
				ifIndex: ifIndex,
			}

		}
	}
}

/*  lldp server go routine to handle tx timer... once the timer fires we will
 *  send the ifindex on the channel to handle send info
 */
func (svr *LLDPServer) TransmitFrames(pHandle *pcap.Handle, ifIndex int32) {
	for {
		gblInfo, exists := svr.lldpGblInfo[ifIndex]
		if !exists {
			return
		}
		/*  Go Routine to send frames is already spawned. But after some time the port is
		 *  in down state. If that is the case then we do not delete this go routine.
		 *  We stop the timer and clear the cache from the global runtime information..
		 *  This for loop is gonna keep on running, so to avoid any un-necessary timer
		 *  trigger we added an extra check for starting the timer only when the port is
		 *  in LLDP_PORT_STATE_UP
		 */
		if gblInfo.Port.OperState == LLDP_PORT_STATE_DOWN {
			continue
		}
		gblInfo.TxInfo.TxTimer =
			time.NewTimer(time.Duration(gblInfo.TxInfo.MessageTxInterval) *
				time.Second)
		svr.lldpGblInfo[ifIndex] = gblInfo
		// Start lldpMessage Tx interval
		<-gblInfo.TxInfo.TxTimer.C

		svr.lldpTxPktCh <- SendPktChannel{
			ifIndex: ifIndex,
		}
	}
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
