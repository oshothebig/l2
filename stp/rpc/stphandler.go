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

	brgconfig.Dot1dBridgeAddress = config.Dot1dBridgeAddress
	brgconfig.Dot1dStpPriority = uint16(config.Dot1dStpPriority)
	brgconfig.Dot1dStpBridgeVlan = uint16(config.Dot1dStpVlan)
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

func ConvertBoolToInt32(val bool) int32 {
	if val {
		return 1
	}
	return 0
}

func ConvertThriftPortConfigToStpPortConfig(config *stpd.Dot1dStpPortEntryConfig, portconfig *stp.StpPortConfig) {

	portconfig.Dot1dStpPort = int32(config.Dot1dStpPort)
	portconfig.Dot1dStpBridgeIfIndex = int32(config.Dot1dBrgIfIndex)
	portconfig.Dot1dStpPortPriority = uint16(config.Dot1dStpPortPriority)
	portconfig.Dot1dStpPortEnable = ConvertInt32ToBool(config.Dot1dStpPortEnable)
	portconfig.Dot1dStpPortPathCost = int32(config.Dot1dStpPortPathCost)
	portconfig.Dot1dStpPortProtocolMigration = int32(config.Dot1dStpPortProtocolMigration)
	portconfig.Dot1dStpPortAdminPointToPoint = int32(config.Dot1dStpPortAdminPointToPoint)
	portconfig.Dot1dStpPortAdminEdgePort = ConvertInt32ToBool(config.Dot1dStpPortAdminEdgePort)
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

//NOTE—The current IETF Bridge MIB (IETF RFC 1493) uses disabled, blocking, listening, learning, forwarding, and
//broken dot1dStpPortStates. The learning and forwarding states correspond exactly to the Learning and Forwarding Port
//States specified in this standard. Disabled, blocking, listening, and broken all correspond to the Discarding Port State —
//while those dot1dStpPortStates serve to distinguish reasons for discarding frames the operation of the Forwarding and
//Learning processes is the same for all of them. The dot1dStpPortState broken represents the failure or unavailability of
//the port’s MAC as indicated by MAC_Operational FALSE; disabled represents exclusion of the port from the active
//topology by management setting of the Administrative Port State to Disabled; blocking represents exclusion of the port
//from the active topology by the spanning tree algorithm [computing an Alternate or Backup Port Role (17.7)]; listening
//represents a port that the spanning tree algorithm has selected to be part of the active topology (computing a Root Port or
//Designated Port role) but is temporarily discarding frames to guard against loops or incorrect learning.
func GetPortState(p *stp.StpPort) (state int32) {
	/* defined by model
	type enumeration {
	          enum disabled   { value 1; }
	          enum blocking   { value 2; }
	          enum listening  { value 3; }
	          enum learning   { value 4; }
	          enum forwarding { value 5; }
	          enum broken     { value 6; }
	        }
	*/
	state = 0
	stp.StpLogger("INFO", fmt.Sprintf("PortEnabled[%t] Learning[%t] Forwarding[%t]", p.PortEnabled, p.Learning, p.Forwarding))
	if !p.PortEnabled {
		state = 1
	} else if p.Forwarding {
		state = 5
	} else if p.Learning {
		state = 4
	} else if p.PortEnabled &&
		!p.Learning &&
		!p.Forwarding {
		state = 2
	}
	// TODO need to determine how to set listening and broken states
	return state
}

// CreateDot1dStpBridgeConfig
func (s *STPDServiceHandler) CreateDot1dStpBridgeConfig(config *stpd.Dot1dStpBridgeConfig) (bool, error) {

	stp.StpLogger("INFO", "CreateDot1dStpBridgeConfig (server): created ")
	stp.StpLogger("INFO", fmt.Sprintf("addr:", config.Dot1dBridgeAddress))
	stp.StpLogger("INFO", fmt.Sprintf("prio:", config.Dot1dStpPriority))
	stp.StpLogger("INFO", fmt.Sprintf("vlan:", config.Dot1dStpVlan))
	stp.StpLogger("INFO", fmt.Sprintf("age:", config.Dot1dStpBridgeMaxAge))
	stp.StpLogger("INFO", fmt.Sprintf("hello:", config.Dot1dStpBridgeHelloTime))        // int32
	stp.StpLogger("INFO", fmt.Sprintf("fwddelay:", config.Dot1dStpBridgeForwardDelay))  // int32
	stp.StpLogger("INFO", fmt.Sprintf("version:", config.Dot1dStpBridgeForceVersion))   // int32
	stp.StpLogger("INFO", fmt.Sprintf("txHoldCount", config.Dot1dStpBridgeTxHoldCount)) //

	brgconfig := &stp.StpBridgeConfig{}
	ConvertThriftBrgConfigToStpBrgConfig(config, brgconfig)

	stp.StpBridgeCreate(brgconfig)
	return true, nil
}

func (s *STPDServiceHandler) HandleDbReadDot1dStpBridgeConfig(dbHdl *sql.DB) error {
	dbCmd := "select * from Dot1dStpBridgeConfig"
	rows, err := dbHdl.Query(dbCmd)
	if err != nil {
		stp.StpLogger("ERROR", fmt.Sprintf("DB method Query failed for 'Dot1dStpBridgeConfig' with error", dbCmd, err))
		return err
	}

	defer rows.Close()

	for rows.Next() {

		object := new(stpd.Dot1dStpBridgeConfig)
		if err = rows.Scan(&object.Dot1dBridgeAddress,
			&object.Dot1dStpVlan,
			&object.Dot1dStpPriority,
			&object.Dot1dStpBridgeMaxAge,
			&object.Dot1dStpBridgeHelloTime,
			&object.Dot1dStpBridgeForwardDelay,
			&object.Dot1dStpBridgeForceVersion,
			&object.Dot1dStpBridgeTxHoldCount); err != nil {
			stp.StpLogger("ERROR", "Db method Scan failed when interating over Dot1dStpBridgeConfig")
			return err
		}
		_, err = s.CreateDot1dStpBridgeConfig(object)
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
		stp.StpLogger("ERROR", fmt.Sprintf("DB method Query failed for 'Dot1dStpBridgeConfig' with error", dbCmd, err))
		return err
	}

	defer rows.Close()

	for rows.Next() {

		object := new(stpd.Dot1dStpPortEntryConfig)
		if err = rows.Scan(&object.Dot1dStpPort,
			&object.Dot1dBrgIfIndex,
			&object.Dot1dStpPortPriority,
			&object.Dot1dStpPortEnable,
			&object.Dot1dStpPortPathCost,
			&object.Dot1dStpPortPathCost32,
			&object.Dot1dStpPortProtocolMigration,
			&object.Dot1dStpPortAdminPointToPoint,
			&object.Dot1dStpPortAdminEdgePort,
			&object.Dot1dStpPortAdminPathCost); err != nil {
			stp.StpLogger("ERROR", "Db method Scan failed when interating over Dot1dStpPortEntryConfig")
			return err
		}
		_, err = s.CreateDot1dStpPortEntryConfig(object)
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
		stp.StpLogger("ERROR", fmt.Sprintf("Failed to open the DB at %s with error %s", dbPath, err))
		return err
	}

	defer dbHdl.Close()

	if err := s.HandleDbReadDot1dStpBridgeConfig(dbHdl); err != nil {
		stp.StpLogger("ERROR", "Error getting All Dot1dStpBridgeConfig objects")
		return err
	}

	if err = s.HandleDbReadDot1dStpPortEntryConfig(dbHdl); err != nil {
		stp.StpLogger("ERROR", "Error getting All Dot1dStpPortEntryConfig objects")
		return err
	}

	return nil
}

func (s *STPDServiceHandler) DeleteDot1dStpBridgeConfig(config *stpd.Dot1dStpBridgeConfig) (bool, error) {

	// Aggregation found now lets delete
	//lacp.DeleteLaAgg(GetIdByName(config.NameKey))
	stp.StpLogger("INFO", "DeleteDot1dStpBridgeConfig (server): deleted ")
	brgconfig := &stp.StpBridgeConfig{}
	ConvertThriftBrgConfigToStpBrgConfig(config, brgconfig)
	stp.StpBridgeDelete(brgconfig)
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
			stp.StpLogger("INFO", fmt.Sprintf("UpdateDot1dStpBridgeConfig (server): changed ", objName))
		}
	}
	return true, nil
}

