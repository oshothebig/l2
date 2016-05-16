package packet

import (
	"errors"
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	_ "github.com/google/gopacket/pcap"
	"l2/lldp/utils"
	"net"
	"time"
)

func RxInit() *RX {
	var err error
	rxInfo := &RX{}
	rxInfo.DstMAC, err = net.ParseMAC(LLDP_PROTO_DST_MAC)
	if err != nil {
		debug.Logger.Err(fmt.Sprintln("parsing lldp protocol Mac failed",
			err))
	}

	return rxInfo
}

/*  Upon receiving incoming packet check whether all the madatory layer info is
 *  correct or not.. If not then treat the packet as corrupted and move on
 */
func (p *RX) VerifyFrame(lldpInfo *layers.LinkLayerDiscovery) error {

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

func (p *RX) Process(gblInfo *RX, pkt gopacket.Packet) error {
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
	err := p.VerifyFrame(lldpLayer.(*layers.LinkLayerDiscovery))
	if err != nil {
		return err
	}
	if gblInfo.RxFrame == nil {
		gblInfo.RxFrame = new(layers.LinkLayerDiscovery)
	}
	// Store lldp frame information received from direct connection
	*gblInfo.RxFrame = *lldpLayer.(*layers.LinkLayerDiscovery)

	if gblInfo.RxLinkInfo == nil {
		gblInfo.RxLinkInfo = new(layers.LinkLayerDiscoveryInfo)
	}
	// Store lldp link layer optional tlv information
	*gblInfo.RxLinkInfo = *lldpLayerInfo.(*layers.LinkLayerDiscoveryInfo)
	return nil
}

/*
 *  Handle TTL timer. Once the timer expires, we will delete the remote entry
 *  if timer is running then reset the value
 */
func (gblInfo *RX) CheckPeerEntry(port string) {
	if gblInfo.ClearCacheTimer != nil {
		// timer is running reset the time so that it doesn't expire
		gblInfo.ClearCacheTimer.Reset(time.Duration(
			gblInfo.RxFrame.TTL) * time.Second)
	} else {
		var clearPeerInfo_func func()
		// On timer expiration we will delete peer info and set it to nil
		clearPeerInfo_func = func() {
			debug.Logger.Info("Recipient info delete timer expired for " +
				"peer connected to port " + port +
				" and hence deleting peer information from runtime")
			gblInfo.RxFrame = nil
			gblInfo.RxLinkInfo = nil
		}
		// First time start function
		gblInfo.ClearCacheTimer = time.AfterFunc(
			time.Duration(gblInfo.RxFrame.TTL)*time.Second,
			clearPeerInfo_func)
	}
}
