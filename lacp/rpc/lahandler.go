// lahandler
package rpc

import (
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	lacp "l2/lacp/protocol"
	"lacpd"
	"models"
	"net"
	"reflect"
	"strconv"
	"strings"
	"time"
)

const DBName string = "UsrConfDb.db"

type LACPDServiceHandler struct {
}

func NewLACPDServiceHandler() *LACPDServiceHandler {
	// link up/down events for now
	startEvtHandler()
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
	if yangLagType == 0 {
		LagType = lacp.LaAggTypeLACP
	} else {
		LagType = lacp.LaAggTypeSTATIC
	}
	return LagType
}

func ConvertLaAggTypeToModelLagType(aggType uint32) int32 {
	var LagType int32
	switch aggType {
	case lacp.LaAggTypeLACP:
		LagType = 0
		break
	case lacp.LaAggTypeSTATIC:
		LagType = 1
		break
	default:
		fmt.Println("ERROR: unknown LagType %d", aggType)
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

func ConvertLaAggModeToModelLacpMode(lacpMode uint32) int32 {
	var mode int32
	if lacpMode == lacp.LacpModeActive {
		// ACTIVE
		mode = 0
	} else {
		// PASSIVE
		mode = 1
	}

	return mode
}

func ConvertModelLacpPeriodToLaAggInterval(yangInterval int32) time.Duration {
	var interval time.Duration
	if yangInterval == 1 {
		interval = lacp.LacpSlowPeriodicTime
	} else {
		interval = lacp.LacpFastPeriodicTime
	}
	return interval
}

func ConvertLaAggIntervalToLacpPeriod(interval time.Duration) int32 {
	var period int32
	switch interval {
	case lacp.LacpSlowPeriodicTime:
		period = 1
		break
	case lacp.LacpFastPeriodicTime:
		period = 0
		break
	default:
		period = 0
	}
	return period
}

var gAggKeyMap map[string]uint16
var gAggKeyVal uint16
var gAggKeyFreeList []uint16

func GenerateKeyByAggName(AggName string) uint16 {
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

func GetKeyByAggName(AggName string) uint16 {

	var Key uint16
	if gAggKeyMap == nil {
		gAggKeyMap = make(map[string]uint16)
		gAggKeyFreeList = make([]uint16, 0)
	}

	if _, ok := gAggKeyMap[AggName]; ok {
		Key = gAggKeyMap[AggName]
	} else {
		Key = GenerateKeyByAggName(AggName)
		// store the newly generated Key
		gAggKeyMap[AggName] = Key
	}
	return Key
}

// the id of the agg should be part of the name
// format expected <name>-<#number>
func GetIdByName(AggName string) int {
	var i int
	if strings.Contains(AggName, "-") {
		i, _ = strconv.Atoi(strings.Split(AggName, "-")[1])
	} else {
		i = 0
	}
	return i
}

/*
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
*/
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
func (la LACPDServiceHandler) CreateAggregationLacpConfig(config *lacpd.AggregationLacpConfig) (bool, error) {

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
			HashMode: uint32(config.LagHash),
		}
		lacp.CreateLaAgg(conf)
	}
	return true, nil
}

func (la *LACPDServiceHandler) ReadConfigFromDB(filePath string) error {
	var dbPath string = filePath + DBName

	dbHdl, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		//h.logger.Err(fmt.Sprintf("Failed to open the DB at %s with error %s", dbPath, err))
		fmt.Printf("Failed to open the DB at %s with error %s", dbPath, err)
		return err
	}

	defer dbHdl.Close()

	dbObj := new(models.AggregationLacpConfig)
	aggObjList, err2 := dbObj.GetAllObjFromDb(dbHdl)
	if err2 == nil {
		for _, aggObj := range aggObjList {
			aggthriftobj := new(lacpd.AggregationLacpConfig)
			models.ConvertlacpdAggregationLacpConfigObjToThrift(aggObj, aggthriftobj)
			_, err = la.CreateAggregationLacpConfig(aggthriftobj)
			if err != nil {
				return err
			}
		}
	} else {
		fmt.Println("Error getting All AggregationLacpConfig objects")
		return err2
	}

	dbObj2 := new(models.EthernetConfig)
	portObjList, err3 := dbObj2.GetAllObjFromDb(dbHdl)
	if err3 == nil {
		for _, portObj := range portObjList {
			portthriftobj := new(lacpd.EthernetConfig)
			models.ConvertlacpdEthernetConfigObjToThrift(portObj, portthriftobj)
			_, err = la.CreateEthernetConfig(portthriftobj)
			if err != nil {
				return err
			}
		}
	} else {
		fmt.Println("Error getting All AggregationLacpConfig objects")
		return err3
	}

	return nil
}