func (s *STPDServiceHandler) CreateDot1dStpPortEntryConfig(config *stpd.Dot1dStpPortEntryConfig) (bool, error) {
	stp.StpLogger("INFO", "CreateDot1dStpPortEntryConfig (server): created ")
	portconfig := &stp.StpPortConfig{}
	ConvertThriftPortConfigToStpPortConfig(config, portconfig)

	stp.StpPortCreate(portconfig)
	return true, nil
}

func (s *STPDServiceHandler) DeleteDot1dStpPortEntryConfig(config *stpd.Dot1dStpPortEntryConfig) (bool, error) {
	stp.StpLogger("INFO", "DeleteDot1dStpPortEntryConfig (server): deleted ")
	portconfig := &stp.StpPortConfig{}
	ConvertThriftPortConfigToStpPortConfig(config, portconfig)

	stp.StpPortDelete(portconfig)
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
			stp.StpLogger("INFO", fmt.Sprintf("UpdateDot1dStpBridgeConfig (server): changed ", objName))
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
	brgListLen := stpd.Int(len(stp.BridgeListTable))
	for currIndex := fromIndex; validCount != count && currIndex < brgListLen; currIndex++ {

		b = stp.BridgeListTable[currIndex]
		nextStpBridgeState = &stpBridgeStateList[validCount]

		nextStpBridgeState.Dot1dStpBridgeForceVersion = b.ForceVersion
		nextStpBridgeState.Dot1dBridgeAddress = ConvertBridgeIdToString(b.BridgeIdentifier)
		nextStpBridgeState.Dot1dStpBridgeHelloTime = int32(b.BridgeTimes.HelloTime)
		nextStpBridgeState.Dot1dStpBridgeTxHoldCount = stp.TransmitHoldCountDefault
		nextStpBridgeState.Dot1dStpBridgeForwardDelay = int32(b.BridgeTimes.ForwardingDelay)
		nextStpBridgeState.Dot1dStpBridgeMaxAge = int32(b.BridgeTimes.MaxAge)
		nextStpBridgeState.Dot1dStpPriority = int32(stp.GetBridgePriorityFromBridgeId(b.BridgeIdentifier))
		nextStpBridgeState.Dot1dBrgIfIndex = b.BrgIfIndex
		nextStpBridgeState.Dot1dStpProtocolSpecification = 2
		//nextStpBridgeState.Dot1dStpTimeSinceTopologyChange uint32 //The time (in hundredths of a second) since the last time a topology change was detected by the bridge entity. For RSTP, this reports the time since the tcWhile timer for any port on this Bridge was nonzero.
		//nextStpBridgeState.Dot1dStpTopChanges              uint32 //The total number of topology changes detected by this bridge since the management entity was last reset or initialized.
		nextStpBridgeState.Dot1dStpDesignatedRoot = ConvertBridgeIdToString(b.BridgePriority.RootBridgeId)
		nextStpBridgeState.Dot1dStpRootCost = int32(b.BridgePriority.RootPathCost)
		nextStpBridgeState.Dot1dStpRootPort = int32(b.BridgePriority.DesignatedPortId)
		nextStpBridgeState.Dot1dStpMaxAge = int32(b.RootTimes.MaxAge)
		nextStpBridgeState.Dot1dStpHelloTime = int32(b.RootTimes.HelloTime)
		nextStpBridgeState.Dot1dStpHoldTime = int32(b.TxHoldCount)
		nextStpBridgeState.Dot1dStpForwardDelay = int32(b.RootTimes.ForwardingDelay)
		nextStpBridgeState.Dot1dStpVlan = int16(b.Vlan)

		if len(returnStpBridgeStates) == 0 {
			returnStpBridgeStates = make([]*stpd.Dot1dStpBridgeState, 0)
		}
		returnStpBridgeStates = append(returnStpBridgeStates, nextStpBridgeState)
		validCount++
		toIndex++
	}
	// lets try and get the next agg if one exists then there are more routes
	moreRoutes := false
	if fromIndex+count < brgListLen {
		moreRoutes = true
	}

	// lets try and get the next agg if one exists then there are more routes
	obj.Dot1dStpBridgeStateList = returnStpBridgeStates
	obj.StartIdx = fromIndex
	obj.EndIdx = toIndex + 1
	obj.More = moreRoutes
	obj.Count = validCount

	return obj, nil
}

