// rx_test.go
// This is a test file to test the rx/portrcvfsm
package stp

import (
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"net"
	//"strconv"
	//"strings"
	"testing"
	"time"
)

var TEST_RX_PORT_CONFIG_IFINDEX int32
var TEST_TX_PORT_CONFIG_IFINDEX int32

func InitPortConfigTest() {

	if PortConfigMap == nil {
		PortConfigMap = make(map[int32]portConfig)
	}
	// In order to test a packet we must listen on loopback interface
	// and send on interface we expect to receive on.  In order
	// to do this a couple of things must occur the PortConfig
	// must be updated with "dummy" ifindex pointing to 'lo'
	TEST_RX_PORT_CONFIG_IFINDEX = 0x0ADDBEEF
	PortConfigMap[TEST_RX_PORT_CONFIG_IFINDEX] = portConfig{Name: "lo"}
	PortConfigMap[TEST_TX_PORT_CONFIG_IFINDEX] = portConfig{Name: "lo"}
	/*
		intfs, err := net.Interfaces()
		if err == nil {
			for _, intf := range intfs {
				if strings.Contains(intf.Name, "eth") {
					ifindex, _ := strconv.Atoi(strings.Split(intf.Name, "eth")[1])
					if ifindex == 0 {
						TEST_TX_PORT_CONFIG_IFINDEX = int32(ifindex)
					}
					PortConfigMap[int32(ifindex)] = portConfig{Name: intf.Name}
				}
			}
		}
	*/
}

func SendValidStpFrame(txifindex int32, t *testing.T) {
	ifname, _ := PortConfigMap[TEST_TX_PORT_CONFIG_IFINDEX]
	handle, err := pcap.OpenLive(ifname.Name, 65536, false, 50*time.Millisecond)
	if err != nil {
		t.Error("Error opening pcap TX interface", TEST_TX_PORT_CONFIG_IFINDEX, ifname.Name, err)
		return
	}
	//txIface, _ := net.InterfaceByName(ifname.Name)

	eth := layers.Ethernet{
		SrcMAC: net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x66},
		DstMAC: layers.BpduDMAC,
		// length
		EthernetType: layers.EthernetTypeLLC,
		Length:       uint16(layers.STPProtocolLength + 3), // found from PCAP from packetlife.net
	}

	llc := layers.LLC{
		DSAP:    0x42,
		IG:      false,
		SSAP:    0x42,
		CR:      false,
		Control: 0x03,
	}

	stp := layers.STP{
		ProtocolId:        layers.RSTPProtocolIdentifier,
		ProtocolVersionId: layers.STPProtocolVersion,
		BPDUType:          byte(layers.BPDUTypeSTP),
		Flags:             0,
		RootId:            [8]byte{0x80, 0x01, 0x00, 0x19, 0x06, 0xEA, 0xB8, 0x80},
		RootCostPath:      1,
		BridgeId:          [8]byte{0x80, 0x01, 0x00, 0x19, 0x06, 0xEA, 0xB8, 0x80},
		PortId:            0x1111,
		MsgAge:            0,
		MaxAge:            20,
		HelloTime:         2,
		FwdDelay:          15,
	}

	// Set up buffer and options for serialization.
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}
	// Send one packet for every address.
	gopacket.SerializeLayers(buf, opts, &eth, &llc, &stp)
	if err = handle.WritePacketData(buf.Bytes()); err != nil {
		t.Error("Error writing packet to interface")
	}
	fmt.Println("Sent packet on interface", ifname.Name)
	handle.Close()
	handle = nil
}

