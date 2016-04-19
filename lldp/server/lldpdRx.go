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
		gblInfo.rxFrame.ChassisID))
	gblInfo.logger.Info(fmt.Sprintln("PortID info is",
		gblInfo.rxFrame.PortID))
	gblInfo.logger.Info(fmt.Sprintln("TTL info is", gblInfo.rxFrame.TTL))
	gblInfo.logger.Info(fmt.Sprintln("Optional Values is",
		gblInfo.rxLinkInfo))
}

/* process incoming pkt from peer
 */
func (gblInfo *LLDPGlobalInfo) ProcessRxPkt(pkt gopacket.Packet) {
	ethernetLayer := pkt.Layer(layers.LayerTypeEthernet)
	if ethernetLayer == nil {
		return
	}
	eth := ethernetLayer.(*layers.Ethernet)
	// copy src mac and dst mac
	gblInfo.SrcMAC = eth.SrcMAC
	if gblInfo.DstMAC.String() != eth.DstMAC.String() {
		gblInfo.logger.Err("Invalid DST MAC in received frame " +
			eth.DstMAC.String())
		return
	}

	lldpLayer := pkt.Layer(layers.LayerTypeLinkLayerDiscovery)
	lldpLayerInfo := pkt.Layer(layers.LayerTypeLinkLayerDiscoveryInfo)
	if lldpLayer == nil || lldpLayerInfo == nil {
		gblInfo.logger.Err("Invalid frame")
		return
	}
	if gblInfo.rxFrame == nil {
		gblInfo.rxFrame = new(layers.LinkLayerDiscovery)
	}
	// Store lldp frame information received from direct connection
	*gblInfo.rxFrame = *lldpLayer.(*layers.LinkLayerDiscovery)

	if gblInfo.rxLinkInfo == nil {
		gblInfo.rxLinkInfo = new(layers.LinkLayerDiscoveryInfo)
	}
	// Store lldp link layer optional tlv information
	*gblInfo.rxLinkInfo = *lldpLayerInfo.(*layers.LinkLayerDiscoveryInfo)
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
func (gblInfo *LLDPGlobalInfo) CheckPeerEntry() {
	if gblInfo.clearCacheTimer != nil {
		// timer is running reset the time so that it doesn't expire
		gblInfo.clearCacheTimer.Reset(time.Duration(
			gblInfo.rxFrame.TTL) * time.Second)
	} else {
		var clearPeerInfo_func func()
		// On timer expiration we will delete peer info and set it to nil
		clearPeerInfo_func = func() {
			gblInfo.logger.Info("Recipient info delete timer expired for " +
				"peer connected to port " + gblInfo.Name +
				" and hence deleting peer information from runtime")
			gblInfo.rxFrame = nil
			gblInfo.rxLinkInfo = nil
		}
		// First time start function
		gblInfo.clearCacheTimer = time.AfterFunc(
			time.Duration(gblInfo.rxFrame.TTL)*time.Second,
			clearPeerInfo_func)
	}
}