// GetBulkAggregationLacpMemberStateCounters will return the status of all
// the lag members.
func (s *STPDServiceHandler) GetBulkDot1dStpPortEntryStateCountersFsmStates(fromIndex stpd.Int, count stpd.Int) (obj *stpd.Dot1dStpPortEntryStateCountersFsmStatesGetInfo, err error) {

	var stpPortStateList []stpd.Dot1dStpPortEntryStateCountersFsmStates = make([]stpd.Dot1dStpPortEntryStateCountersFsmStates, count)
	var nextStpPortState *stpd.Dot1dStpPortEntryStateCountersFsmStates
	var returnStpPortStates []*stpd.Dot1dStpPortEntryStateCountersFsmStates
	var returnStpPortStateGetInfo stpd.Dot1dStpPortEntryStateCountersFsmStatesGetInfo
	//var a *lacp.LaAggregator
	validCount := stpd.Int(0)
	toIndex := fromIndex
	obj = &returnStpPortStateGetInfo
	stpPortListLen := stpd.Int(len(stp.PortListTable))
	stp.StpLogger("INFO", fmt.Sprintf("Total configured ports %d fromIndex %d count %d", stpPortListLen, fromIndex, count))
	for currIndex := fromIndex; validCount != count && currIndex < stpPortListLen; currIndex++ {

		stp.StpLogger("INFO", fmt.Sprintf("CurrIndex %d stpPortListLen %d", currIndex, stpPortListLen))

		p := stp.PortListTable[currIndex]
		nextStpPortState = &stpPortStateList[validCount]

		nextStpPortState.Dot1dStpPortOperPointToPoint = ConvertBoolToInt32(p.OperPointToPointMAC)
		nextStpPortState.Dot1dBrgIfIndex = p.BrgIfIndex
		nextStpPortState.Dot1dStpPortOperEdgePort = ConvertBoolToInt32(p.OperEdge)
		nextStpPortState.Dot1dStpPortDesignatedPort = fmt.Sprintf("%d", p.PortPriority.DesignatedPortId)
		nextStpPortState.Dot1dStpPortAdminEdgePort = ConvertBoolToInt32(p.AdminEdge)
		//nextStpPortState.Dot1dStpPortForwardTransitions uint32 //The number of times this port has transitioned from the Learning state to the Forwarding state.
		//nextStpPortState.Dot1dStpPortProtocolMigration  int32  //When operating in RSTP (version 2) mode, writing true(1) to this object forces this port to transmit RSTP BPDUs. Any other operation on this object has no effect and it always returns false(2) when read.
		nextStpPortState.Dot1dStpPort = p.IfIndex
		nextStpPortState.Dot1dStpPortPathCost = int32(p.PortPathCost) //The contribution of this port to the path cost of paths towards the spanning tree root which include this port.  802.1D-1998 recommends that the default value of this parameter be in inverse proportion to    the speed of the attached LAN.  New implementations should support dot1dStpPortPathCost32. If the port path costs exceeds the maximum value of this object then this object should report the maximum value, namely 65535.  Applications should try to read the dot1dStpPortPathCost32 object if this object reports the maximum value.
		nextStpPortState.Dot1dStpPortPriority = int32(p.Priority)     //The value of the priority field that is contained in the first (in network byte order) octet of the (2 octet long) Port ID.  The other octet of the Port ID is given by the value of dot1dStpPort. On bridges supporting IEEE 802.1t or IEEE 802.1w, permissible values are 0-240, in steps of 16.
		nextStpPortState.Dot1dStpPortDesignatedBridge = stp.CreateBridgeIdStr(p.PortPriority.DesignatedBridgeId)
		//nextStpPortState.Dot1dStpPortAdminPointToPoint  int32(p.)  //The administrative point-to-point status of the LAN segment attached to this port, using the enumeration values of the IEEE 802.1w clause.  A value of forceTrue(0) indicates that this port should always be treated as if it is connected to a point-to-point link.  A value of forceFalse(1) indicates that this port should be treated as having a shared media connection.  A value of auto(2) indicates that this port is considered to have a point-to-point link if it is an Aggregator and all of its    members are aggregatable, or if the MAC entity is configured for full duplex operation, either through auto-negotiation or by management means.  Manipulating this object changes the underlying adminPortToPortMAC.  The value of this object MUST be retained across reinitializations of the management system.
		nextStpPortState.Dot1dStpPortState = GetPortState(p)
		nextStpPortState.Dot1dStpPortEnable = ConvertBoolToInt32(p.PortEnabled)
		nextStpPortState.Dot1dStpPortDesignatedRoot = stp.CreateBridgeIdStr(p.PortPriority.RootBridgeId)
		nextStpPortState.Dot1dStpPortDesignatedCost = int32(p.PortPathCost)
		nextStpPortState.Dot1dStpPortAdminPathCost = p.AdminPathCost
		nextStpPortState.Dot1dStpPortPathCost32 = int32(p.PortPathCost)
		// root timers
		nextStpPortState.Dot1dStpBridgePortMaxAge = int32(p.PortTimes.MaxAge)
		nextStpPortState.Dot1dStpBridgePortForwardDelay = int32(p.PortTimes.ForwardingDelay)
		nextStpPortState.Dot1dStpBridgePortHelloTime = int32(p.PortTimes.HelloTime)
		// counters
		nextStpPortState.StpInPkts = int64(p.StpRx)
		nextStpPortState.StpOutPkts = int64(p.StpTx)
		nextStpPortState.RstpInPkts = int64(p.RstpRx)
		nextStpPortState.RstpOutPkts = int64(p.RstpTx)
		nextStpPortState.TcInPkts = int64(p.TcRx)
		nextStpPortState.TcOutPkts = int64(p.TcTx)
		nextStpPortState.PvstInPkts = int64(p.PvstRx)
		nextStpPortState.PvstOutPkts = int64(p.PvstTx)
		nextStpPortState.BpduInPkts = int64(p.BpduRx)
		nextStpPortState.BpduOutPkts = int64(p.BpduTx)
		// fsm-states
		nextStpPortState.PimPrevState = p.PimMachineFsm.GetPrevStateStr()
		nextStpPortState.PimCurrState = p.PimMachineFsm.GetCurrStateStr()
		nextStpPortState.PrtmPrevState = p.PrtMachineFsm.GetPrevStateStr()
		nextStpPortState.PrtmCurrState = p.PrtMachineFsm.GetCurrStateStr()
		nextStpPortState.PrxmPrevState = p.PrxmMachineFsm.GetPrevStateStr()
		nextStpPortState.PrxmCurrState = p.PrxmMachineFsm.GetCurrStateStr()
		nextStpPortState.PstmPrevState = p.PstMachineFsm.GetPrevStateStr()
		nextStpPortState.PstmCurrState = p.PstMachineFsm.GetCurrStateStr()
		nextStpPortState.TcmPrevState = p.TcMachineFsm.GetPrevStateStr()
		nextStpPortState.TcmCurrState = p.TcMachineFsm.GetCurrStateStr()
		nextStpPortState.PpmPrevState = p.PpmmMachineFsm.GetPrevStateStr()
		nextStpPortState.PpmCurrState = p.PpmmMachineFsm.GetCurrStateStr()
		nextStpPortState.PtxmPrevState = p.PtxmMachineFsm.GetPrevStateStr()
		nextStpPortState.PtxmCurrState = p.PtxmMachineFsm.GetCurrStateStr()
		nextStpPortState.PtimPrevState = p.PtmMachineFsm.GetPrevStateStr()
		nextStpPortState.PtimCurrState = p.PtmMachineFsm.GetCurrStateStr()
		nextStpPortState.BdmPrevState = p.BdmMachineFsm.GetPrevStateStr()
		nextStpPortState.BdmCurrState = p.BdmMachineFsm.GetCurrStateStr()

		if len(returnStpPortStates) == 0 {
			returnStpPortStates = make([]*stpd.Dot1dStpPortEntryStateCountersFsmStates, 0)
		}
		returnStpPortStates = append(returnStpPortStates, nextStpPortState)
		validCount++
		toIndex++
	}
	// lets try and get the next agg if one exists then there are more routes
	moreRoutes := false
	if fromIndex+count < stpPortListLen {
		moreRoutes = true
	}
	// lets try and get the next agg if one exists then there are more routes
	obj.Dot1dStpPortEntryStateCountersFsmStatesList = returnStpPortStates
	obj.StartIdx = fromIndex
	obj.EndIdx = toIndex + 1
	obj.More = moreRoutes
	obj.Count = validCount

	return obj, nil
}