func SendValidRStpFrame(txifindex int32, t *testing.T) {
	ifname, _ := PortConfigMap[TEST_TX_PORT_CONFIG_IFINDEX]
	handle, err := pcap.OpenLive(ifname.Name, 65536, false, 50*time.Millisecond)
	if err != nil {
		t.Error("Error opening pcap TX interface", TEST_TX_PORT_CONFIG_IFINDEX, ifname.Name, err)
		return
	}
	//txIface, _ := net.InterfaceByName(ifname.Name)
	
	eth := layers.Ethernet{
		SrcMAC: net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x66},
		DstMAC: layers.BpduDMAC,
		// length
		EthernetType: layers.EthernetTypeLLC,
		Length:       uint16(layers.STPProtocolLength + 3), // found from PCAP from packetlife.net
	}

	llc := layers.LLC{
		DSAP:    0x42,
		IG:      false,
		SSAP:    0x42,
		CR:      false,
		Control: 0x03,
	}

	stp := layers.RSTP{
		ProtocolId:        layers.RSTPProtocolIdentifier,
		ProtocolVersionId: layers.RSTPProtocolVersion,
		BPDUType:          byte(layers.BPDUTypeRSTP),
		Flags:             0,
		RootId:            [8]byte{0x80, 0x01, 0x00, 0x19, 0x06, 0xEA, 0xB8, 0x80},
		RootCostPath:      1,
		BridgeId:          [8]byte{0x80, 0x01, 0x00, 0x19, 0x06, 0xEA, 0xB8, 0x80},
		PortId:            0x1111,
		MsgAge:            0,
		MaxAge:            20,
		HelloTime:         2,
		FwdDelay:          15,
		Version1Length:    0,
	}

	StpSetBpduFlags(0, 0, 0, 0, PortRoleDesignatedPort, 1, 0, &stp.Flags)

	// Set up buffer and options for serialization.
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}
	// Send one packet for every address.
	gopacket.SerializeLayers(buf, opts, &eth, &llc, &stp)
	if err = handle.WritePacketData(buf.Bytes()); err != nil {
		t.Error("Error writing packet to interface")
	}
	fmt.Println("Sent packet on interface", ifname.Name)
	handle.Close()
	handle = nil
}
func SendInvalidStpFrame(txifindex int32, stp *layers.STP, t *testing.T) {
	ifname, _ := PortConfigMap[TEST_TX_PORT_CONFIG_IFINDEX]
	handle, err := pcap.OpenLive(ifname.Name, 65536, false, 50*time.Millisecond)
	if err != nil {
		t.Error("Error opening pcap TX interface", TEST_TX_PORT_CONFIG_IFINDEX, ifname.Name, err)
		return
	}
	//txIface, _ := net.InterfaceByName(ifname.Name)

	eth := layers.Ethernet{
		SrcMAC: net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x66},
		DstMAC: layers.BpduDMAC,
		// length
		EthernetType: layers.EthernetTypeLLC,
		Length:       uint16(layers.STPProtocolLength + 3), // found from PCAP from packetlife.net
	}

	llc := layers.LLC{
		DSAP:    0x42,
		IG:      false,
		SSAP:    0x42,
		CR:      false,
		Control: 0x03,
	}

	StpSetBpduFlags(0, 0, 0, 0, PortRoleDesignatedPort, 1, 0, &stp.Flags)

	// Set up buffer and options for serialization.
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}
	// Send one packet for every address.
	gopacket.SerializeLayers(buf, opts, &eth, &llc, stp)
	if err = handle.WritePacketData(buf.Bytes()); err != nil {
		t.Error("Error writing packet to interface")
	}
	fmt.Println("Sent packet on interface", ifname.Name)
	handle.Close()
	handle = nil
}

func SendInvalidRStpFrame(txifindex int32, rstp *layers.RSTP, t *testing.T) {
	ifname, _ := PortConfigMap[TEST_TX_PORT_CONFIG_IFINDEX]
	handle, err := pcap.OpenLive(ifname.Name, 65536, false, 50*time.Millisecond)
	if err != nil {
		t.Error("Error opening pcap TX interface", TEST_TX_PORT_CONFIG_IFINDEX, ifname.Name, err)
		return
	}
	//txIface, _ := net.InterfaceByName(ifname.Name)

	eth := layers.Ethernet{
		SrcMAC: net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x66},
		DstMAC: layers.BpduDMAC,
		// length
		EthernetType: layers.EthernetTypeLLC,
		Length:       uint16(layers.STPProtocolLength + 3), // found from PCAP from packetlife.net
	}

	llc := layers.LLC{
		DSAP:    0x42,
		IG:      false,
		SSAP:    0x42,
		CR:      false,
		Control: 0x03,
	}

	StpSetBpduFlags(0, 0, 0, 0, PortRoleDesignatedPort, 1, 0, &rstp.Flags)

	// Set up buffer and options for serialization.
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}
	// Send one packet for every address.
	gopacket.SerializeLayers(buf, opts, &eth, &llc, rstp)
	if err = handle.WritePacketData(buf.Bytes()); err != nil {
		t.Error("Error writing packet to interface")
	}
	fmt.Println("Sent packet on interface", ifname.Name)
	handle.Close()
	handle = nil
}

