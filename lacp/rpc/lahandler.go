// lahandler
package rpc

import (
	"encoding/hex"
	"errors"
	"fmt"
	lacp "l2/lacp/protocol"
	"lacpdServices"
	"net"
	"strconv"
	"strings"
	"time"
)

type LACPDServiceHandler struct {
}

func NewLACPDServiceHandler() *LACPDServiceHandler {
	return &LACPDServiceHandler{}
}

func ConvertStringToUint8Array(s string) [6]uint8 {
	var arr [6]uint8
	x, _ := hex.DecodeString(s)
	for k, v := range x {
		arr[k] = uint8(v)
	}
	return arr
}

func ConvertModelLagTypeToLaAggType(yangLagType int32) uint32 {
	var LagType uint32
	switch yangLagType {
	case 0: //LagAggregationTypeLACP:
		LagType = lacp.LaAggTypeLACP
		break
	case 1: //LagAggregationTypeSTATIC:
		LagType = lacp.LaAggTypeSTATIC
		break
	default:
		fmt.Println("ERROR: unknown LagType %d", yangLagType)
	}
	return LagType
}

func ConvertModelSpeedToLaAggSpeed(yangSpeed string) int {
	speedMap := map[string]int{
		"SPEED_100Gb":   1, //EthernetSpeedSPEED100Gb,
		"SPEED_10Gb":    2, //EthernetSpeedSPEED10Gb,
		"SPEED_40Gb":    3, //EthernetSpeedSPEED40Gb,
		"SPEED_25Gb":    4, //EthernetSpeedSPEED25Gb,
		"SPEED_1Gb":     5, //EthernetSpeedSPEED1Gb,
		"SPEED_100Mb":   6, //EthernetSpeedSPEED100Mb,
		"SPEED_10Mb":    7, //EthernetSpeedSPEED10Mb,
		"SPEED_UNKNOWN": 8, //EthernetSpeedSPEEDUNKNOWN
	}

	speed, err := speedMap[yangSpeed]
	if err {
		return 8 //EthernetSpeedSPEEDUNKNOWN
	}
	return speed
}

func ConvertModelLacpModeToLaAggMode(yangLacpMode int32) uint32 {
	var mode uint32
	if yangLacpMode == 0 {
		// ACTIVE
		mode = lacp.LacpModeActive
	} else {
		// PASSIVE
		mode = lacp.LacpModePassive
	}

	return mode
}

func ConvertModelLacpPeriodToLaAggInterval(yangInterval int32) time.Duration {
	var interval time.Duration
	switch yangInterval {
	case 0: // SLOW
		interval = lacp.LacpSlowPeriodicTime
		break
	case 1: // FAST
		interval = lacp.LacpFastPeriodicTime
		break
	default:
		interval = lacp.LacpSlowPeriodicTime
	}
	return interval
}

var gAggKeyMap map[string]uint16
var gAggKeyVal uint16
var gAggKeyFreeList []uint16

func GenerateKeyByAggName(aggName string) uint16 {
	var rKey uint16
	if len(gAggKeyFreeList) == 0 {
		gAggKeyVal += 1
		rKey = gAggKeyVal
	} else {
		rKey = gAggKeyFreeList[0]
		// remove element from list
		gAggKeyFreeList = append(gAggKeyFreeList[:0], gAggKeyFreeList[1:]...)
	}
	return rKey
}

func GetKeyByAggName(aggName string) uint16 {

	var key uint16
	if gAggKeyMap == nil {
		gAggKeyMap = make(map[string]uint16)
		gAggKeyFreeList = make([]uint16, 0)
	}

	if _, ok := gAggKeyMap[aggName]; ok {
		key = gAggKeyMap[aggName]
	} else {
		key = GenerateKeyByAggName(aggName)
		// store the newly generated key
		gAggKeyMap[aggName] = key
	}
	return key
}

// the id of the agg should be part of the name
// format expected <name>-<#number>
func GetIdByName(aggName string) int {
	var i int
	if strings.Contains(aggName, "-") {
		i, _ = strconv.Atoi(strings.Split(aggName, "-")[1])
	} else {
		i = 0
	}
	return i
}