func (la LACPDServiceHandler) DeleteAggregationLacpConfig(config *lacpd.AggregationLacpConfig) (bool, error) {
	return true, nil
}

func (la LACPDServiceHandler) UpdateAggregationLacpConfig(origconfig *lacpd.AggregationLacpConfig, updateconfig *lacpd.AggregationLacpConfig, attrset []bool) (bool, error) {
	objTyp := reflect.TypeOf(*origconfig)
	//objVal := reflect.ValueOf(origconfig)
	//updateObjVal := reflect.ValueOf(*updateconfig)

	conf := &lacp.LaAggConfig{
		Id:  GetIdByName(updateconfig.NameKey),
		Key: GetKeyByAggName(updateconfig.NameKey),
		// Identifier of the lag
		Name: updateconfig.NameKey,
		// Type of LAG STATIC or LACP
		Type:     ConvertModelLagTypeToLaAggType(updateconfig.LagType),
		MinLinks: uint16(updateconfig.MinLinks),
		Enabled:  updateconfig.Enabled,
		// lacp config
		Lacp: lacp.LacpConfigInfo{
			Interval:       ConvertModelLacpPeriodToLaAggInterval(updateconfig.Interval),
			Mode:           ConvertModelLacpModeToLaAggMode(updateconfig.LacpMode),
			SystemIdMac:    updateconfig.SystemIdMac,
			SystemPriority: uint16(updateconfig.SystemPriority),
		},
		Properties: lacp.PortProperties{
			Mtu: int(updateconfig.Mtu),
		},
		HashMode: uint32(updateconfig.LagHash),
	}

	// important to note that the attrset starts at index 0 which is the BaseObj
	// which is not the first element on the thrift obj, thus we need to skip
	// this attribute
	for i := 0; i < objTyp.NumField(); i++ {
		objName := objTyp.Field(i).Name
		//fmt.Println("UpdateAggregationLacpConfig (server): (index, objName) ", i, objName)
		if attrset[i] {
			fmt.Println("UpdateAggregationLacpConfig (server): changed ", objName)
			if objName == "Enabled" {

				if conf.Enabled == true {
					lacp.SaveLaAggConfig(conf)
					EnableLaAgg(conf)
				} else {
					DisableLaAgg(conf)
					lacp.SaveLaAggConfig(conf)
				}
				return true, nil

			} else if objName == "LagType" {
				// transitioning to on or lacp
				SetLaAggType(conf)

			} else {
				switch objName {
				case "LagHash":
					SetLaAggHashMode(conf)
					break
				case "LacpMode":
					SetLaAggMode(conf)
					break
				case "Interval":
					SetLaAggPeriod(conf)
					break
				// port move cause systemId changes
				case "SystemIdMac", "SystemPriority":
					SetLaAggSystemInfo(conf)
					break
				// this may cause lag to go down if min ports is > actual ports
				case "MinLinks":
					break
				default:
					// unhandled config
				}

				lacp.SaveLaAggConfig(conf)
			}
		}
	}
	return true, nil
}

func (la LACPDServiceHandler) CreateEthernetConfig(config *lacpd.EthernetConfig) (bool, error) {

	aggModeMap := map[uint32]string{
		//LacpActivityTypeACTIVE:  "ACTIVE",
		//LacpActivityTypeSTANDBY: "STANDBY",
		lacp.LacpModeOn:      "ON",
		lacp.LacpModeActive:  "ACTIVE",
		lacp.LacpModePassive: "PASSIVE",
	}
	aggIntervalToTimeoutMap := map[time.Duration]string{
		//LacpPeriodTypeSLOW: "LONG",
		//LacpPeriodTypeFAST: "SHORT",
		lacp.LacpSlowPeriodicTime: "LONG",
		lacp.LacpFastPeriodicTime: "SHORT",
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
		fmt.Println("\nDid not find agg", config.AggregateId)
		// lets create a port with some defaults
		// origional tested thrift api
		la.CreateLaAggPort(
			lacpd.Uint16(GetIdByName(config.NameKey)),
			0, //Prio lacpdServices.Uint16,
			lacpd.Uint16(GetKeyByAggName(config.AggregateId)),
			lacpd.Int(GetIdByName(config.AggregateId)),
			config.Enabled,
			"ON",                //"ON", "ACTIVE", "PASSIVE"
			"LONG",              //"SHORT", "LONG"
			"00:00:00:00:00:00", // default
			lacpd.Int(ConvertModelSpeedToLaAggSpeed(config.Speed)),
			lacpd.Int(config.DuplexMode),
			lacpd.Int(config.Mtu),
			config.MacAddress,
			config.NameKey,
		)
	} else {
		fmt.Println("\nFound agg", config.AggregateId)
		//fmt.Printf("LAG:\n%#v\n", a)
		// Found the lag lets add the lacp info to the port
		//	    2 : i32 	Interval
		// 	    3 : i32 	LacpMode
		//	    4 : string 	SystemIdMac
		//	    5 : i16 	SystemPriority
		mode, ok := aggModeMap[uint32(a.Config.Mode)]
		if !ok || a.AggType == lacp.LaAggTypeSTATIC {
			mode = "ON"
		}

		timeout, ok := aggIntervalToTimeoutMap[a.Config.Interval]
		if !ok {
			timeout = "LONG"
		}

		// origional tested thrift api
		la.CreateLaAggPort(
			lacpd.Uint16(GetIdByName(config.NameKey)),
			lacpd.Uint16(a.Config.SystemPriority),
			lacpd.Uint16(GetKeyByAggName(config.AggregateId)),
			lacpd.Int(GetIdByName(config.AggregateId)),
			config.Enabled,
			mode,
			timeout,
			config.MacAddress,
			lacpd.Int(ConvertModelSpeedToLaAggSpeed(config.Speed)),
			lacpd.Int(config.DuplexMode),
			lacpd.Int(config.Mtu),
			a.Config.SystemIdMac,
			config.NameKey,
		)
	}
	return true, nil
}