func TestValidStpPacket(t *testing.T) {
	InitPortConfigTest()

	// configure a port
	stpconfig := &StpPortConfig{
		Dot1dStpPortKey:               TEST_RX_PORT_CONFIG_IFINDEX,
		Dot1dStpPortPriority:          0x80,
		Dot1dStpPortEnable:            true,
		Dot1dStpPortPathCost:          1,
		Dot1dStpPortPathCost32:        1,
		Dot1dStpPortProtocolMigration: 0,
		Dot1dStpPortAdminPointToPoint: StpPointToPointForceFalse,
		Dot1dStpPortAdminEdgePort:     0,
		Dot1dStpPortAdminPathCost:     0,
	}

	// create a port
	p := NewStpPort(stpconfig)

	// lets only start the Port Receive State Machine
	p.PrxmMachineMain()

	// going just send event and not start main as we just did above
	p.BEGIN(true)

	if p.PrxmMachineFsm.Machine.Curr.PreviousState() != PrxmStateNone {
		t.Error("Failed to Initial Rx machine state not set correctly", p.PrxmMachineFsm.Machine.Curr.PreviousState())
		t.FailNow()
	}

	if p.PrxmMachineFsm.Machine.Curr.CurrentState() != PrxmStateDiscard {
		t.Error("Failed to transition from None to Discard State")
		t.FailNow()
	}

	// send a packet
	SendValidStpFrame(TEST_TX_PORT_CONFIG_IFINDEX, t)

	testWait := make(chan bool)
	// may need to delay a bit in order to allow for packet to be receive
	// by pcap
	go func() {
		for i := 0; i < 10 &&
			(p.BpduRx == 0); i++ {
			time.Sleep(time.Millisecond * 20)
		}
		testWait <- true
	}()

	<-testWait

	fmt.Println("packet cnt", p.BpduRx)

	if p.RcvdBPDU == true {
		t.Error("Failed to receive RcvdBPDU is set")
		t.FailNow()
	}

	if p.OperEdge == true {
		t.Error("Failed to receive OperEdge is set")
		t.FailNow()
	}

	if p.RcvdSTP != true {
		t.Error("Failed to receive RcvdSTP not set")
		t.FailNow()
	}
	if p.RcvdRSTP == true {
		t.Error("Failed received RcvdRSTP is set")
		t.FailNow()
	}

	if p.RcvdMsg != true {
		t.Error("Failed received RcvdMsg not set")
		t.FailNow()
	}

	if p.EdgeDelayWhileTimer.count != MigrateTimeDefault {
		t.Error("Failed EdgeDelayWhiletimer tick count not set to MigrateTimeDefault")
		t.FailNow()
	}

	if p.PrxmMachineFsm.Machine.Curr.CurrentState() != PrxmStateReceive {
		t.Error("Failed to transition state to Receive")
		t.FailNow()
	}

	DelStpPort(p)

}

