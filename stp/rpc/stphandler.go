// lahandler
package rpc

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	stp "l2/stp/protocol"
	"reflect"
	"stpd"
	//"time"
)

const DBName string = "UsrConfDb.db"

type STPDServiceHandler struct {
}

func NewSTPDServiceHandler() *STPDServiceHandler {
	//lacp.LacpStartTime = time.Now()
	// link up/down events for now
	startEvtHandler()
	return &STPDServiceHandler{}
}

//
func ConvertThriftBrgConfigToStpBrgConfig(config *stpd.Dot1dStpBridgeConfig, brgconfig *stp.StpBridgeConfig) {

	brgconfig.Dot1dBridgeAddress = config.Dot1dBridgeAddressKey
	brgconfig.Dot1dStpPriority = uint16(config.Dot1dStpPriorityKey)
	brgconfig.Dot1dStpBridgeMaxAge = uint16(config.Dot1dStpBridgeMaxAge)
	brgconfig.Dot1dStpBridgeHelloTime = uint16(config.Dot1dStpBridgeHelloTime)
	brgconfig.Dot1dStpBridgeForwardDelay = uint16(config.Dot1dStpBridgeForwardDelay)
	brgconfig.Dot1dStpBridgeForceVersion = int32(config.Dot1dStpBridgeForceVersion)
	brgconfig.Dot1dStpBridgeTxHoldCount = int32(config.Dot1dStpBridgeTxHoldCount)
}

func ConvertInt32ToBool(val int32) bool {
	if val == 0 {
		return false
	}
	return true
}

func ConvertThriftPortConfigToStpPortConfig(config *stpd.Dot1dStpPortEntryConfig, portconfig *stp.StpPortConfig) {

	portconfig.Dot1dStpPort = int32(config.Dot1dStpPortKey)
	portconfig.Dot1dStpPortPriority = uint16(config.Dot1dStpPortPriority)
	portconfig.Dot1dStpPortEnable = ConvertInt32ToBool(config.Dot1dStpPortEnable)
	portconfig.Dot1dStpPortPathCost = int32(config.Dot1dStpPortPathCost)
	portconfig.Dot1dStpPortProtocolMigration = int32(config.Dot1dStpPortProtocolMigration)
	portconfig.Dot1dStpPortAdminPointToPoint = int32(config.Dot1dStpPortAdminPointToPoint)
	portconfig.Dot1dStpPortAdminEdgePort = int32(config.Dot1dStpPortAdminEdgePort)
	portconfig.Dot1dStpPortAdminPathCost = int32(config.Dot1dStpPortAdminPathCost)
}

func ConvertBridgeIdToString(bridgeid stp.BridgeId) string {

	return fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x:%02x:%02x:",
		bridgeid[0],
		bridgeid[1],
		bridgeid[2],
		bridgeid[3],
		bridgeid[4],
		bridgeid[5],
		bridgeid[6],
		bridgeid[7])
}

// CreateDot1dStpBridgeConfig
func (s *STPDServiceHandler) CreateDot1dStpBridgeConfig(config *stpd.Dot1dStpBridgeConfig) (bool, error) {

	brgconfig := &stp.StpBridgeConfig{}
	fmt.Println("CreateDot1dStpBridgeConfig (server): created ")
	fmt.Println("addr:", config.Dot1dBridgeAddressKey)
	// parent restricted-int32
	fmt.Println("prio:", config.Dot1dStpPriorityKey) // int32 `SNAPROUTE: KEY`
	//yang_name: dot1dStpBridgeMaxAge class: restricted-int32
	fmt.Println("age:", config.Dot1dStpBridgeMaxAge)
	//yang_name: dot1dStpBridgeHelloTime class: restricted-int32
	fmt.Println("hello:", config.Dot1dStpBridgeHelloTime) // int32
	//yang_name: dot1dStpBridgeForwardDelay class: restricted-int32
	fmt.Println("fwddelay:", config.Dot1dStpBridgeForwardDelay) // int32
	//yang_name: dot1dStpBridgeForceVersion class: leaf
	fmt.Println("version:", config.Dot1dStpBridgeForceVersion) // int32
	//yang_name: dot1dStpBridgeTxHoldCount class: leaf
	fmt.Println("txHoldCount", config.Dot1dStpBridgeTxHoldCount) //

	ConvertThriftBrgConfigToStpBrgConfig(config, brgconfig)

	stp.StpBridgeCreate(brgconfig)
	return true, nil
}

