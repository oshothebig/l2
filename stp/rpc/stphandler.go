Copyright [2016] [SnapRoute Inc]

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

	 Unless required by applicable law or agreed to in writing, software
	 distributed under the License is distributed on an "AS IS" BASIS,
	 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
	 See the License for the specific language governing permissions and
	 limitations under the License.
// lahandler
package rpc

import (
	"fmt"
	stp "l2/stp/protocol"
	"models"
	"reflect"
	"stpd"
	"utils/dbutils"
	//"time"
	"errors"
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
func ConvertThriftBrgConfigToStpBrgConfig(config *stpd.StpBridgeInstance, brgconfig *stp.StpBridgeConfig) {

	brgconfig.Address = config.Address
	brgconfig.Priority = uint16(config.Priority)
	brgconfig.Vlan = uint16(config.Vlan)
	brgconfig.MaxAge = uint16(config.MaxAge)
	brgconfig.HelloTime = uint16(config.HelloTime)
	brgconfig.ForwardDelay = uint16(config.ForwardDelay)
	brgconfig.ForceVersion = int32(config.ForceVersion)
	brgconfig.TxHoldCount = int32(config.TxHoldCount)
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

func ConvertThriftPortConfigToStpPortConfig(config *stpd.StpPort, portconfig *stp.StpPortConfig) {

	portconfig.IfIndex = int32(config.IfIndex)
	portconfig.BrgIfIndex = int32(config.BrgIfIndex)
	portconfig.Priority = uint16(config.Priority)
	portconfig.Enable = ConvertInt32ToBool(config.Enable)
	portconfig.PathCost = int32(config.PathCost)
	portconfig.ProtocolMigration = int32(config.ProtocolMigration)
	portconfig.AdminPointToPoint = int32(config.AdminPointToPoint)
	portconfig.AdminEdgePort = ConvertInt32ToBool(config.AdminEdgePort)
	portconfig.AdminPathCost = int32(config.AdminPathCost)
	portconfig.BridgeAssurance = ConvertInt32ToBool(config.BridgeAssurance)
	portconfig.BpduGuard = ConvertInt32ToBool(config.BpduGuard)
	portconfig.BpduGuardInterval = config.BpduGuardInterval
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

func ConvertAddrToString(mac [6]uint8) string {
	return fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x",
		mac[0],
		mac[1],
		mac[2],
		mac[3],
		mac[4],
		mac[5])
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
	//stp.StpLogger("INFO", fmt.Sprintf("PortEnabled[%t] Learning[%t] Forwarding[%t]", p.PortEnabled, p.Learning, p.Forwarding))
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
func (s *STPDServiceHandler) CreateStpBridgeInstance(config *stpd.StpBridgeInstance) (bool, error) {

	stp.StpLogger("INFO", "CreateStpBridgeInstance (server): created ")
	stp.StpLogger("INFO", fmt.Sprintf("addr:", config.Address))
	stp.StpLogger("INFO", fmt.Sprintf("prio:", config.Priority))
	stp.StpLogger("INFO", fmt.Sprintf("vlan:", config.Vlan))
	stp.StpLogger("INFO", fmt.Sprintf("age:", config.MaxAge))
	stp.StpLogger("INFO", fmt.Sprintf("hello:", config.HelloTime))        // int32
	stp.StpLogger("INFO", fmt.Sprintf("fwddelay:", config.ForwardDelay))  // int32
	stp.StpLogger("INFO", fmt.Sprintf("version:", config.ForceVersion))   // int32
	stp.StpLogger("INFO", fmt.Sprintf("txHoldCount", config.TxHoldCount)) //

	brgconfig := &stp.StpBridgeConfig{}
	ConvertThriftBrgConfigToStpBrgConfig(config, brgconfig)

	if brgconfig.Vlan == 0 {
		brgconfig.Vlan = stp.DEFAULT_STP_BRIDGE_VLAN
	}

	err := stp.StpBrgConfigParamCheck(brgconfig)
	if err == nil {
		stp.StpBridgeCreate(brgconfig)
		return true, err
	}
	return false, err
}

func (s *STPDServiceHandler) HandleDbReadStpBridgeInstance(dbHdl *dbutils.DBUtil) error {
	if dbHdl != nil {
		var dbObj models.StpBridgeInstance
		objList, err := dbHdl.GetAllObjFromDb(dbObj)
		if err != nil {
			stp.StpLogger("ERROR", "DB Query failed when retrieving StpBridgeInstance objects")
			return err
		}
		for idx := 0; idx < len(objList); idx++ {
			obj := stpd.NewStpBridgeInstance()
			dbObject := objList[idx].(models.StpBridgeInstance)
			models.ConvertstpdStpBridgeInstanceObjToThrift(&dbObject, obj)
			_, err = s.CreateStpBridgeInstance(obj)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *STPDServiceHandler) HandleDbReadStpPort(dbHdl *dbutils.DBUtil) error {
	if dbHdl != nil {
		var dbObj models.StpPort
		objList, err := dbHdl.GetAllObjFromDb(dbObj)
		if err != nil {
			stp.StpLogger("ERROR", "DB Query failed when retrieving StpPort objects")
			return err
		}
		for idx := 0; idx < len(objList); idx++ {
			obj := stpd.NewStpPort()
			dbObject := objList[idx].(models.StpPort)
			models.ConvertstpdStpPortObjToThrift(&dbObject, obj)
			_, err = s.CreateStpPort(obj)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *STPDServiceHandler) ReadConfigFromDB() error {

	dbHdl := dbutils.NewDBUtil(nil)
	err := dbHdl.Connect()
	if err != nil {
		stp.StpLogger("ERROR", fmt.Sprintf("Failed to open connection to DB with error %s", err))
		return err
	}
	defer dbHdl.Disconnect()

	if err := s.HandleDbReadStpBridgeInstance(dbHdl); err != nil {
		stp.StpLogger("ERROR", "Error getting All StpBridgeInstance objects")
		return err
	}

	if err = s.HandleDbReadStpPort(dbHdl); err != nil {
		stp.StpLogger("ERROR", "Error getting All StpPort objects")
		return err
	}

	return nil
}

func (s *STPDServiceHandler) DeleteStpBridgeInstance(config *stpd.StpBridgeInstance) (bool, error) {

	// Aggregation found now lets delete
	//lacp.DeleteLaAgg(GetIdByName(config.NameKey))
	stp.StpLogger("INFO", "DeleteStpBridgeInstance (server): deleted ")
	brgconfig := &stp.StpBridgeConfig{}
	ConvertThriftBrgConfigToStpBrgConfig(config, brgconfig)
	err := stp.StpBridgeDelete(brgconfig)
	if err == nil {
		return true, err
	}
	return false, err
}

func (s *STPDServiceHandler) UpdateStpBridgeInstance(origconfig *stpd.StpBridgeInstance, updateconfig *stpd.StpBridgeInstance, attrset []bool) (bool, error) {
	var b *stp.Bridge
	brgconfig := &stp.StpBridgeConfig{}
	objTyp := reflect.TypeOf(*origconfig)
	//objVal := reflect.ValueOf(origconfig)
	//updateObjVal := reflect.ValueOf(*updateconfig)

	key := stp.BridgeKey{
		Vlan: uint16(origconfig.Vlan),
	}
	brgIfIndex := int32(0)
	if stp.StpFindBridgeById(key, &b) {
		brgIfIndex = b.BrgIfIndex
	} else {
		return false, errors.New("Unknown Bridge in update config")
	}

	ConvertThriftBrgConfigToStpBrgConfig(updateconfig, brgconfig)
	err := stp.StpBrgConfigParamCheck(brgconfig)
	if err != nil {
		return false, err
	}

	// important to note that the attrset starts at index 0 which is the BaseObj
	// which is not the first element on the thrift obj, thus we need to skip
	// this attribute
	for i := 0; i < objTyp.NumField(); i++ {
		objName := objTyp.Field(i).Name
		//fmt.Println("UpdateStpBridgeInstance (server): (index, objName) ", i, objName)
		if attrset[i] {
			stp.StpLogger("INFO", fmt.Sprintf("UpdateStpBridgeInstance (server): changed ", objName))

			if objName == "MaxAge" {
				stp.StpBrgMaxAgeSet(brgIfIndex, uint16(updateconfig.MaxAge))
			}
			if objName == "HelloTime" {
				stp.StpBrgHelloTimeSet(brgIfIndex, uint16(updateconfig.HelloTime))
			}
			if objName == "ForwardDelay" {
				stp.StpBrgForwardDelaySet(brgIfIndex, uint16(updateconfig.ForwardDelay))
			}
			if objName == "TxHoldCount" {
				stp.StpBrgTxHoldCountSet(brgIfIndex, uint16(updateconfig.TxHoldCount))
			}
			// causes re-selection
			if objName == "Priority" {
				stp.StpBrgPrioritySet(brgIfIndex, uint16(updateconfig.Priority))
			}
			// causes restart
			if objName == "ForceVersion" {
				stp.StpBrgForceVersion(brgIfIndex, updateconfig.ForceVersion)
			}
		}
	}
	return true, nil
}

func (s *STPDServiceHandler) CreateStpPort(config *stpd.StpPort) (bool, error) {
	stp.StpLogger("INFO", fmt.Sprintf("CreateStpPort (server): created %#v", config))
	portconfig := &stp.StpPortConfig{}
	ConvertThriftPortConfigToStpPortConfig(config, portconfig)
	err := stp.StpPortConfigParamCheck(portconfig)
	if err == nil {
		err = stp.StpPortCreate(portconfig)
		if err == nil {
			return true, err
		}
	}
	return false, err
}

func (s *STPDServiceHandler) DeleteStpPort(config *stpd.StpPort) (bool, error) {
	stp.StpLogger("INFO", "DeleteStpPort (server): deleted ")
	portconfig := &stp.StpPortConfig{}
	ConvertThriftPortConfigToStpPortConfig(config, portconfig)

	err := stp.StpPortDelete(portconfig)
	if err == nil {
		return true, err
	}
	return false, err
}

func (s *STPDServiceHandler) UpdateStpPort(origconfig *stpd.StpPort, updateconfig *stpd.StpPort, attrset []bool) (bool, error) {
	var p *stp.StpPort
	portconfig := &stp.StpPortConfig{}
	objTyp := reflect.TypeOf(*origconfig)
	//objVal := reflect.ValueOf(origconfig)
	//updateObjVal := reflect.ValueOf(*updateconfig)

	ifIndex := int32(origconfig.IfIndex)
	brgIfIndex := int32(origconfig.BrgIfIndex)
	if stp.StpFindPortByIfIndex(ifIndex, brgIfIndex, &p) {
		ifIndex = p.IfIndex
	} else {
		return false, errors.New("Unknown Stp port in update config")
	}

	ConvertThriftPortConfigToStpPortConfig(updateconfig, portconfig)
	err := stp.StpPortConfigParamCheck(portconfig)
	if err != nil {
		return false, err
	}
	err = stp.StpPortConfigSave(portconfig, true)
	if err != nil {
		return false, err
	}

	// important to note that the attrset starts at index 0 which is the BaseObj
	// which is not the first element on the thrift obj, thus we need to skip
	// this attribute
	for i := 0; i < objTyp.NumField(); i++ {
		objName := objTyp.Field(i).Name
		//fmt.Println("UpdateDot1dStpBridgeConfig (server): (index, objName) ", i, objName)
		if attrset[i] {
			stp.StpLogger("INFO", fmt.Sprintf("StpPort (server): changed ", objName))

			var err error
			if objName == "Priority" {
				err = stp.StpPortPrioritySet(ifIndex, brgIfIndex, uint16(updateconfig.Priority))
			}
			if objName == "Enable" {
				err = stp.StpPortEnable(ifIndex, brgIfIndex, ConvertInt32ToBool(updateconfig.Enable))
			}
			if objName == "PathCost" {
				err = stp.StpPortPortPathCostSet(ifIndex, brgIfIndex, uint32(updateconfig.PathCost))
			}
			if objName == "ProtocolMigration" {
				err = stp.StpPortProtocolMigrationSet(ifIndex, brgIfIndex, ConvertInt32ToBool(updateconfig.ProtocolMigration))
			}
			if objName == "AdminPointToPoint" {
				// TODO
			}
			if objName == "AdminEdgePort" {
				err = stp.StpPortAdminEdgeSet(ifIndex, brgIfIndex, ConvertInt32ToBool(updateconfig.AdminEdgePort))
			}
			if objName == "AdminPathCost" {
				// TODO
			}
			if objName == "BpduGuard" {
				// TOOD
				err = stp.StpPortBpduGuardSet(ifIndex, brgIfIndex, ConvertInt32ToBool(updateconfig.BpduGuard))
			}
			if objName == "BridgeAssurance" {
				err = stp.StpPortBridgeAssuranceSet(ifIndex, brgIfIndex, ConvertInt32ToBool(updateconfig.BridgeAssurance))

			}

			if err != nil {
				return false, err
			}
		}
	}

	return true, nil
}

func (s *STPDServiceHandler) GetStpBridgeState(vlan int16) (*stpd.StpBridgeState, error) {
	sbs := &stpd.StpBridgeState{}

	key := stp.BridgeKey{
		Vlan: uint16(vlan),
	}
	var b *stp.Bridge
	if stp.StpFindBridgeById(key, &b) {
		sbs.BridgeHelloTime = int32(b.BridgeTimes.HelloTime)
		sbs.TxHoldCount = stp.TransmitHoldCountDefault
		sbs.BridgeForwardDelay = int32(b.BridgeTimes.ForwardingDelay)
		sbs.BridgeMaxAge = int32(b.BridgeTimes.MaxAge)
		sbs.Address = ConvertAddrToString(stp.GetBridgeAddrFromBridgeId(b.BridgePriority.DesignatedBridgeId))
		sbs.Priority = int32(stp.GetBridgePriorityFromBridgeId(b.BridgePriority.DesignatedBridgeId))
		sbs.IfIndex = b.BrgIfIndex
		sbs.ProtocolSpecification = 2
		//nextStpBridgeState.TimeSinceTopologyChange uint32 //The time (in hundredths of a second) since the last time a topology change was detected by the bridge entity. For RSTP, this reports the time since the tcWhile timer for any port on this Bridge was nonzero.
		//nextStpBridgeState.TopChanges              uint32 //The total number of topology changes detected by this bridge since the management entity was last reset or initialized.
		sbs.DesignatedRoot = ConvertBridgeIdToString(b.BridgePriority.RootBridgeId)
		sbs.RootCost = int32(b.BridgePriority.RootPathCost)
		sbs.RootPort = int32(b.BridgePriority.DesignatedPortId)
		sbs.MaxAge = int32(b.RootTimes.MaxAge)
		sbs.HelloTime = int32(b.RootTimes.HelloTime)
		sbs.HoldTime = int32(b.TxHoldCount)
		sbs.ForwardDelay = int32(b.RootTimes.ForwardingDelay)
		sbs.Vlan = int16(b.Vlan)
	} else {
		return sbs, errors.New(fmt.Sprintf("STP: Error could not find bridge vlan %d", vlan))
	}

	return sbs, nil
}

// GetBulkAggregationLacpState will return the status of all the lag groups
// All lag groups are stored in a map, thus we will assume that the order
// at which a for loop iterates over the map is preserved.  It is assumed
// that at the time of this operation that no new aggragators are added,
// otherwise can get inconsistent results
func (s *STPDServiceHandler) GetBulkStpBridgeState(fromIndex stpd.Int, count stpd.Int) (obj *stpd.StpBridgeStateGetInfo, err error) {

	var stpBridgeStateList []stpd.StpBridgeState = make([]stpd.StpBridgeState, count)
	var nextStpBridgeState *stpd.StpBridgeState
	var returnStpBridgeStates []*stpd.StpBridgeState
	var returnStpBridgeStateGetInfo stpd.StpBridgeStateGetInfo
	var b *stp.Bridge
	validCount := stpd.Int(0)
	toIndex := fromIndex
	obj = &returnStpBridgeStateGetInfo
	brgListLen := stpd.Int(len(stp.BridgeListTable))
	for currIndex := fromIndex; validCount != count && currIndex < brgListLen; currIndex++ {

		b = stp.BridgeListTable[currIndex]
		nextStpBridgeState = &stpBridgeStateList[validCount]
		nextStpBridgeState.BridgeHelloTime = int32(b.BridgeTimes.HelloTime)
		nextStpBridgeState.TxHoldCount = stp.TransmitHoldCountDefault
		nextStpBridgeState.BridgeForwardDelay = int32(b.BridgeTimes.ForwardingDelay)
		nextStpBridgeState.BridgeMaxAge = int32(b.BridgeTimes.MaxAge)
		nextStpBridgeState.Address = ConvertAddrToString(stp.GetBridgeAddrFromBridgeId(b.BridgePriority.DesignatedBridgeId))
		nextStpBridgeState.Priority = int32(stp.GetBridgePriorityFromBridgeId(b.BridgePriority.DesignatedBridgeId))
		nextStpBridgeState.IfIndex = b.BrgIfIndex
		nextStpBridgeState.ProtocolSpecification = 2
		//nextStpBridgeState.TimeSinceTopologyChange uint32 //The time (in hundredths of a second) since the last time a topology change was detected by the bridge entity. For RSTP, this reports the time since the tcWhile timer for any port on this Bridge was nonzero.
		//nextStpBridgeState.TopChanges              uint32 //The total number of topology changes detected by this bridge since the management entity was last reset or initialized.
		nextStpBridgeState.DesignatedRoot = ConvertBridgeIdToString(b.BridgePriority.RootBridgeId)
		nextStpBridgeState.RootCost = int32(b.BridgePriority.RootPathCost)
		nextStpBridgeState.RootPort = int32(b.BridgePriority.DesignatedPortId)
		nextStpBridgeState.MaxAge = int32(b.RootTimes.MaxAge)
		nextStpBridgeState.HelloTime = int32(b.RootTimes.HelloTime)
		nextStpBridgeState.HoldTime = int32(b.TxHoldCount)
		nextStpBridgeState.ForwardDelay = int32(b.RootTimes.ForwardingDelay)
		nextStpBridgeState.Vlan = int16(b.Vlan)

		if len(returnStpBridgeStates) == 0 {
			returnStpBridgeStates = make([]*stpd.StpBridgeState, 0)
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
	obj.StpBridgeStateList = returnStpBridgeStates
	obj.StartIdx = fromIndex
	obj.EndIdx = toIndex + 1
	obj.More = moreRoutes
	obj.Count = validCount

	return obj, nil
}

func (s *STPDServiceHandler) GetStpPortState(ifIndex int32, brgIfIndex int32) (*stpd.StpPortState, error) {
	sps := &stpd.StpPortState{}

	var p *stp.StpPort
	if stp.StpFindPortByIfIndex(ifIndex, brgIfIndex, &p) {

		sps.OperPointToPoint = ConvertBoolToInt32(p.OperPointToPointMAC)
		sps.BrgIfIndex = p.BrgIfIndex
		sps.OperEdgePort = ConvertBoolToInt32(p.OperEdge)
		sps.DesignatedPort = fmt.Sprintf("%d", p.PortPriority.DesignatedPortId)
		sps.AdminEdgePort = ConvertBoolToInt32(p.AdminEdge)
		sps.ForwardTransitions = int32(p.ForwardingTransitions)
		//nextStpPortState.ProtocolMigration  int32  //When operating in RSTP (version 2) mode, writing true(1) to this object forces this port to transmit RSTP BPDUs. Any other operation on this object has no effect and it always returns false(2) when read.
		sps.IfIndex = p.IfIndex
		//nextStpPortState.PathCost = int32(p.PortPathCost) //The contribution of this port to the path cost of paths towards the spanning tree root which include this port.  802.1D-1998 recommends that the default value of this parameter be in inverse proportion to    the speed of the attached LAN.  New implementations should support PathCost32. If the port path costs exceeds the maximum value of this object then this object should report the maximum value, namely 65535.  Applications should try to read the PathCost32 object if this object reports the maximum value.
		sps.Priority = int32(p.Priority) //The value of the priority field that is contained in the first (in network byte order) octet of the (2 octet long) Port ID.  The other octet of the Port ID is given by the value of IfIndex. On bridges supporting IEEE 802.1t or IEEE 802.1w, permissible values are 0-240, in steps of 16.
		sps.DesignatedBridge = stp.CreateBridgeIdStr(p.PortPriority.DesignatedBridgeId)
		//nextStpPortState.AdminPointToPoint  int32(p.)  //The administrative point-to-point status of the LAN segment attached to this port, using the enumeration values of the IEEE 802.1w clause.  A value of forceTrue(0) indicates that this port should always be treated as if it is connected to a point-to-point link.  A value of forceFalse(1) indicates that this port should be treated as having a shared media connection.  A value of auto(2) indicates that this port is considered to have a point-to-point link if it is an Aggregator and all of its    members are aggregatable, or if the MAC entity is configured for full duplex operation, either through auto-negotiation or by management means.  Manipulating this object changes the underlying adminPortToPortMAC.  The value of this object MUST be retained across reinitializations of the management system.
		sps.State = GetPortState(p)
		sps.Enable = ConvertBoolToInt32(p.PortEnabled)
		sps.DesignatedRoot = stp.CreateBridgeIdStr(p.PortPriority.RootBridgeId)
		sps.DesignatedCost = int32(p.PortPathCost)
		//nextStpPortState.AdminPathCost = p.AdminPathCost
		//nextStpPortState.PathCost32 = int32(p.PortPathCost)
		// Bridge Assurance
		sps.BridgeAssuranceInconsistant = ConvertBoolToInt32(p.BridgeAssuranceInconsistant)
		sps.BridgeAssurance = ConvertBoolToInt32(p.BridgeAssurance)
		// Bpdu Guard
		sps.BpduGuard = ConvertBoolToInt32(p.BpduGuard)
		sps.BpduGuardDetected = ConvertBoolToInt32(p.BPDUGuardTimer.GetCount() != 0)
		// root timers
		sps.MaxAge = int32(p.PortTimes.MaxAge)
		sps.ForwardDelay = int32(p.PortTimes.ForwardingDelay)
		sps.HelloTime = int32(p.PortTimes.HelloTime)
		// counters
		sps.StpInPkts = int64(p.StpRx)
		sps.StpOutPkts = int64(p.StpTx)
		sps.RstpInPkts = int64(p.RstpRx)
		sps.RstpOutPkts = int64(p.RstpTx)
		sps.TcInPkts = int64(p.TcRx)
		sps.TcOutPkts = int64(p.TcTx)
		sps.TcAckInPkts = int64(p.TcAckRx)
		sps.TcAckOutPkts = int64(p.TcAckTx)
		sps.PvstInPkts = int64(p.PvstRx)
		sps.PvstOutPkts = int64(p.PvstTx)
		sps.BpduInPkts = int64(p.BpduRx)
		sps.BpduOutPkts = int64(p.BpduTx)
		// fsm-states
		sps.PimPrevState = p.PimMachineFsm.GetPrevStateStr()
		sps.PimCurrState = p.PimMachineFsm.GetCurrStateStr()
		sps.PrtmPrevState = p.PrtMachineFsm.GetPrevStateStr()
		sps.PrtmCurrState = p.PrtMachineFsm.GetCurrStateStr()
		sps.PrxmPrevState = p.PrxmMachineFsm.GetPrevStateStr()
		sps.PrxmCurrState = p.PrxmMachineFsm.GetCurrStateStr()
		sps.PstmPrevState = p.PstMachineFsm.GetPrevStateStr()
		sps.PstmCurrState = p.PstMachineFsm.GetCurrStateStr()
		sps.TcmPrevState = p.TcMachineFsm.GetPrevStateStr()
		sps.TcmCurrState = p.TcMachineFsm.GetCurrStateStr()
		sps.PpmPrevState = p.PpmmMachineFsm.GetPrevStateStr()
		sps.PpmCurrState = p.PpmmMachineFsm.GetCurrStateStr()
		sps.PtxmPrevState = p.PtxmMachineFsm.GetPrevStateStr()
		sps.PtxmCurrState = p.PtxmMachineFsm.GetCurrStateStr()
		sps.PtimPrevState = p.PtmMachineFsm.GetPrevStateStr()
		sps.PtimCurrState = p.PtmMachineFsm.GetCurrStateStr()
		sps.BdmPrevState = p.BdmMachineFsm.GetPrevStateStr()
		sps.BdmCurrState = p.BdmMachineFsm.GetCurrStateStr()
		// current counts
		sps.EdgeDelayWhile = p.EdgeDelayWhileTimer.GetCount()
		sps.FdWhile = p.FdWhileTimer.GetCount()
		sps.HelloWhen = p.HelloWhenTimer.GetCount()
		sps.MdelayWhile = p.MdelayWhiletimer.GetCount()
		sps.RbWhile = p.RbWhileTimer.GetCount()
		sps.RcvdInfoWhile = p.RcvdInfoWhiletimer.GetCount()
		sps.RrWhile = p.RrWhileTimer.GetCount()
		sps.TcWhile = p.TcWhileTimer.GetCount()
		sps.BaWhile = p.BAWhileTimer.GetCount()

	} else {
		return sps, errors.New(fmt.Sprintf("STP: Error unabled to locate bridge ifindex %d stp port ifindex %d", brgIfIndex, ifIndex))
	}
	return sps, nil
}

// GetBulkAggregationLacpMemberStateCounters will return the status of all
// the lag members.
func (s *STPDServiceHandler) GetBulkStpPortState(fromIndex stpd.Int, count stpd.Int) (obj *stpd.StpPortStateGetInfo, err error) {

	var stpPortStateList []stpd.StpPortState = make([]stpd.StpPortState, count)
	var nextStpPortState *stpd.StpPortState
	var returnStpPortStates []*stpd.StpPortState
	var returnStpPortStateGetInfo stpd.StpPortStateGetInfo
	//var a *lacp.LaAggregator
	validCount := stpd.Int(0)
	toIndex := fromIndex
	obj = &returnStpPortStateGetInfo
	stpPortListLen := stpd.Int(len(stp.PortListTable))
	stp.StpLogger("INFO", fmt.Sprintf("Total configured ports %d fromIndex %d count %d", stpPortListLen, fromIndex, count))
	for currIndex := fromIndex; validCount != count && currIndex < stpPortListLen; currIndex++ {

		//stp.StpLogger("INFO", fmt.Sprintf("CurrIndex %d stpPortListLen %d", currIndex, stpPortListLen))

		p := stp.PortListTable[currIndex]
		nextStpPortState = &stpPortStateList[validCount]

		nextStpPortState.OperPointToPoint = ConvertBoolToInt32(p.OperPointToPointMAC)
		nextStpPortState.BrgIfIndex = p.BrgIfIndex
		nextStpPortState.OperEdgePort = ConvertBoolToInt32(p.OperEdge)
		nextStpPortState.DesignatedPort = fmt.Sprintf("%d", p.PortPriority.DesignatedPortId)
		nextStpPortState.AdminEdgePort = ConvertBoolToInt32(p.AdminEdge)
		nextStpPortState.ForwardTransitions = int32(p.ForwardingTransitions)
		//nextStpPortState.ProtocolMigration  int32  //When operating in RSTP (version 2) mode, writing true(1) to this object forces this port to transmit RSTP BPDUs. Any other operation on this object has no effect and it always returns false(2) when read.
		nextStpPortState.IfIndex = p.IfIndex
		//nextStpPortState.PathCost = int32(p.PortPathCost) //The contribution of this port to the path cost of paths towards the spanning tree root which include this port.  802.1D-1998 recommends that the default value of this parameter be in inverse proportion to    the speed of the attached LAN.  New implementations should support PathCost32. If the port path costs exceeds the maximum value of this object then this object should report the maximum value, namely 65535.  Applications should try to read the PathCost32 object if this object reports the maximum value.
		nextStpPortState.Priority = int32(p.Priority) //The value of the priority field that is contained in the first (in network byte order) octet of the (2 octet long) Port ID.  The other octet of the Port ID is given by the value of IfIndex. On bridges supporting IEEE 802.1t or IEEE 802.1w, permissible values are 0-240, in steps of 16.
		nextStpPortState.DesignatedBridge = stp.CreateBridgeIdStr(p.PortPriority.DesignatedBridgeId)
		//nextStpPortState.AdminPointToPoint  int32(p.)  //The administrative point-to-point status of the LAN segment attached to this port, using the enumeration values of the IEEE 802.1w clause.  A value of forceTrue(0) indicates that this port should always be treated as if it is connected to a point-to-point link.  A value of forceFalse(1) indicates that this port should be treated as having a shared media connection.  A value of auto(2) indicates that this port is considered to have a point-to-point link if it is an Aggregator and all of its    members are aggregatable, or if the MAC entity is configured for full duplex operation, either through auto-negotiation or by management means.  Manipulating this object changes the underlying adminPortToPortMAC.  The value of this object MUST be retained across reinitializations of the management system.
		nextStpPortState.State = GetPortState(p)
		nextStpPortState.Enable = ConvertBoolToInt32(p.PortEnabled)
		nextStpPortState.DesignatedRoot = stp.CreateBridgeIdStr(p.PortPriority.RootBridgeId)
		nextStpPortState.DesignatedCost = int32(p.PortPathCost)
		//nextStpPortState.AdminPathCost = p.AdminPathCost
		//nextStpPortState.PathCost32 = int32(p.PortPathCost)
		// Bridge Assurance
		nextStpPortState.BridgeAssuranceInconsistant = ConvertBoolToInt32(p.BridgeAssuranceInconsistant)
		nextStpPortState.BridgeAssurance = ConvertBoolToInt32(p.BridgeAssurance)
		// Bpdu Guard
		nextStpPortState.BpduGuard = ConvertBoolToInt32(p.BpduGuard)
		nextStpPortState.BpduGuardDetected = ConvertBoolToInt32(p.BPDUGuardTimer.GetCount() != 0)
		// root timers
		nextStpPortState.MaxAge = int32(p.PortTimes.MaxAge)
		nextStpPortState.ForwardDelay = int32(p.PortTimes.ForwardingDelay)
		nextStpPortState.HelloTime = int32(p.PortTimes.HelloTime)
		// counters
		nextStpPortState.StpInPkts = int64(p.StpRx)
		nextStpPortState.StpOutPkts = int64(p.StpTx)
		nextStpPortState.RstpInPkts = int64(p.RstpRx)
		nextStpPortState.RstpOutPkts = int64(p.RstpTx)
		nextStpPortState.TcInPkts = int64(p.TcRx)
		nextStpPortState.TcOutPkts = int64(p.TcTx)
		nextStpPortState.TcAckInPkts = int64(p.TcAckRx)
		nextStpPortState.TcAckOutPkts = int64(p.TcAckTx)
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
		// current counts
		nextStpPortState.EdgeDelayWhile = p.EdgeDelayWhileTimer.GetCount()
		nextStpPortState.FdWhile = p.FdWhileTimer.GetCount()
		nextStpPortState.HelloWhen = p.HelloWhenTimer.GetCount()
		nextStpPortState.MdelayWhile = p.MdelayWhiletimer.GetCount()
		nextStpPortState.RbWhile = p.RbWhileTimer.GetCount()
		nextStpPortState.RcvdInfoWhile = p.RcvdInfoWhiletimer.GetCount()
		nextStpPortState.RrWhile = p.RrWhileTimer.GetCount()
		nextStpPortState.TcWhile = p.TcWhileTimer.GetCount()
		nextStpPortState.BaWhile = p.BAWhileTimer.GetCount()

		if len(returnStpPortStates) == 0 {
			returnStpPortStates = make([]*stpd.StpPortState, 0)
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
	obj.StpPortStateList = returnStpPortStates
	obj.StartIdx = fromIndex
	obj.EndIdx = toIndex + 1
	obj.More = moreRoutes
	obj.Count = validCount

	return obj, nil
}