func TestValidRStpPacket(t *testing.T) {
	InitPortConfigTest()

	// configure a port
	stpconfig := &StpPortConfig{
		Dot1dStpPortKey:               TEST_RX_PORT_CONFIG_IFINDEX,
		Dot1dStpPortPriority:          0x80,
		Dot1dStpPortEnable:            true,
		Dot1dStpPortPathCost:          1,
		Dot1dStpPortPathCost32:        1,
		Dot1dStpPortProtocolMigration: 0,
		Dot1dStpPortAdminPointToPoint: StpPointToPointForceFalse,
		Dot1dStpPortAdminEdgePort:     0,
		Dot1dStpPortAdminPathCost:     0,
	}

	// create a port
	p := NewStpPort(stpconfig)

	// lets only start the Port Receive State Machine
	p.PrxmMachineMain()

	// going just send event and not start main as we just did above
	p.BEGIN(true)

	if p.PrxmMachineFsm.Machine.Curr.PreviousState() != PrxmStateNone {
		t.Error("Failed to Initial Rx machine state not set correctly", p.PrxmMachineFsm.Machine.Curr.PreviousState())
		t.FailNow()
	}

	if p.PrxmMachineFsm.Machine.Curr.CurrentState() != PrxmStateDiscard {
		t.Error("Failed to transition from None to Discard State")
		t.FailNow()
	}

	// send a packet
	SendValidRStpFrame(TEST_TX_PORT_CONFIG_IFINDEX, t)

	testWait := make(chan bool)
	// may need to delay a bit in order to allow for packet to be receive
	// by pcap
	go func() {
		for i := 0; i < 10 &&
			(p.BpduRx == 0); i++ {
			time.Sleep(time.Millisecond * 20)
		}
		testWait <- true
	}()

	<-testWait

	fmt.Println("packet cnt", p.BpduRx)

	if p.RcvdBPDU == true {
		t.Error("Failed to receive RcvdBPDU is set")
		t.FailNow()
	}

	if p.OperEdge == true {
		t.Error("Failed to receive OperEdge is set")
		t.FailNow()
	}

	if p.RcvdSTP != false {
		t.Error("Failed to receive RcvdSTP is set")
		t.FailNow()
	}
	if p.RcvdRSTP != true {
		t.Error("Failed received RcvdRSTP not set")
		t.FailNow()
	}

	if p.RcvdMsg != true {
		t.Error("Failed received RcvdMsg not set")
		t.FailNow()
	}

	if p.EdgeDelayWhileTimer.count != MigrateTimeDefault {
		t.Error("Failed EdgeDelayWhiletimer tick count not set to MigrateTimeDefault")
		t.FailNow()
	}

	if p.PrxmMachineFsm.Machine.Curr.CurrentState() != PrxmStateReceive {
		t.Error("Failed to transition state to Receive")
		t.FailNow()
	}

	DelStpPort(p)
}

func TestInvalidRStpPacketBPDUTypeInvalid(t *testing.T) {
	InitPortConfigTest()

	// configure a port
	stpconfig := &StpPortConfig{
		Dot1dStpPortKey:               TEST_RX_PORT_CONFIG_IFINDEX,
		Dot1dStpPortPriority:          0x80,
		Dot1dStpPortEnable:            true,
		Dot1dStpPortPathCost:          1,
		Dot1dStpPortPathCost32:        1,
		Dot1dStpPortProtocolMigration: 0,
		Dot1dStpPortAdminPointToPoint: StpPointToPointForceFalse,
		Dot1dStpPortAdminEdgePort:     0,
		Dot1dStpPortAdminPathCost:     0,
	}

	// create a port
	p := NewStpPort(stpconfig)

	// lets only start the Port Receive State Machine
	p.PrxmMachineMain()

	// going just send event and not start main as we just did above
	p.BEGIN(true)

	if p.PrxmMachineFsm.Machine.Curr.PreviousState() != PrxmStateNone {
		t.Error("Failed to Initial Rx machine state not set correctly", p.PrxmMachineFsm.Machine.Curr.PreviousState())
		t.FailNow()
	}

	if p.PrxmMachineFsm.Machine.Curr.CurrentState() != PrxmStateDiscard {
		t.Error("Failed to transition from None to Discard State")
		t.FailNow()
	}

	// send a packet
	rstp := layers.RSTP{
		ProtocolId:        layers.RSTPProtocolIdentifier,
		ProtocolVersionId: layers.RSTPProtocolVersion,
		BPDUType:          byte(layers.BPDUTypeSTP),
		Flags:             0,
		RootId:            [8]byte{0x80, 0x01, 0x00, 0x19, 0x06, 0xEA, 0xB8, 0x80},
		RootCostPath:      1,
		BridgeId:          [8]byte{0x80, 0x01, 0x00, 0x19, 0x06, 0xEA, 0xB8, 0x80},
		PortId:            0x1111,
		MsgAge:            0,
		MaxAge:            20,
		HelloTime:         2,
		FwdDelay:          15,
		Version1Length:    0,
	}

	SendInvalidRStpFrame(TEST_TX_PORT_CONFIG_IFINDEX, &rstp, t)

	testWait := make(chan bool)
	// may need to delay a bit in order to allow for packet to be receive
	// by pcap
	go func() {
		for i := 0; i < 10 &&
			(p.BpduRx == 0); i++ {
			time.Sleep(time.Millisecond * 20)
		}
		testWait <- true
	}()

	<-testWait

	fmt.Println("packet cnt", p.BpduRx)

	if p.RcvdBPDU == true {
		t.Error("Failed to receive RcvdBPDU is set")
		t.FailNow()
	}

	if p.OperEdge == true {
		t.Error("Failed to receive OperEdge is set")
		t.FailNow()
	}

	if p.RcvdSTP != false {
		t.Error("Failed to receive RcvdSTP is set")
		t.FailNow()
	}
	if p.RcvdRSTP != false {
		t.Error("Failed received RcvdRSTP is set")
		t.FailNow()
	}

	if p.RcvdMsg != false {
		t.Error("Failed received RcvdMsg not set")
		t.FailNow()
	}

	if p.EdgeDelayWhileTimer.count != MigrateTimeDefault {
		t.Error("Failed EdgeDelayWhiletimer tick count not set to MigrateTimeDefault")
		t.FailNow()
	}

	DelStpPort(p)
}