func (s *STPDServiceHandler) HandleDbReadDot1dStpBridgeConfig(dbHdl *sql.DB) error {
	dbCmd := "select * from Dot1dStpBridgeConfig"
	rows, err := dbHdl.Query(dbCmd)
	if err != nil {
		fmt.Println(fmt.Sprintf("DB method Query failed for 'Dot1dStpBridgeConfig' with error", dbCmd, err))
		return err
	}

	defer rows.Close()

	for rows.Next() {

		object := new(stpd.Dot1dStpBridgeConfig)
		if err = rows.Scan(&object.Dot1dBridgeAddressKey, &object.Dot1dStpPriorityKey, &object.Dot1dStpBridgeMaxAge, &object.Dot1dStpBridgeHelloTime, &object.Dot1dStpBridgeForwardDelay, &object.Dot1dStpBridgeForceVersion, &object.Dot1dStpBridgeTxHoldCount); err != nil {
			fmt.Println("Db method Scan failed when interating over Dot1dStpBridgeConfig")
		}
		// TODO: do something
		//_, err = la.CreateAggregationLacpConfig(object)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *STPDServiceHandler) HandleDbReadDot1dStpPortEntryConfig(dbHdl *sql.DB) error {
	dbCmd := "select * from Dot1dStpPortEntryConfig"
	rows, err := dbHdl.Query(dbCmd)
	if err != nil {
		fmt.Println(fmt.Sprintf("DB method Query failed for 'Dot1dStpBridgeConfig' with error", dbCmd, err))
		return err
	}

	defer rows.Close()

	for rows.Next() {

		object := new(stpd.Dot1dStpPortEntryConfig)
		if err = rows.Scan(&object.Dot1dStpPortKey, &object.Dot1dStpPortPriority, &object.Dot1dStpPortEnable, &object.Dot1dStpPortPathCost, &object.Dot1dStpPortPathCost32, &object.Dot1dStpPortProtocolMigration, &object.Dot1dStpPortAdminPointToPoint, &object.Dot1dStpPortAdminEdgePort, &object.Dot1dStpPortAdminPathCost); err != nil {
			fmt.Println("Db method Scan failed when interating over Dot1dStpPortEntryConfig")
		}
		// TODO: do something
		//_, err = la.CreateAggregationLacpConfig(object)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *STPDServiceHandler) ReadConfigFromDB(filePath string) error {
	var dbPath string = filePath + DBName

	dbHdl, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		//h.logger.Err(fmt.Sprintf("Failed to open the DB at %s with error %s", dbPath, err))
		fmt.Printf("Failed to open the DB at %s with error %s", dbPath, err)
		return err
	}

	defer dbHdl.Close()

	if err := s.HandleDbReadDot1dStpBridgeConfig(dbHdl); err != nil {
		fmt.Println("Error getting All Dot1dStpBridgeConfig objects")
		return err
	}

	if err = s.HandleDbReadDot1dStpPortEntryConfig(dbHdl); err != nil {
		fmt.Println("Error getting All Dot1dStpPortEntryConfig objects")
		return err
	}

	return nil
}

func (s *STPDServiceHandler) DeleteDot1dStpBridgeConfig(config *stpd.Dot1dStpBridgeConfig) (bool, error) {

	// Aggregation found now lets delete
	//lacp.DeleteLaAgg(GetIdByName(config.NameKey))
	fmt.Println("DeleteDot1dStpBridgeConfig (server): deleted ")
	return true, nil
}

func (s *STPDServiceHandler) UpdateDot1dStpBridgeConfig(origconfig *stpd.Dot1dStpBridgeConfig, updateconfig *stpd.Dot1dStpBridgeConfig, attrset []bool) (bool, error) {
	objTyp := reflect.TypeOf(*origconfig)
	//objVal := reflect.ValueOf(origconfig)
	//updateObjVal := reflect.ValueOf(*updateconfig)

	// important to note that the attrset starts at index 0 which is the BaseObj
	// which is not the first element on the thrift obj, thus we need to skip
	// this attribute
	for i := 0; i < objTyp.NumField(); i++ {
		objName := objTyp.Field(i).Name
		//fmt.Println("UpdateDot1dStpBridgeConfig (server): (index, objName) ", i, objName)
		if attrset[i] {
			fmt.Println("UpdateDot1dStpBridgeConfig (server): changed ", objName)
		}
	}
	return true, nil
}

func (s *STPDServiceHandler) CreateDot1dStpPortEntryConfig(config *stpd.Dot1dStpPortEntryConfig) (bool, error) {
	portconfig := &stp.StpPortConfig{}
	fmt.Println("CreateDot1dStpPortEntryConfig (server): created ")

	ConvertThriftPortConfigToStpPortConfig(config, portconfig)

	stp.StpPortCreate(portconfig)
	return true, nil
}

func (s *STPDServiceHandler) DeleteDot1dStpPortEntryConfig(config *stpd.Dot1dStpPortEntryConfig) (bool, error) {
	fmt.Println("DeleteDot1dStpPortEntryConfig (server): deleted ")
	return true, nil
}

func (s *STPDServiceHandler) UpdateDot1dStpPortEntryConfig(origconfig *stpd.Dot1dStpPortEntryConfig, updateconfig *stpd.Dot1dStpPortEntryConfig, attrset []bool) (bool, error) {
	objTyp := reflect.TypeOf(*origconfig)
	//objVal := reflect.ValueOf(origconfig)
	//updateObjVal := reflect.ValueOf(*updateconfig)

	// important to note that the attrset starts at index 0 which is the BaseObj
	// which is not the first element on the thrift obj, thus we need to skip
	// this attribute
	for i := 0; i < objTyp.NumField(); i++ {
		objName := objTyp.Field(i).Name
		//fmt.Println("UpdateDot1dStpBridgeConfig (server): (index, objName) ", i, objName)
		if attrset[i] {
			fmt.Println("UpdateDot1dStpBridgeConfig (server): changed ", objName)
		}
	}

	return true, nil
}

// GetBulkAggregationLacpState will return the status of all the lag groups
// All lag groups are stored in a map, thus we will assume that the order
// at which a for loop iterates over the map is preserved.  It is assumed
// that at the time of this operation that no new aggragators are added,
// otherwise can get inconsistent results
func (s *STPDServiceHandler) GetBulkDot1dStpBridgeState(fromIndex stpd.Int, count stpd.Int) (obj *stpd.Dot1dStpBridgeStateGetInfo, err error) {

	var stpBridgeStateList []stpd.Dot1dStpBridgeState = make([]stpd.Dot1dStpBridgeState, count)
	var nextStpBridgeState *stpd.Dot1dStpBridgeState
	var returnStpBridgeStates []*stpd.Dot1dStpBridgeState
	var returnStpBridgeStateGetInfo stpd.Dot1dStpBridgeStateGetInfo
	var b *stp.Bridge
	validCount := stpd.Int(0)
	toIndex := fromIndex
	obj = &returnStpBridgeStateGetInfo
	for currIndex := fromIndex; validCount != count; currIndex++ {

		if currIndex < len(stp.BridgeListTable) {
			b = stp.BridgeListTable[currIndex]
			nextStpBridgeState = &lagStateList[validCount]

			nextStpBridgeState.Dot1dStpBridgeForceVersion = b.ForceVersion
			nextStpBridgeState.Dot1dBridgeAddressKey = ConvertBridgeIdToString(b.BridgeIdentifier)
			nextStpBridgeState.Dot1dStpBridgeHelloTime = b.BridgeTimes.HelloTime
			nextStpBridgeState.Dot1dStpBridgeTxHoldCount = stp.TransmitHoldCountDefault
			nextStpBridgeState.Dot1dStpBridgeForwardDelay = b.BridgeTimes.ForwardingDelay
			nextStpBridgeState.Dot1dStpBridgeMaxAge = b.BridgeTimes.MaxAge
			nextStpBridgeState.Dot1dStpPriorityKey = b.BridgePriority
			nextStpBridgeState.Dot1dBrgIfIndex = b.BrgIfIndex
			nextStpBridgeState.Dot1dStpProtocolSpecification = 2
			//nextStpBridgeState.Dot1dStpTimeSinceTopologyChange uint32 //The time (in hundredths of a second) since the last time a topology change was detected by the bridge entity. For RSTP, this reports the time since the tcWhile timer for any port on this Bridge was nonzero.
			//nextStpBridgeState.Dot1dStpTopChanges              uint32 //The total number of topology changes detected by this bridge since the management entity was last reset or initialized.
			nextStpBridgeState.Dot1dStpDesignatedRoot = b.BridgePriority.DesignatedPortId
			nextStpBridgeState.Dot1dStpRootCost = b.BridgePriority.RootPathCost
			nextStpBridgeState.Dot1dStpRootPort = b.BridgePriority.DesignatedPortId
			//nextStpBridgeState.Dot1dStpMaxAge                  int32  //The maximum age of Spanning Tree Protocol information learned from the network on any port before it is discarded, in units of hundredths of a second.  This is the actual value that this bridge is currently using.
			nextStpBridgeState.Dot1dStpHelloTime = b.RootTimes.HelloTime
			//nextStpBridgeState.Dot1dStpHoldTime = b.RootTimes.              int32  //This time value determines the interval length during which no more than two Configuration bridge PDUs shall be transmitted by this node, in units of hundredths of a second.
			nextStpBridgeState.Dot1dStpForwardDelay = b.RootTimes.ForwardingDelay
			nextStpBridgeState.Dot1dStpVlan = b.Vlan

			if len(returnStpBridgeStates) == 0 {
				returnStpBridgeStates = make([]*stp.Dot1dStpBridgeState, 0)
			}
			returnLagStates = append(returnStpBridgeStates, nextStpBridgeState)
			validCount++
			toIndex++
		}
	}
	// lets try and get the next agg if one exists then there are more routes
	moreRoutes := false
	if a != nil {
		moreRoutes = lacp.LaGetAggNext(&a)
	}

	// lets try and get the next agg if one exists then there are more routes
	moreRoutes := false
	obj.Dot1dStpBridgeStateList = returnStpBridgeStates
	obj.StartIdx = fromIndex
	obj.EndIdx = toIndex + 1
	obj.More = moreRoutes
	obj.Count = validCount

	return obj, nil
}

// GetBulkAggregationLacpMemberStateCounters will return the status of all
// the lag members.
func (s *STPDServiceHandler) GetBulkDot1dStpPortEntryState(fromIndex stpd.Int, count stpd.Int) (obj *stpd.Dot1dStpPortEntryStateGetInfo, err error) {

	//var stpPortStateList []stpd.Dot1dStpPortEntryState = make([]stpd.Dot1dStpPortEntryState, count)
	//var nextStpPortState *stpd.Dot1dStpPortEntryState
	var returnStpPortStates []*stpd.Dot1dStpPortEntryState
	var returnStpPortStateGetInfo stpd.Dot1dStpPortEntryStateGetInfo
	//var a *lacp.LaAggregator
	validCount := stpd.Int(0)
	toIndex := fromIndex
	obj = &returnStpPortStateGetInfo

	// lets try and get the next agg if one exists then there are more routes
	moreRoutes := false
	obj.Dot1dStpPortEntryStateList = returnStpPortStates
	obj.StartIdx = fromIndex
	obj.EndIdx = toIndex + 1
	obj.More = moreRoutes
	obj.Count = validCount

	return obj, nil
}
