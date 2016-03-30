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
	//"net"
	"reflect"
	"strconv"
	"strings"
	"time"
)

const DBName string = "UsrConfDb.db"

type LACPDServiceHandler struct {
}

func NewLACPDServiceHandler() *LACPDServiceHandler {
	lacp.LacpStartTime = time.Now()
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

func ConvertSqlBooleanToBool(sqlbool string) bool {
	if sqlbool == "true" {
		return true
	} else if sqlbool == "True" {
		return true
	} else if sqlbool == "1" {
		return true
	}
	return false
}

func ConvertAdminStateStringToBool(s string) bool {
	if s == "UP" {
		return true
	} else if s == "ON" {
		return true
	} else if s == "ENABLE" {
		return true
	}
	return false
}

func ConvertRxMachineStateToYangState(state int) int32 {
	var yangstate int32
	switch state {
	case lacp.LacpRxmStateInitialize:
		yangstate = 3
		break
	case lacp.LacpRxmStatePortDisabled:
		yangstate = 5
		break
	case lacp.LacpRxmStateExpired:
		yangstate = 1
		break
	case lacp.LacpRxmStateLacpDisabled:
		yangstate = 4
		break
	case lacp.LacpRxmStateDefaulted:
		yangstate = 2
		break
	case lacp.LacpRxmStateCurrent:
		yangstate = 0
		break
	}
	return yangstate
}

func ConvertMuxMachineStateToYangState(state int) int32 {
	var yangstate int32
	switch state {
	case lacp.LacpMuxmStateDetached, lacp.LacpMuxmStateCDetached:
		yangstate = 0
		break
	case lacp.LacpMuxmStateWaiting, lacp.LacpMuxmStateCWaiting:
		yangstate = 1
		break
	case lacp.LacpMuxmStateAttached, lacp.LacpMuxmStateCAttached:
		yangstate = 2
		break
	case lacp.LacpMuxmStateCollecting:
		yangstate = 3
		break
	case lacp.LacpMuxmStateDistributing:
		yangstate = 4
		break
	case lacp.LacpMuxStateCCollectingDistributing:
		yangstate = 5
		break
	}
	return yangstate
}

func ConvertCdmMachineStateToYangState(state int) int32 {
	var yangstate int32
	switch state {
	case lacp.LacpCdmStateNoActorChurn:
		yangstate = 0
		break
	case lacp.LacpCdmStateActorChurn:
		yangstate = 1
		break
	}
	return yangstate
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

func (la *LACPDServiceHandler) HandleDbReadLaPortChannel(dbHdl *sql.DB) error {
	dbCmd := "select * from AggregationLacpConfig"
	rows, err := dbHdl.Query(dbCmd)
	if err != nil {
		fmt.Println(fmt.Sprintf("DB method Query failed for 'AggregationLacpConfig' with error AggregationLacpConfig", dbCmd, err))
		return err
	}

	defer rows.Close()

	for rows.Next() {

		object := new(lacpd.LaPortChannel)
		if err = rows.Scan(&object.LagId, &object.LagType, &object.MinLinks, &object.Interval, &object.LacpMode, &object.SystemIdMac, &object.SystemPriority, &object.LagHash, &object.AdminState, &object.Members); err != nil {

			fmt.Println("Db method Scan failed when interating over AggregationLacpConfig")
		}
		_, err = la.CreateLaPortChannel(object)
		if err != nil {
			return err
		}
	}
	return nil
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

	if err := la.HandleDbReadLaPortChannel(dbHdl); err != nil {
		fmt.Println("Error getting All AggregationLacpConfig objects")
		return err
	}

	return nil
}

// CreateLaPortChannel will create an lacp lag
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
func (la LACPDServiceHandler) CreateLaPortChannel(config *lacpd.LaPortChannel) (bool, error) {

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

	nameKey := fmt.Sprintf("agg-%d", config.LagId)

	var a *lacp.LaAggregator
	if lacp.LaFindAggByName(nameKey, &a) {

		return false, errors.New(fmt.Sprintf("LACP: Error trying to create Lag %d that already exists", config.LagId))

	} else {

		conf := &lacp.LaAggConfig{
			Id:  int(config.LagId),
			Key: GetKeyByAggName(nameKey),
			// Identifier of the lag
			Name: nameKey,
			// Type of LAG STATIC or LACP
			Type:     ConvertModelLagTypeToLaAggType(config.LagType),
			MinLinks: uint16(config.MinLinks),
			Enabled:  ConvertAdminStateStringToBool(config.AdminState),
			// lacp config
			Lacp: lacp.LacpConfigInfo{
				Interval:       ConvertModelLacpPeriodToLaAggInterval(config.Interval),
				Mode:           ConvertModelLacpModeToLaAggMode(config.LacpMode),
				SystemIdMac:    config.SystemIdMac,
				SystemPriority: uint16(config.SystemPriority),
			},
			//Properties: lacp.PortProperties{
			//	Mtu: int(config.Mtu),
			//},
			HashMode: uint32(config.LagHash),
		}
		lacp.CreateLaAgg(conf)

		var a *lacp.LaAggregator
		if lacp.LaFindAggById(conf.Id, &a) {

			for _, ifindex := range config.Members {
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
					lacpd.Uint16(ifindex),
					lacpd.Uint16(a.Config.SystemPriority),
					lacpd.Uint16(config.LagId),
					lacpd.Int(GetIdByName(nameKey)),
					conf.Enabled,
					mode,
					timeout,
					"", // taken from port
					0,  // taken from port
					0,  // taken from port
					0,  // taken from port
					a.Config.SystemIdMac,
					fmt.Sprintf("fpPort%s", ifindex), // taken from port
				)
			}
		}
	}
	return true, nil
}

func (la LACPDServiceHandler) DeleteLaPortChannel(config *lacpd.LaPortChannel) (bool, error) {

	nameKey := fmt.Sprintf("agg-%d", config.LagId)

	// Aggregation found now lets delete
	lacp.DeleteLaAgg(GetIdByName(nameKey))
	return true, nil
}

func GetAddDelMembers(orig []uint16, update []int32) (add, del []int32) {

	origMap := make(map[int32]bool, len(orig))
	// make the origional list a map
	for _, m := range orig {
		origMap[int32(m)] = false
	}

	// iterate through list, mark found entries
	// if entry is not found then added to add slice
	for _, m := range update {
		if _, ok := origMap[m]; ok {
			// found entry, don't want to just
			// remove from map in case the user supplied duplicate values
			origMap[m] = true
		} else {
			add = append(add, m)
		}
	}

	// iterate through map to find entries wither were not found in update
	for m, v := range origMap {
		if !v {
			del = append(del, m)
		}
	}
	return
}

func (la LACPDServiceHandler) UpdateLaPortChannel(origconfig *lacpd.LaPortChannel, updateconfig *lacpd.LaPortChannel, attrset []bool) (bool, error) {

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

	objTyp := reflect.TypeOf(*origconfig)
	//objVal := reflect.ValueOf(origconfig)
	//updateObjVal := reflect.ValueOf(*updateconfig)

	nameKey := fmt.Sprintf("agg-%d", updateconfig.LagId)

	conf := &lacp.LaAggConfig{
		Id:  GetIdByName(nameKey),
		Key: GetKeyByAggName(nameKey),
		// Identifier of the lag
		Name: nameKey,
		// Type of LAG STATIC or LACP
		Type:     ConvertModelLagTypeToLaAggType(updateconfig.LagType),
		MinLinks: uint16(updateconfig.MinLinks),
		Enabled:  ConvertAdminStateStringToBool(updateconfig.AdminState),
		// lacp config
		Lacp: lacp.LacpConfigInfo{
			Interval:       ConvertModelLacpPeriodToLaAggInterval(updateconfig.Interval),
			Mode:           ConvertModelLacpModeToLaAggMode(updateconfig.LacpMode),
			SystemIdMac:    updateconfig.SystemIdMac,
			SystemPriority: uint16(updateconfig.SystemPriority),
		},
		HashMode: uint32(updateconfig.LagHash),
	}

	// lets deal with Members attribute first
	for i := 0; i < objTyp.NumField(); i++ {
		objName := objTyp.Field(i).Name
		//fmt.Println("UpdateAggregationLacpConfig (server): (index, objName) ", i, objName)
		if attrset[i] {
			fmt.Println("UpdateAggregationLacpConfig (server): changed ", objName)
			if objName == "Members" {
				var a *lacp.LaAggregator

				if lacp.LaFindAggById(int(conf.Id), &a) {
					addList, delList := GetAddDelMembers(a.PortNumList, updateconfig.Members)
					for _, m := range delList {
						la.DeleteLaAggPort(lacpd.Uint16(m))
					}
					for _, ifindex := range addList {
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
							lacpd.Uint16(ifindex),
							lacpd.Uint16(updateconfig.SystemPriority),
							lacpd.Uint16(conf.Id),
							lacpd.Int(GetIdByName(nameKey)),
							conf.Enabled,
							mode,
							timeout,
							"", // taken from port
							0,  // taken from port
							0,  // taken from port
							0,  // taken from port
							updateconfig.SystemIdMac,
							fmt.Sprintf("fpPort%s", ifindex), // TODO read from port if taken from port
						)
					}
				}
			}
		}
	}

	// important to note that the attrset starts at index 0 which is the BaseObj
	// which is not the first element on the thrift obj, thus we need to skip
	// this attribute
	for i := 0; i < objTyp.NumField(); i++ {
		objName := objTyp.Field(i).Name
		//fmt.Println("UpdateAggregationLacpConfig (server): (index, objName) ", i, objName)
		if attrset[i] {
			fmt.Println("UpdateAggregationLacpConfig (server): changed ", objName)
			if objName == "AdminState" {
				if conf.Enabled {
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

	//mac, _ := net.ParseMAC(Mac)
	conf := &lacp.LaAggPortConfig{
		Id:       uint16(Id),
		Prio:     uint16(Prio),
		AggId:    int(AggId),
		Key:      uint16(Key),
		Enable:   Enable,
		Mode:     m,
		Timeout:  t,
		TraceEna: true,
		IntfId:   IntfId,
	}
	lacp.CreateLaAggPort(conf)
	return 0, nil
}

func (la LACPDServiceHandler) DeleteLaAggPort(Id lacpd.Uint16) (lacpd.Int, error) {
	var p *lacp.LaAggPort
	if lacp.LaFindPortById(uint16(Id), &p) {
		if p.AggId != 0 && p.Key != 0 {
			lacp.DeleteLaAggPortFromAgg(p.Key, uint16(Id))
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

func (la LACPDServiceHandler) GetPortChannelState(portChannel *lacpd.LaPortChannelState) (*lacpd.LaPortChannelState, error) {
	pcs := &lacpd.LaPortChannelState{}

	var a *lacp.LaAggregator
	if lacp.LaFindAggById(int(portChannel.LagId), &a) {
		pcs.LagId = int32(a.AggId)
		pcs.IfIndex = int32(a.AggId)
		pcs.LagType = ConvertLaAggTypeToModelLagType(a.AggType)
		pcs.AdminState = "DOWN"
		if a.AdminState {
			pcs.AdminState = "UP"
		}
		pcs.OperState = "DOWN"
		if a.OperState {
			pcs.OperState = "UP"
		}
		pcs.MinLinks = int16(a.AggMinLinks)
		pcs.Interval = ConvertLaAggIntervalToLacpPeriod(a.Config.Interval)
		pcs.LacpMode = ConvertLaAggModeToModelLacpMode(a.Config.Mode)
		pcs.SystemIdMac = a.Config.SystemIdMac
		pcs.SystemPriority = int16(a.Config.SystemPriority)
		pcs.LagHash = int32(a.LagHash)
		//pcs.Ifindex = int32(a.HwAggId)
		for _, m := range a.PortNumList {
			pcs.Members = append(pcs.Members, int32(m))
			var p *lacp.LaAggPort
			if lacp.LaFindPortById(m, &p) {
				if lacp.LacpStateIsSet(p.ActorOper.State, lacp.LacpStateDistributingBit) {
					pcs.MembersUpInBundle = append(pcs.MembersUpInBundle, int32(m))
				}
			}
		}
	} else {
		return pcs, errors.New(fmt.Sprintf("LACP: Unable to find port channel from LagId %d", portChannel.LagId))
	}
	return pcs, nil
}

// GetBulkLaAggrGroupState will return the status of all the lag groups
// All lag groups are stored in a map, thus we will assume that the order
// at which a for loop iterates over the map is preserved.  It is assumed
// that at the time of this operation that no new aggragators are added,
// otherwise can get inconsistent results
func (la LACPDServiceHandler) GetBulkLaPortChannelState(fromIndex lacpd.Int, count lacpd.Int) (obj *lacpd.LaPortChannelStateGetInfo, err error) {

	var lagStateList []lacpd.LaPortChannelState = make([]lacpd.LaPortChannelState, count)
	var nextLagState *lacpd.LaPortChannelState
	var returnLagStates []*lacpd.LaPortChannelState
	var returnLagStateGetInfo lacpd.LaPortChannelStateGetInfo
	var a *lacp.LaAggregator
	validCount := lacpd.Int(0)
	toIndex := fromIndex
	obj = &returnLagStateGetInfo

	for currIndex := lacpd.Int(0); validCount != count && lacp.LaGetAggNext(&a); currIndex++ {

		if currIndex < fromIndex {
			continue
		} else {

			nextLagState = &lagStateList[validCount]
			nextLagState.LagId = int32(a.AggId)
			nextLagState.IfIndex = int32(a.AggId)
			nextLagState.LagType = ConvertLaAggTypeToModelLagType(a.AggType)
			nextLagState.AdminState = "DOWN"
			if a.AdminState {
				nextLagState.AdminState = "UP"
			}
			nextLagState.OperState = "DOWN"
			if a.OperState {
				nextLagState.OperState = "UP"
			}
			nextLagState.MinLinks = int16(a.AggMinLinks)
			nextLagState.Interval = ConvertLaAggIntervalToLacpPeriod(a.Config.Interval)
			nextLagState.LacpMode = ConvertLaAggModeToModelLacpMode(a.Config.Mode)
			nextLagState.SystemIdMac = a.Config.SystemIdMac
			nextLagState.SystemPriority = int16(a.Config.SystemPriority)
			nextLagState.LagHash = int32(a.LagHash)
			//nextLagState.Ifindex = int32(a.HwAggId)
			for _, m := range a.PortNumList {
				nextLagState.Members = append(nextLagState.Members, int32(m))
				var p *lacp.LaAggPort
				if lacp.LaFindPortById(m, &p) {
					if lacp.LacpStateIsSet(p.ActorOper.State, lacp.LacpStateDistributingBit) {
						nextLagState.MembersUpInBundle = append(nextLagState.MembersUpInBundle, int32(m))
					}
				}
			}

			if len(returnLagStates) == 0 {
				returnLagStates = make([]*lacpd.LaPortChannelState, 0)
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
	obj.LaPortChannelStateList = returnLagStates
	obj.StartIdx = fromIndex
	obj.EndIdx = toIndex + 1
	obj.More = moreRoutes
	obj.Count = validCount

	return obj, nil
}

func (la LACPDServiceHandler) GetPortChannelMemberState(portchannelmemberstate *lacpd.LaPortChannelMemberState) (*lacpd.LaPortChannelMemberState, error) {
	pcms := &lacpd.LaPortChannelMemberState{}
	var p *lacp.LaAggPort
	if lacp.LaFindPortByPortId(int(portchannelmemberstate.IfIndex), &p) {
		// actor info
		pcms.Aggregatable = lacp.LacpStateIsSet(p.ActorOper.State, lacp.LacpStateAggregationBit)
		pcms.Collecting = lacp.LacpStateIsSet(p.ActorOper.State, lacp.LacpStateCollectingBit)
		pcms.Distributing = lacp.LacpStateIsSet(p.ActorOper.State, lacp.LacpStateDistributingBit)
		pcms.Defaulted = lacp.LacpStateIsSet(p.ActorOper.State, lacp.LacpStateDefaultedBit)

		if pcms.Distributing {
			pcms.OperState = "UP"
		} else {
			pcms.OperState = "DOWN"
		}

		if lacp.LacpStateIsSet(p.ActorOper.State, lacp.LacpStateSyncBit) {
			// in sync
			pcms.Synchronization = 0
		} else {
			// out of sync
			pcms.Synchronization = 1
		}
		// short 1, long 0
		if lacp.LacpStateIsSet(p.ActorOper.State, lacp.LacpStateTimeoutBit) {
			// short
			pcms.Timeout = 1
		} else {
			// long
			pcms.Timeout = 0
		}

		if lacp.LacpStateIsSet(p.ActorOper.State, lacp.LacpStateActivityBit) {
			// active
			pcms.Activity = 0
		} else {
			// passive
			pcms.Activity = 1
		}

		pcms.OperKey = int16(p.ActorOper.Key)
		pcms.IfIndex = int32(p.PortNum)
		if p.AggAttached != nil {
			pcms.LagId = int32(p.AggId)
			//		pcms.Mode = ConvertLaAggModeToModelLacpMode(p.AggAttached.Config.Mode)
		}

		// partner info
		pcms.PartnerId = p.PartnerOper.System.LacpSystemConvertSystemIdToString()
		pcms.PartnerKey = int16(p.PartnerOper.Key)

		// System
		//pcms.SystemIdMac = p.ActorOper.System.LacpSystemConvertSystemIdToString()[6:]
		//pcms.LagType = ConvertLaAggTypeToModelLagType(p.AggAttached.AggType)
		pcms.SystemId = p.ActorOper.System.LacpSystemConvertSystemIdToString()
		//pcms.Interval = ConvertLaAggIntervalToLacpPeriod(p.AggAttached.Config.Interval)
		//pcms.Enabled = p.PortEnabled

		// stats
		pcms.LacpInPkts = int64(p.LacpCounter.AggPortStatsLACPDUsRx)
		pcms.LacpOutPkts = int64(p.LacpCounter.AggPortStatsLACPDUsTx)
		pcms.LacpRxErrors = int64(p.LacpCounter.AggPortStatsIllegalRx)
		pcms.LacpTxErrors = 0
		pcms.LacpUnknownErrors = int64(p.LacpCounter.AggPortStatsUnknownRx)
		pcms.LacpErrors = int64(p.LacpCounter.AggPortStatsIllegalRx) + int64(p.LacpCounter.AggPortStatsUnknownRx)
		pcms.LampInPdu = int64(p.LacpCounter.AggPortStatsMarkerPDUsRx)
		pcms.LampInResponsePdu = int64(p.LacpCounter.AggPortStatsMarkerResponsePDUsRx)
		pcms.LampOutPdu = int64(p.LacpCounter.AggPortStatsMarkerPDUsTx)
		pcms.LampOutResponsePdu = int64(p.LacpCounter.AggPortStatsMarkerResponsePDUsTx)

		// debug
		pcms.DebugId = int32(p.AggPortDebug.AggPortDebugInformationID)
		pcms.RxMachine = ConvertRxMachineStateToYangState(p.AggPortDebug.AggPortDebugRxState)
		pcms.RxTime = int32(p.AggPortDebug.AggPortDebugLastRxTime)
		pcms.MuxMachine = ConvertMuxMachineStateToYangState(p.AggPortDebug.AggPortDebugMuxState)
		pcms.MuxReason = string(p.AggPortDebug.AggPortDebugMuxReason)
		pcms.ActorChurnMachine = ConvertCdmMachineStateToYangState(p.AggPortDebug.AggPortDebugActorChurnState)
		pcms.PartnerChurnMachine = ConvertCdmMachineStateToYangState(p.AggPortDebug.AggPortDebugPartnerChurnState)
		pcms.ActorChurnCount = int64(p.AggPortDebug.AggPortDebugActorChurnCount)
		pcms.PartnerChurnCount = int64(p.AggPortDebug.AggPortDebugPartnerChurnCount)
		pcms.ActorSyncTransitionCount = int64(p.AggPortDebug.AggPortDebugActorSyncTransitionCount)
		pcms.PartnerSyncTransitionCount = int64(p.AggPortDebug.AggPortDebugPartnerSyncTransitionCount)
		pcms.ActorChangeCount = int64(p.AggPortDebug.AggPortDebugActorChangeCount)
		pcms.PartnerChangeCount = int64(p.AggPortDebug.AggPortDebugPartnerChangeCount)
		pcms.ActorCdsChurnMachine = int32(p.AggPortDebug.AggPortDebugActorCDSChurnState)
		pcms.PartnerCdsChurnMachine = int32(p.AggPortDebug.AggPortDebugPartnerCDSChurnState)
		pcms.ActorCdsChurnCount = int64(p.AggPortDebug.AggPortDebugActorCDSChurnCount)
		pcms.PartnerCdsChurnCount = int64(p.AggPortDebug.AggPortDebugPartnerCDSChurnCount)
	} else {
		return pcms, errors.New(fmt.Sprintf("LACP: Unabled to find port by IfIndex %d", portchannelmemberstate.IfIndex))
	}
	return pcms, nil
}

// GetBulkAggregationLacpMemberStateCounters will return the status of all
// the lag members.
func (la LACPDServiceHandler) GetBulkLaPortChannelMemberState(fromIndex lacpd.Int, count lacpd.Int) (obj *lacpd.LaPortChannelMemberStateGetInfo, err error) {

	var lagMemberStateList []lacpd.LaPortChannelMemberState = make([]lacpd.LaPortChannelMemberState, count)
	var nextLagMemberState *lacpd.LaPortChannelMemberState
	var returnLagMemberStates []*lacpd.LaPortChannelMemberState
	var returnLagMemberStateGetInfo lacpd.LaPortChannelMemberStateGetInfo
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
			nextLagMemberState.Defaulted = lacp.LacpStateIsSet(p.ActorOper.State, lacp.LacpStateDefaultedBit)

			if nextLagMemberState.Distributing {
				nextLagMemberState.OperState = "UP"
			} else {
				nextLagMemberState.OperState = "DOWN"
			}

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
			nextLagMemberState.IfIndex = int32(p.PortNum)
			if p.AggAttached != nil {
				nextLagMemberState.LagId = int32(p.AggId)
				//		nextLagMemberState.Mode = ConvertLaAggModeToModelLacpMode(p.AggAttached.Config.Mode)
			}

			// partner info
			nextLagMemberState.PartnerId = p.PartnerOper.System.LacpSystemConvertSystemIdToString()
			nextLagMemberState.PartnerKey = int16(p.PartnerOper.Key)

			// System
			//nextLagMemberState.SystemIdMac = p.ActorOper.System.LacpSystemConvertSystemIdToString()[6:]
			//nextLagMemberState.LagType = ConvertLaAggTypeToModelLagType(p.AggAttached.AggType)
			nextLagMemberState.SystemId = p.ActorOper.System.LacpSystemConvertSystemIdToString()
			//nextLagMemberState.Interval = ConvertLaAggIntervalToLacpPeriod(p.AggAttached.Config.Interval)
			//nextLagMemberState.Enabled = p.PortEnabled

			// stats
			nextLagMemberState.LacpInPkts = int64(p.LacpCounter.AggPortStatsLACPDUsRx)
			nextLagMemberState.LacpOutPkts = int64(p.LacpCounter.AggPortStatsLACPDUsTx)
			nextLagMemberState.LacpRxErrors = int64(p.LacpCounter.AggPortStatsIllegalRx)
			nextLagMemberState.LacpTxErrors = 0
			nextLagMemberState.LacpUnknownErrors = int64(p.LacpCounter.AggPortStatsUnknownRx)
			nextLagMemberState.LacpErrors = int64(p.LacpCounter.AggPortStatsIllegalRx) + int64(p.LacpCounter.AggPortStatsUnknownRx)
			nextLagMemberState.LampInPdu = int64(p.LacpCounter.AggPortStatsMarkerPDUsRx)
			nextLagMemberState.LampInResponsePdu = int64(p.LacpCounter.AggPortStatsMarkerResponsePDUsRx)
			nextLagMemberState.LampOutPdu = int64(p.LacpCounter.AggPortStatsMarkerPDUsTx)
			nextLagMemberState.LampOutResponsePdu = int64(p.LacpCounter.AggPortStatsMarkerResponsePDUsTx)

			// debug
			nextLagMemberState.DebugId = int32(p.AggPortDebug.AggPortDebugInformationID)
			nextLagMemberState.RxMachine = ConvertRxMachineStateToYangState(p.AggPortDebug.AggPortDebugRxState)
			nextLagMemberState.RxTime = int32(p.AggPortDebug.AggPortDebugLastRxTime)
			nextLagMemberState.MuxMachine = ConvertMuxMachineStateToYangState(p.AggPortDebug.AggPortDebugMuxState)
			nextLagMemberState.MuxReason = string(p.AggPortDebug.AggPortDebugMuxReason)
			nextLagMemberState.ActorChurnMachine = ConvertCdmMachineStateToYangState(p.AggPortDebug.AggPortDebugActorChurnState)
			nextLagMemberState.PartnerChurnMachine = ConvertCdmMachineStateToYangState(p.AggPortDebug.AggPortDebugPartnerChurnState)
			nextLagMemberState.ActorChurnCount = int64(p.AggPortDebug.AggPortDebugActorChurnCount)
			nextLagMemberState.PartnerChurnCount = int64(p.AggPortDebug.AggPortDebugPartnerChurnCount)
			nextLagMemberState.ActorSyncTransitionCount = int64(p.AggPortDebug.AggPortDebugActorSyncTransitionCount)
			nextLagMemberState.PartnerSyncTransitionCount = int64(p.AggPortDebug.AggPortDebugPartnerSyncTransitionCount)
			nextLagMemberState.ActorChangeCount = int64(p.AggPortDebug.AggPortDebugActorChangeCount)
			nextLagMemberState.PartnerChangeCount = int64(p.AggPortDebug.AggPortDebugPartnerChangeCount)
			nextLagMemberState.ActorCdsChurnMachine = int32(p.AggPortDebug.AggPortDebugActorCDSChurnState)
			nextLagMemberState.PartnerCdsChurnMachine = int32(p.AggPortDebug.AggPortDebugPartnerCDSChurnState)
			nextLagMemberState.ActorCdsChurnCount = int64(p.AggPortDebug.AggPortDebugActorCDSChurnCount)
			nextLagMemberState.PartnerCdsChurnCount = int64(p.AggPortDebug.AggPortDebugPartnerCDSChurnCount)

			if len(returnLagMemberStates) == 0 {
				returnLagMemberStates = make([]*lacpd.LaPortChannelMemberState, 0)
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
	obj.LaPortChannelMemberStateList = returnLagMemberStates
	obj.StartIdx = fromIndex
	obj.EndIdx = toIndex + 1
	obj.More = moreRoutes
	obj.Count = validCount

	return obj, nil
}