func TestInvalidRStpPacketProtocolVersionInvalid(t *testing.T) {
	InitPortConfigTest()

	// configure a port
	stpconfig := &StpPortConfig{
		Dot1dStpPortKey:               TEST_RX_PORT_CONFIG_IFINDEX,
		Dot1dStpPortPriority:          0x80,
		Dot1dStpPortEnable:            true,
		Dot1dStpPortPathCost:          1,
		Dot1dStpPortPathCost32:        1,
		Dot1dStpPortProtocolMigration: 0,
		Dot1dStpPortAdminPointToPoint: StpPointToPointForceFalse,
		Dot1dStpPortAdminEdgePort:     0,
		Dot1dStpPortAdminPathCost:     0,
	}

	// create a port
	p := NewStpPort(stpconfig)

	// lets only start the Port Receive State Machine
	p.PrxmMachineMain()

	// going just send event and not start main as we just did above
	p.BEGIN(true)

	if p.PrxmMachineFsm.Machine.Curr.PreviousState() != PrxmStateNone {
		t.Error("Failed to Initial Rx machine state not set correctly", p.PrxmMachineFsm.Machine.Curr.PreviousState())
		t.FailNow()
	}

	if p.PrxmMachineFsm.Machine.Curr.CurrentState() != PrxmStateDiscard {
		t.Error("Failed to transition from None to Discard State")
		t.FailNow()
	}

	// send a packet
	rstp := layers.RSTP{
		ProtocolId:        layers.RSTPProtocolIdentifier,
		ProtocolVersionId: layers.STPProtocolVersion,
		BPDUType:          byte(layers.BPDUTypeRSTP),
		Flags:             0,
		RootId:            [8]byte{0x80, 0x01, 0x00, 0x19, 0x06, 0xEA, 0xB8, 0x80},
		RootCostPath:      1,
		BridgeId:          [8]byte{0x80, 0x01, 0x00, 0x19, 0x06, 0xEA, 0xB8, 0x80},
		PortId:            0x1111,
		MsgAge:            0,
		MaxAge:            20,
		HelloTime:         2,
		FwdDelay:          15,
		Version1Length:    0,
	}

	SendInvalidRStpFrame(TEST_TX_PORT_CONFIG_IFINDEX, &rstp, t)

	testWait := make(chan bool)
	// may need to delay a bit in order to allow for packet to be receive
	// by pcap
	go func() {
		for i := 0; i < 10 &&
			(p.BpduRx == 0); i++ {
			time.Sleep(time.Millisecond * 20)
		}
		testWait <- true
	}()

	<-testWait

	fmt.Println("packet cnt", p.BpduRx)

	if p.RcvdBPDU == true {
		t.Error("Failed to receive RcvdBPDU is set")
		t.FailNow()
	}

	if p.OperEdge == true {
		t.Error("Failed to receive OperEdge is set")
		t.FailNow()
	}

	if p.RcvdSTP != false {
		t.Error("Failed to receive RcvdSTP is set")
		t.FailNow()
	}
	if p.RcvdRSTP != false {
		t.Error("Failed received RcvdRSTP is set")
		t.FailNow()
	}

	if p.RcvdMsg != false {
		t.Error("Failed received RcvdMsg not set")
		t.FailNow()
	}

	if p.EdgeDelayWhileTimer.count != MigrateTimeDefault {
		t.Error("Failed EdgeDelayWhiletimer tick count not set to MigrateTimeDefault")
		t.FailNow()
	}

	DelStpPort(p)
}

