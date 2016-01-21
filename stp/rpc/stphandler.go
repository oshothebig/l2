// lahandler
package rpc

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	//stp "l2/stp/protocol"
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
func (s *STPDServiceHandler) CreateDot1dStpBridgeConfig(config *stpd.Dot1dStpBridgeConfig) (bool, error) {

	fmt.Println("CreateDot1dStpBridgeConfig (server): created ")
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
	fmt.Println("CreateDot1dStpPortEntryConfig (server): created ")
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

	//var stpBridgeStateList []stpd.Dot1dStpBridgeState = make([]stpd.Dot1dStpBridgeState, count)
	//var nextStpBridgeState *stpd.Dot1dStpBridgeState
	var returnStpBridgeStates []*stpd.Dot1dStpBridgeState
	var returnStpBridgeStateGetInfo stpd.Dot1dStpBridgeStateGetInfo
	//var a *lacp.LaAggregator
	validCount := stpd.Int(0)
	toIndex := fromIndex
	obj = &returnStpBridgeStateGetInfo

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