// OPEN CONFIG YANG specific
// CreateAggreagationConfig will create a static lag
func (la LACPDServiceHandler) CreateAggregationConfig(config *lacpdServices.AggregationConfig) (bool, error) {
	//        1 : string      NameKey
	//        2 : i32         LagType
	//        3 : i16         MinLinks
	conf := &lacp.LaAggConfig{
		Id:  GetIdByName(config.NameKey),
		Key: GetKeyByAggName(config.NameKey),
		// Identifier of the lag
		Name: config.NameKey,
		// Type of LAG STATIC or LACP
		Type:     ConvertModelLagTypeToLaAggType(config.LagType),
		MinLinks: uint16(config.MinLinks),
	}
	lacp.CreateLaAgg(conf)
	return true, nil
}

func (la LACPDServiceHandler) DeleteAggregationConfig(config *lacpdServices.AggregationConfig) (bool, error) {
	return true, nil
}

func (la LACPDServiceHandler) UpdateAggregationConfig(config *lacpdServices.AggregationConfig) (bool, error) {
	return true, nil
}

// CreateAggregationLacpConfig will create an lacp lag
//	1 : i32 	LagType  (0 == LACP, 1 == STATIC)
//	2 : string 	Description
//	3 : bool 	Enabled
//	4 : i16 	Mtu
//	5 : i16 	MinLinks
//	6 : string 	Type
//	7 : string 	NameKey
//	8 : i32 	Interval (0 == LONG, 1 == SHORT)
//	9 : i32 	LacpMode (0 == ACTIVE, 1 == PASSIVE)
//	10 : string SystemIdMac
//	11 : i16 	SystemPriority
func (la LACPDServiceHandler) CreateAggregationLacpConfig(config *lacpdServices.AggregationLacpConfig) (bool, error) {

	var a *lacp.LaAggregator
	if lacp.LaFindAggByName(config.NameKey, &a) {

		// TODO, lets delete the previous lag???

	} else {

		conf := &lacp.LaAggConfig{
			Id:  GetIdByName(config.NameKey),
			Key: GetKeyByAggName(config.NameKey),
			// Identifier of the lag
			Name: config.NameKey,
			// Type of LAG STATIC or LACP
			Type:     ConvertModelLagTypeToLaAggType(config.LagType),
			MinLinks: uint16(config.MinLinks),
			Enabled:  config.Enabled,
			// lacp config
			Lacp: lacp.LacpConfigInfo{
				Interval:       ConvertModelLacpPeriodToLaAggInterval(config.Interval),
				Mode:           ConvertModelLacpModeToLaAggMode(config.LacpMode),
				SystemIdMac:    config.SystemIdMac,
				SystemPriority: uint16(config.SystemPriority),
			},
			Properties: lacp.PortProperties{
				Mtu: int(config.Mtu),
			},
		}
		lacp.CreateLaAgg(conf)
	}
	return true, nil
}

func (la LACPDServiceHandler) DeleteAggregationLacpConfig(config *lacpdServices.AggregationLacpConfig) (bool, error) {
	return true, nil
}

func (la LACPDServiceHandler) UpdateAggregationLacpConfig(config *lacpdServices.AggregationLacpConfig) (bool, error) {
	return true, nil
}

