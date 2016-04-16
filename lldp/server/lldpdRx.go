package lldpServer

import (
	"errors"
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
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
				svr.logger.Info(fmt.Sprintln("ifIndex:",
					ifIndex, "received lldp exit"))
			}
		default:

			// process packets
			gblInfo, exists := svr.lldpGblInfo[ifIndex]
			if !exists {
				svr.logger.Info(fmt.Sprintln("No Entry for",
					ifIndex, "terminate go routine"))
				return
			}
			// pcap will be closed only in two places
			// 1) during interface state down
			// 2) during os exit
			// Because this is read we do not need to worry about
			// doing any locks...
			if gblInfo.PcapHandle == nil {
				svr.logger.Info("Pcap closed terminate go " +
					"routine for " + gblInfo.Name)
				return
			}
			svr.lldpRxPktCh <- InPktChannel{
				pkt:     pkt,
				ifIndex: ifIndex,
			}

		}
	}
}

/*  dump received lldp frame and other TX information
 */
func (gblInfo LLDPGlobalInfo) DumpFrame() {
	gblInfo.logger.Info(fmt.Sprintln("L2 Port:", gblInfo.Name, "Port Num:",
		gblInfo.PortNum))
	gblInfo.logger.Info(fmt.Sprintln("SrcMAC:", gblInfo.SrcMAC.String(),
		"DstMAC:", gblInfo.DstMAC.String()))
	gblInfo.logger.Info(fmt.Sprintln("ChassisID info is",
		gblInfo.lldpFrame.ChassisID))
	gblInfo.logger.Info(fmt.Sprintln("PortID info is",
		gblInfo.lldpFrame.PortID))
	gblInfo.logger.Info(fmt.Sprintln("TTL info is", gblInfo.lldpFrame.TTL))
	gblInfo.logger.Info(fmt.Sprintln("Optional Values is",
		gblInfo.lldpLinkInfo))
}

/* process incoming pkt from peer
 */
func (svr *LLDPServer) ProcessRxPkt(pkt gopacket.Packet, ifIndex int32) {
	//Sanity check for port up or down
	gblInfo, exists := svr.lldpGblInfo[ifIndex]
	if !exists {
		//@FIXME: is this bad???
		svr.logger.Info(fmt.Sprintln("No entry for", ifIndex,
			"during processing packet"))
		return
	}
	ethernetLayer := pkt.Layer(layers.LayerTypeEthernet)
	if ethernetLayer == nil {
		return
	}
	eth := ethernetLayer.(*layers.Ethernet)
	// copy src mac and dst mac
	gblInfo.SrcMAC = eth.SrcMAC
	gblInfo.DstMAC = eth.DstMAC

	lldpLayer := pkt.Layer(layers.LayerTypeLinkLayerDiscovery)
	if gblInfo.lldpFrame == nil {
		gblInfo.lldpFrame = new(layers.LinkLayerDiscovery)
	}
	// Store lldp frame information received from direct connection
	*gblInfo.lldpFrame = *lldpLayer.(*layers.LinkLayerDiscovery)
	lldpLayerInfo := pkt.Layer(layers.LayerTypeLinkLayerDiscoveryInfo)

	if gblInfo.lldpLinkInfo == nil {
		gblInfo.lldpLinkInfo = new(layers.LinkLayerDiscoveryInfo)
	}
	// Store lldp link layer optional tlv information
	*gblInfo.lldpLinkInfo = *lldpLayerInfo.(*layers.LinkLayerDiscoveryInfo)
	svr.lldpGblInfo[ifIndex] = gblInfo
	// reset/start timer for recipient information
	svr.CheckPeerEntry(ifIndex)
	gblInfo.DumpFrame()
}

/*  Upon receiving incoming packet check whether all the layers which we want
 *  are received or not.. If not then treat the packet as corrupted and move on
 */
func (svr *LLDPServer) VerifyLayers(pkt gopacket.Packet) error {
	wantedLayers := []gopacket.LayerType{layers.LayerTypeEthernet,
		layers.LayerTypeLinkLayerDiscovery,
		layers.LayerTypeLinkLayerDiscoveryInfo,
	}
	rcvdLayers := pkt.Layers()
	if len(rcvdLayers) != len(wantedLayers) {
		return LLDP_INVALID_LAYERS
	}
	for idx, layer := range rcvdLayers {
		if layer.LayerType() != wantedLayers[idx] {
			errString := fmt.Sprintln("Layer", idx, "mismatch: got",
				layer.LayerType(), "wanted", wantedLayers[idx])
			return errors.New(errString)
		}
	}
	return nil
}

/*
 *  Handle TTL timer. Once the timer expires, we will delete the remote entry
 *  if timer is running then reset the value
 */
func (svr *LLDPServer) CheckPeerEntry(ifIndex int32) {
	gblInfo, exists := svr.lldpGblInfo[ifIndex]
	if !exists {
		svr.logger.Err(fmt.Sprintln("No object found for ifIndex:",
			ifIndex))
		return
	}
	if gblInfo.clearCacheTimer != nil {
		// timer is running reset the time so that it doesn't expire
		gblInfo.clearCacheTimer.Reset(time.Duration(
			gblInfo.lldpFrame.TTL) * time.Second)
	} else {
		var clearPeerInfo_func func()
		// On timer expiration we will delete peer info and set it to nil
		clearPeerInfo_func = func() {
			svr.logger.Info("Recipient info delete timer expired for " +
				"peer connected to port " + gblInfo.Name +
				" and hence deleting peer information from runtime")
			gblInfo.lldpFrame = nil
			gblInfo.lldpLinkInfo = nil
		}
		// First time start function
		gblInfo.clearCacheTimer = time.AfterFunc(
			time.Duration(gblInfo.lldpFrame.TTL)*time.Second,
			clearPeerInfo_func)
	}
	svr.lldpGblInfo[ifIndex] = gblInfo
}