func TestInvalidStpPacketMsgAgeGreaterMaxAge(t *testing.T) {
	InitPortConfigTest()

	// configure a port
	stpconfig := &StpPortConfig{
		Dot1dStpPortKey:               TEST_RX_PORT_CONFIG_IFINDEX,
		Dot1dStpPortPriority:          0x80,
		Dot1dStpPortEnable:            true,
		Dot1dStpPortPathCost:          1,
		Dot1dStpPortPathCost32:        1,
		Dot1dStpPortProtocolMigration: 0,
		Dot1dStpPortAdminPointToPoint: StpPointToPointForceFalse,
		Dot1dStpPortAdminEdgePort:     0,
		Dot1dStpPortAdminPathCost:     0,
	}

	// create a port
	p := NewStpPort(stpconfig)

	// lets only start the Port Receive State Machine
	p.PrxmMachineMain()

	// going just send event and not start main as we just did above
	p.BEGIN(true)

	if p.PrxmMachineFsm.Machine.Curr.PreviousState() != PrxmStateNone {
		t.Error("Failed to Initial Rx machine state not set correctly", p.PrxmMachineFsm.Machine.Curr.PreviousState())
		t.FailNow()
	}

	if p.PrxmMachineFsm.Machine.Curr.CurrentState() != PrxmStateDiscard {
		t.Error("Failed to transition from None to Discard State")
		t.FailNow()
	}

	// send a packet
	stp := layers.STP{
		ProtocolId:        layers.RSTPProtocolIdentifier,
		ProtocolVersionId: layers.RSTPProtocolVersion,
		BPDUType:          byte(layers.BPDUTypeRSTP),
		Flags:             0,
		RootId:            [8]byte{0x80, 0x01, 0x00, 0x19, 0x06, 0xEA, 0xB8, 0x80},
		RootCostPath:      1,
		BridgeId:          [8]byte{0x80, 0x01, 0x00, 0x19, 0x06, 0xEA, 0xB8, 0x80},
		PortId:            0x1111,
		MsgAge:            21,
		MaxAge:            20,
		HelloTime:         2,
		FwdDelay:          15,
	}

	SendInvalidStpFrame(TEST_TX_PORT_CONFIG_IFINDEX, &stp, t)

	testWait := make(chan bool)
	// may need to delay a bit in order to allow for packet to be receive
	// by pcap
	go func() {
		for i := 0; i < 10 &&
			(p.BpduRx == 0); i++ {
			time.Sleep(time.Millisecond * 20)
		}
		testWait <- true
	}()

	<-testWait

	fmt.Println("packet cnt", p.BpduRx)

	if p.RcvdBPDU == true {
		t.Error("Failed to receive RcvdBPDU is set")
		t.FailNow()
	}

	if p.OperEdge == true {
		t.Error("Failed to receive OperEdge is set")
		t.FailNow()
	}

	if p.RcvdSTP != false {
		t.Error("Failed to receive RcvdSTP is set")
		t.FailNow()
	}
	if p.RcvdRSTP != false {
		t.Error("Failed received RcvdRSTP is set")
		t.FailNow()
	}

	if p.RcvdMsg != false {
		t.Error("Failed received RcvdMsg not set")
		t.FailNow()
	}

	if p.EdgeDelayWhileTimer.count != MigrateTimeDefault {
		t.Error("Failed EdgeDelayWhiletimer tick count not set to MigrateTimeDefault")
		t.FailNow()
	}

	DelStpPort(p)
}