func (la LACPDServiceHandler) CreateEthernetConfig(config *lacpdServices.EthernetConfig) (bool, error) {

	yangModeMap := map[uint32]string{
		//LacpActivityTypeACTIVE:  "ACTIVE",
		//LacpActivityTypeSTANDBY: "STANDBY",
		0: "ACTIVE",
		1: "STANDBY",
	}
	yangTimeoutMap := map[uint32]string{
		//LacpPeriodTypeSLOW: "LONG",
		//LacpPeriodTypeFAST: "SHORT",
		0: "LONG",
		1: "SHORT",
	}
	//	1 : string 	NameKey
	//	2 : bool 	Enabled
	//	3 : string 	Description
	//	4 : i16 	Mtu
	//	5 : string 	Type
	//	6 : string 	MacAddress
	//	7 : i32 	DuplexMode
	//	8 : bool 	Auto
	//	9 : string 	Speed
	//	10 : bool 	EnableFlowControl
	//	11 : string 	AggregateId
	var a *lacp.LaAggregator
	if !lacp.LaFindAggByName(config.AggregateId, &a) {
		fmt.Println("\nDid not find agg", config.AggregatId)
		// lets create a port with some defaults
		la.CreateLaAggPort(
			lacpdServices.Uint16(GetIdByName(config.NameKey)),
			0, //Prio lacpdServices.Uint16,
			lacpdServices.Uint16(GetKeyByAggName(config.AggregateId)),
			lacpdServices.Int(GetIdByName(config.AggregateId)),
			config.Enabled,
			"ON",                //"ON", "ACTIVE", "PASSIVE"
			"LONG",              //"SHORT", "LONG"
			"00:00:00:00:00:00", // default
			lacpdServices.Int(ConvertModelSpeedToLaAggSpeed(config.Speed)),
			lacpdServices.Int(config.DuplexMode),
			lacpdServices.Int(config.Mtu),
			config.MacAddress,
			config.NameKey,
		)
	} else {
		fmt.Println("\nFound agg", config.AggregatId)
		// Found the lag lets add the lacp info to the port
		//	    2 : i32 	Interval
		// 	    3 : i32 	LacpMode
		//	    4 : string 	SystemIdMac
		//	    5 : i16 	SystemPriority
		mode, ok := yangModeMap[uint32(a.Config.Mode)]
		if !ok {
			mode = "ON"
		}
		timeout, ok := yangTimeoutMap[uint32(a.Config.Interval)]
		if !ok {
			timeout = "LONG"
		}

		la.CreateLaAggPort(
			lacpdServices.Uint16(GetIdByName(config.NameKey)),
			lacpdServices.Uint16(a.Config.SystemPriority),
			lacpdServices.Uint16(GetKeyByAggName(config.AggregateId)),
			lacpdServices.Int(GetIdByName(config.AggregateId)),
			config.Enabled,
			mode,
			timeout,
			config.MacAddress,
			lacpdServices.Int(ConvertModelSpeedToLaAggSpeed(config.Speed)),
			lacpdServices.Int(config.DuplexMode),
			lacpdServices.Int(config.Mtu),
			a.Config.SystemIdMac,
			config.NameKey,
		)
	}
	return true, nil
}

func (la LACPDServiceHandler) DeleteEthernetConfig(config *lacpdServices.EthernetConfig) (bool, error) {
	return true, nil
}

func (la LACPDServiceHandler) UpdateEthernetConfig(config *lacpdServices.EthernetConfig) (bool, error) {
	return true, nil
}

func (la LACPDServiceHandler) CreateLaAggregator(aggId lacpdServices.Int,
	actorAdminKey lacpdServices.Uint16,
	actorSystemId string,
	aggMacAddr string) (lacpdServices.Int, error) {

	conf := &lacp.LaAggConfig{
		// Aggregator_MAC_address
		Mac: ConvertStringToUint8Array(aggMacAddr),
		// Aggregator_Identifier
		Id: int(aggId),
		// Actor_Admin_Aggregator_Key
		Key: uint16(actorAdminKey),
	}

	lacp.CreateLaAgg(conf)
	return 0, nil
}

func (la LACPDServiceHandler) DeleteLaAggregator(aggId lacpdServices.Int) (lacpdServices.Int, error) {

	lacp.DeleteLaAgg(int(aggId))
	return 0, nil
}

func (la LACPDServiceHandler) CreateLaAggPort(Id lacpdServices.Uint16,
	Prio lacpdServices.Uint16,
	Key lacpdServices.Uint16,
	AggId lacpdServices.Int,
	Enable bool,
	Mode string, //"ON", "ACTIVE", "PASSIVE"
	Timeout string, //"SHORT", "LONG"
	Mac string,
	Speed lacpdServices.Int,
	Duplex lacpdServices.Int,
	Mtu lacpdServices.Int,
	SysId string,
	IntfId string) (lacpdServices.Int, error) {

	// TODO: check syntax on strings?
	t := lacp.LacpShortTimeoutTime
	if Timeout == "LONG" {
		t = lacp.LacpLongTimeoutTime
	}
	m := lacp.LacpModeOn
	if Mode == "ACTIVE" {
		m = lacp.LacpModeActive
	} else if Mode == "PASSIVE" {
		m = lacp.LacpModePassive
	}

	mac, _ := net.ParseMAC(Mac)
	sysId, _ := net.ParseMAC(SysId)
	conf := &lacp.LaAggPortConfig{
		Id:      uint16(Id),
		Prio:    uint16(Prio),
		AggId:   int(AggId),
		Key:     uint16(Key),
		Enable:  Enable,
		Mode:    m,
		Timeout: t,
		Properties: lacp.PortProperties{
			Mac:    mac,
			Speed:  int(Speed),
			Duplex: int(Duplex),
			Mtu:    int(Mtu),
		},
		TraceEna: true,
		IntfId:   IntfId,
		SysId:    sysId,
	}
	lacp.CreateLaAggPort(conf)
	return 0, nil
}