// All function below are not used/supported
func (s *STPDServiceHandler) GetBulkDot1dTpDot1dTpPortEntry(fromIndex stpd.Int, count stpd.Int) (obj *stpd.Dot1dTpDot1dTpPortEntryGetInfo, err error) {
	return obj, nil
}

func (s *STPDServiceHandler) GetBulkDot1dTpDot1dTpFdbEntry(fromIndex stpd.Int, count stpd.Int) (obj *stpd.Dot1dTpDot1dTpFdbEntryGetInfo, err error) {
	return obj, nil
}

func (s *STPDServiceHandler) GetBulkDot1dBaseDot1dBasePortEntry(fromIndex stpd.Int, count stpd.Int) (obj *stpd.Dot1dBaseDot1dBasePortEntryGetInfo, err error) {
	return obj, nil
}

func (s *STPDServiceHandler) CreateDot1dStaticDot1dStaticEntry(config *stpd.Dot1dStaticDot1dStaticEntry) (bool, error) {
	return true, nil
}

func (s *STPDServiceHandler) UpdateDot1dStaticDot1dStaticEntry(origconfig *stpd.Dot1dStaticDot1dStaticEntry, newconfig *stpd.Dot1dStaticDot1dStaticEntry, attrset []bool) (bool, error) {
	return true, nil
}

func (s *STPDServiceHandler) DeleteDot1dStaticDot1dStaticEntry(config *stpd.Dot1dStaticDot1dStaticEntry) (bool, error) {
	return true, nil
}