func (la LACPDServiceHandler) DeleteEthernetConfig(config *lacpd.EthernetConfig) (bool, error) {
	return true, nil
}

func (la LACPDServiceHandler) UpdateEthernetConfig(origconfig *lacpd.EthernetConfig, updateconfig *lacpd.EthernetConfig, attrSet []bool) (bool, error) {

	return true, nil
}

func (la LACPDServiceHandler) CreateLaAggregator(aggId lacpd.Int,
	actorAdminKey lacpd.Uint16,
	actorSystemId string,
	aggMacAddr string) (lacpd.Int, error) {

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

func (la LACPDServiceHandler) DeleteLaAggregator(aggId lacpd.Int) (lacpd.Int, error) {

	lacp.DeleteLaAgg(int(aggId))
	return 0, nil
}

func (la LACPDServiceHandler) CreateLaAggPort(Id lacpd.Uint16,
	Prio lacpd.Uint16,
	Key lacpd.Uint16,
	AggId lacpd.Int,
	Enable bool,
	Mode string, //"ON", "ACTIVE", "PASSIVE"
	Timeout string, //"SHORT", "LONG"
	Mac string,
	Speed lacpd.Int,
	Duplex lacpd.Int,
	Mtu lacpd.Int,
	SysId string,
	IntfId string) (lacpd.Int, error) {

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

func (la LACPDServiceHandler) DeleteLaAggPort(Id lacpd.Uint16) (lacpd.Int, error) {
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

func DisableLaAgg(conf *lacp.LaAggConfig) error {
	var a *lacp.LaAggregator
	if lacp.LaFindAggById(conf.Id, &a) {
		fmt.Printf("Disable LaAgg %s portNumList %#v\n", a.AggName, a.PortNumList)
		// configured ports
		for _, pId := range a.PortNumList {
			lacp.DisableLaAggPort(uint16(pId))
		}
	}
	return nil
}

func EnableLaAgg(conf *lacp.LaAggConfig) error {
	var a *lacp.LaAggregator
	if lacp.LaFindAggById(conf.Id, &a) {

		fmt.Printf("Enable LaAgg %s portNumList %#v", a.AggName, a.PortNumList)
		// configured ports
		for _, pId := range a.PortNumList {
			lacp.EnableLaAggPort(uint16(pId))
		}
	}
	return nil
}

// SetLaAggType will set whether the agg is static or lacp enabled
func SetLaAggType(conf *lacp.LaAggConfig) error {
	return SetLaAggMode(conf)
}

// SetLaAggPortMode will set the lacp mode of the port based
// on the model values
func SetLaAggMode(conf *lacp.LaAggConfig) error {

	var a *lacp.LaAggregator
	var p *lacp.LaAggPort
	if lacp.LaFindAggById(conf.Id, &a) {

		if conf.Type == lacp.LaAggTypeSTATIC {
			// configured ports
			for _, pId := range a.PortNumList {
				if lacp.LaFindPortById(uint16(pId), &p) {
					lacp.SetLaAggPortLacpMode(uint16(pId), lacp.LacpModeOn)
				}
			}
		} else {
			for _, pId := range a.PortNumList {
				if lacp.LaFindPortById(uint16(pId), &p) {
					lacp.SetLaAggPortLacpMode(uint16(pId), int(conf.Lacp.Mode))
				}
			}
		}
	}
	lacp.SaveLaAggConfig(conf)

	return nil
}

func SetLaAggHashMode(conf *lacp.LaAggConfig) error {
	lacp.SetLaAggHashMode(conf.Id, conf.HashMode)
	return nil
}

func SetLaAggPeriod(conf *lacp.LaAggConfig) error {
	var a *lacp.LaAggregator
	if lacp.LaFindAggById(conf.Id, &a) {
		// configured ports
		for _, pId := range a.PortNumList {
			lacp.SetLaAggPortLacpPeriod(uint16(pId), conf.Lacp.Interval)
		}
	}
	return nil
}

func SetLaAggSystemInfo(conf *lacp.LaAggConfig) error {
	var a *lacp.LaAggregator
	if lacp.LaFindAggById(conf.Id, &a) {
		// configured ports
		for _, pId := range a.PortNumList {
			lacp.SetLaAggPortSystemInfo(uint16(pId), conf.Lacp.SystemIdMac, conf.Lacp.SystemPriority)
		}
	}
	return nil
}

// SetPortLacpLogEnable will enable on a per port basis logging
// modStr - PORT, RXM, TXM, PTXM, TXM, CDM, ALL
// modStr can be a string containing one or more of the above
func (la LACPDServiceHandler) SetPortLacpLogEnable(Id lacpd.Uint16, modStr string, ena bool) (lacpd.Int, error) {
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

// GetBulkAggregationLacpState will return the status of all the lag groups
// All lag groups are stored in a map, thus we will assume that the order
// at which a for loop iterates over the map is preserved.  It is assumed
// that at the time of this operation that no new aggragators are added,
// otherwise can get inconsistent results
func (la LACPDServiceHandler) GetBulkAggregationLacpState(fromIndex lacpd.Int, count lacpd.Int) (obj *lacpd.AggregationLacpStateGetInfo, err error) {

	var lagStateList []lacpd.AggregationLacpState = make([]lacpd.AggregationLacpState, count)
	var nextLagState *lacpd.AggregationLacpState
	var returnLagStates []*lacpd.AggregationLacpState
	var returnLagStateGetInfo lacpd.AggregationLacpStateGetInfo
	var a *lacp.LaAggregator
	validCount := lacpd.Int(0)
	toIndex := fromIndex
	obj = &returnLagStateGetInfo
	for currIndex := lacpd.Int(0); validCount != count && lacp.LaGetAggNext(&a); currIndex++ {

		if currIndex < fromIndex {
			continue
		} else {

			nextLagState = &lagStateList[validCount]
			nextLagState.Description = a.AggDescription
			nextLagState.LagType = ConvertLaAggTypeToModelLagType(a.AggType)
			nextLagState.Enabled = true
			nextLagState.MinLinks = int16(a.AggMinLinks)
			nextLagState.NameKey = a.AggName
			nextLagState.Type = "ETH"
			nextLagState.Interval = ConvertLaAggIntervalToLacpPeriod(a.Config.Interval)
			nextLagState.LacpMode = ConvertLaAggModeToModelLacpMode(a.Config.Mode)
			nextLagState.SystemIdMac = a.Config.SystemIdMac
			nextLagState.SystemPriority = int16(a.Config.SystemPriority)
			nextLagState.LagHash = int32(a.LagHash)

			if len(returnLagStates) == 0 {
				returnLagStates = make([]*lacpd.AggregationLacpState, 0)
			}
			returnLagStates = append(returnLagStates, nextLagState)
			validCount++
			toIndex++
		}
	}
	// lets try and get the next agg if one exists then there are more routes
	moreRoutes := false
	if a != nil {
		moreRoutes = lacp.LaGetAggNext(&a)
	}

	fmt.Printf("Returning %d list of lagGroups\n", validCount)
	obj.AggregationLacpStateList = returnLagStates
	obj.StartIdx = fromIndex
	obj.EndIdx = toIndex + 1
	obj.More = moreRoutes
	obj.Count = validCount

	return obj, nil
}

// GetBulkAggregationLacpMemberStateCounters will return the status of all
// the lag members.
func (la LACPDServiceHandler) GetBulkAggregationLacpMemberStateCounters(fromIndex lacpd.Int, count lacpd.Int) (obj *lacpd.AggregationLacpMemberStateCountersGetInfo, err error) {

	var lagMemberStateList []lacpd.AggregationLacpMemberStateCounters = make([]lacpd.AggregationLacpMemberStateCounters, count)
	var nextLagMemberState *lacpd.AggregationLacpMemberStateCounters
	var returnLagMemberStates []*lacpd.AggregationLacpMemberStateCounters
	var returnLagMemberStateGetInfo lacpd.AggregationLacpMemberStateCountersGetInfo
	var p *lacp.LaAggPort
	validCount := lacpd.Int(0)
	toIndex := fromIndex
	obj = &returnLagMemberStateGetInfo
	for currIndex := lacpd.Int(0); validCount != count && lacp.LaGetPortNext(&p); currIndex++ {

		if currIndex < fromIndex {
			continue
		} else {

			nextLagMemberState = &lagMemberStateList[validCount]

			// actor info
			nextLagMemberState.Aggregatable = lacp.LacpStateIsSet(p.ActorOper.State, lacp.LacpStateAggregationBit)
			nextLagMemberState.Collecting = lacp.LacpStateIsSet(p.ActorOper.State, lacp.LacpStateCollectingBit)
			nextLagMemberState.Distributing = lacp.LacpStateIsSet(p.ActorOper.State, lacp.LacpStateDistributingBit)

			if lacp.LacpStateIsSet(p.ActorOper.State, lacp.LacpStateSyncBit) {
				// in sync
				nextLagMemberState.Synchronization = 0
			} else {
				// out of sync
				nextLagMemberState.Synchronization = 1
			}
			// short 1, long 0
			if lacp.LacpStateIsSet(p.ActorOper.State, lacp.LacpStateTimeoutBit) {
				// short
				nextLagMemberState.Timeout = 1
			} else {
				// long
				nextLagMemberState.Timeout = 0
			}

			if lacp.LacpStateIsSet(p.ActorOper.State, lacp.LacpStateActivityBit) {
				// active
				nextLagMemberState.Activity = 0
			} else {
				// passive
				nextLagMemberState.Activity = 1
			}

			nextLagMemberState.OperKey = int16(p.ActorOper.Key)
			nextLagMemberState.Interface = p.IntfNum
			if p.AggAttached != nil {
				nextLagMemberState.LacpMode = ConvertLaAggModeToModelLacpMode(p.AggAttached.Config.Mode)
			}

			// partner info
			nextLagMemberState.PartnerId = p.PartnerOper.System.LacpSystemConvertSystemIdToString()
			nextLagMemberState.PartnerKey = int16(p.PartnerOper.Key)

			// System
			nextLagMemberState.Description = ""
			nextLagMemberState.SystemIdMac = p.ActorOper.System.LacpSystemConvertSystemIdToString()[6:]
			nextLagMemberState.LagType = ConvertLaAggTypeToModelLagType(p.AggAttached.AggType)
			nextLagMemberState.SystemId = p.ActorOper.System.LacpSystemConvertSystemIdToString()
			nextLagMemberState.Interval = ConvertLaAggIntervalToLacpPeriod(p.AggAttached.Config.Interval)
			nextLagMemberState.Enabled = p.PortEnabled

			nextLagMemberState.NameKey = p.IntfNum
			nextLagMemberState.SystemPriority = int16(p.ActorOper.System.Actor_System_priority)
			nextLagMemberState.Type = "ETH"
			nextLagMemberState.MinLinks = int16(p.AggAttached.AggMinLinks)

			// stats
			nextLagMemberState.LacpInPkts = int64(p.Counters.LacpInPkts)
			nextLagMemberState.LacpOutPkts = int64(p.Counters.LacpOutPkts)
			nextLagMemberState.LacpRxErrors = int64(p.Counters.LacpRxErrors)
			nextLagMemberState.LacpTxErrors = int64(p.Counters.LacpTxErrors)
			nextLagMemberState.LacpUnknownErrors = int64(p.Counters.LacpUnknownErrors)
			nextLagMemberState.LacpErrors = int64(p.Counters.LacpErrors)

			if len(returnLagMemberStates) == 0 {
				returnLagMemberStates = make([]*lacpd.AggregationLacpMemberStateCounters, 0)
			}
			returnLagMemberStates = append(returnLagMemberStates, nextLagMemberState)
			validCount++
			toIndex++
		}
	}
	// lets try and get the next agg if one exists then there are more routes
	moreRoutes := false
	if p != nil {
		moreRoutes = lacp.LaGetPortNext(&p)
	}

	fmt.Printf("Returning %d list of lagMembers\n", validCount)
	obj.AggregationLacpMemberStateCountersList = returnLagMemberStates
	obj.StartIdx = fromIndex
	obj.EndIdx = toIndex + 1
	obj.More = moreRoutes
	obj.Count = validCount

	return obj, nil
}