func (la LACPDServiceHandler) DeleteLaAggPort(Id lacpdServices.Uint16) (lacpdServices.Int, error) {
	var p *lacp.LaAggPort
	if lacp.LaFindPortById(uint16(Id), &p) {
		if p.AggId != 0 {
			lacp.DeleteLaAggPortFromAgg(p.AggId, uint16(Id))
		}
		lacp.DeleteLaAggPort(uint16(Id))
		return 0, nil
	}
	return 1, errors.New(fmt.Sprintf("LACP: Unable to find Port %d", Id))
}

func (la LACPDServiceHandler) SetLaAggPortMode(Id lacpdServices.Uint16, m string) (lacpdServices.Int, error) {

	supportedModes := map[string]int{"ON": lacp.LacpModeOn,
		"ACTIVE":  lacp.LacpModeActive,
		"PASSIVE": lacp.LacpModePassive}

	if mode, ok := supportedModes[m]; ok {
		var p *lacp.LaAggPort
		if lacp.LaFindPortById(uint16(Id), &p) {
			lacp.SetLaAggPortLacpMode(uint16(Id), mode, p.TimeoutGet())
		} else {
			return 1, errors.New(fmt.Sprintf("LACP: Mode not set,  Unable to find Port", Id))
		}
	} else {
		return 1, errors.New(fmt.Sprintf("LACP: Mode not set,  Unsupported mode %d", m, "supported modes: ", supportedModes))
	}

	return 0, nil
}

func (la LACPDServiceHandler) SetLaAggPortTimeout(Id lacpdServices.Uint16, t string) (lacpdServices.Int, error) {
	// user can supply the periodic time or the
	supportedTimeouts := map[string]time.Duration{"SHORT": lacp.LacpShortTimeoutTime,
		"LONG": lacp.LacpLongTimeoutTime}

	if timeout, ok := supportedTimeouts[t]; ok {
		var p *lacp.LaAggPort
		if lacp.LaFindPortById(uint16(Id), &p) {
			lacp.SetLaAggPortLacpMode(uint16(Id), p.ModeGet(), timeout)
		} else {
			return 1, errors.New(fmt.Sprintf("LACP: timeout not set,  Unable to find Port", Id))
		}
	} else {
		return 1, errors.New(fmt.Sprintf("LACP: Timeout not set,  Unsupported timeout %d", t, "supported modes: ", supportedTimeouts))
	}

	return 0, nil
}

// SetPortLacpLogEnable will enable on a per port basis logging
// modStr - PORT, RXM, TXM, PTXM, TXM, CDM, ALL
// modStr can be a string containing one or more of the above
func (la LACPDServiceHandler) SetPortLacpLogEnable(Id lacpdServices.Uint16, modStr string, ena bool) (lacpdServices.Int, error) {
	modules := make(map[string]chan bool)
	var p *lacp.LaAggPort
	if lacp.LaFindPortById(uint16(Id), &p) {
		modules["RXM"] = p.RxMachineFsm.RxmLogEnableEvent
		modules["TXM"] = p.TxMachineFsm.TxmLogEnableEvent
		modules["PTXM"] = p.PtxMachineFsm.PtxmLogEnableEvent
		modules["TXM"] = p.TxMachineFsm.TxmLogEnableEvent
		modules["CDM"] = p.CdMachineFsm.CdmLogEnableEvent
		modules["MUXM"] = p.MuxMachineFsm.MuxmLogEnableEvent

		for k, v := range modules {
			if strings.Contains(k, "PORT") || strings.Contains(k, "ALL") {
				p.EnableLogging(ena)
			}
			if strings.Contains(k, modStr) || strings.Contains(k, "ALL") {
				v <- ena
			}
		}
		return 0, nil
	}
	return 1, errors.New(fmt.Sprintf("LACP: LOG set failed,  Unable to find Port", Id))
}
