package lldpServer

import (
	"errors"
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

/* Go routine to recieve lldp frames. This go routine is created for all the
 * ports which are in up state.
 */
func (svr *LLDPServer) LLDPReceiveFrames(pHandle *pcap.Handle, ifIndex int32) {
	pktSrc := gopacket.NewPacketSource(pHandle, pHandle.LinkType())
	in := pktSrc.Packets()
	for { //pkt := range pktSrc.Packets() {
		select {
		case pkt, ok := <-in:
			if !ok {
				svr.logger.Info(fmt.Sprintln("Port is down exiting",
					"receive frames for", ifIndex))
				return
			}
			svr.lldpRxPktCh <- LLDPInPktChannel{
				pkt:     pkt,
				ifIndex: ifIndex,
			}
		}
	}
}

/* process incoming pkt from peer
 */
func (svr *LLDPServer) LLDPProcessRxPkt(pkt gopacket.Packet, ifIndex int32) {
	//Sanity check for port up or down
	_, exists := svr.lldpGblInfo[ifIndex]
	if !exists {
		//@FIXME: is this bad???
		svr.logger.Info(fmt.Sprintln("No entry for", ifIndex))
		return
	}
	ethernetLayer := pkt.Layer(layers.LayerTypeEthernet)
	if ethernetLayer == nil {
		return
	}
	eth := ethernetLayer.(*layers.Ethernet)
	lldpLayer := pkt.Layer(layers.LayerTypeLinkLayerDiscovery)
	lldpFrame := lldpLayer.(*layers.LinkLayerDiscovery)
	svr.logger.Info(fmt.Sprintln("SrcMAC:", eth.SrcMAC.String(),
		"DstMAC:", eth.DstMAC.String()))
	svr.logger.Info(fmt.Sprintln("ChassisID info is", lldpFrame.ChassisID))
}

/*  Upon receiving incoming packet check whether all the layers which we want
 *  are received or not.. If not then treat the packet as corrupted and move on
 */
func (svr *LLDPServer) LLDPVerifyLayers(pkt gopacket.Packet) error {
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
