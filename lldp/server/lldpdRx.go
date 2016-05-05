package lldpServer

import (
	"errors"
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
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

/* process incoming pkt from peer
 */
func (gblInfo *LLDPGlobalInfo) ProcessRxPkt(pkt gopacket.Packet) error {
	ethernetLayer := pkt.Layer(layers.LayerTypeEthernet)
	if ethernetLayer == nil {
		return errors.New("Invalid eth layer")
	}
	eth := ethernetLayer.(*layers.Ethernet)
	// copy src mac and dst mac
	gblInfo.SrcMAC = eth.SrcMAC
	if gblInfo.DstMAC.String() != eth.DstMAC.String() {
		return errors.New("Invalid DST MAC in rx frame")
	}
	// Get lldp manadatory layer and optional info
	lldpLayer := pkt.Layer(layers.LayerTypeLinkLayerDiscovery)
	lldpLayerInfo := pkt.Layer(layers.LayerTypeLinkLayerDiscoveryInfo)
	// Verify that the information is not nil
	if lldpLayer == nil || lldpLayerInfo == nil {
		return errors.New("Invalid Frame")
	}

	// Verify that the mandatory layer info is indeed correct
	err := gblInfo.VerifyFrame(lldpLayer.(*layers.LinkLayerDiscovery))
	if err != nil {
		return err
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
	return nil
}

/*  Upon receiving incoming packet check whether all the madatory layer info is
 *  correct or not.. If not then treat the packet as corrupted and move on
 */
func (gblInfo *LLDPGlobalInfo) VerifyFrame(lldpInfo *layers.LinkLayerDiscovery) error {

	if lldpInfo.ChassisID.Subtype > layers.LLDPChassisIDSubTypeLocal {
		return errors.New("Invalid chassis id subtype")
	}

	if lldpInfo.PortID.Subtype > layers.LLDPPortIDSubtypeLocal {
		return errors.New("Invalid port id subtype")
	}

	if lldpInfo.TTL > uint16(LLDP_MAX_TTL) {
		return errors.New("Invalid TTL value")
	}
	return nil
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
			debug.Logger.Info("Recipient info delete timer expired for " +
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
