namespace go stpd
typedef i32 int
typedef i16 uint16
struct Dot1dStpPortEntryConfig{
	1 : i32 	Dot1dStpPort
	2 : i32 	Dot1dBrgIfIndex
	3 : i32 	Dot1dStpPortPriority
	4 : i32 	Dot1dStpPortEnable
	5 : i32 	Dot1dStpPortPathCost
	6 : i32 	Dot1dStpPortPathCost32
	7 : i32 	Dot1dStpPortProtocolMigration
	8 : i32 	Dot1dStpPortAdminPointToPoint
	9 : i32 	Dot1dStpPortAdminEdgePort
	10 : i32 	Dot1dStpPortAdminPathCost
}
struct Dot1dStpPortEntryStateCounters{
	1 : i32 	Dot1dStpPortOperPointToPoint
	2 : i32 	Dot1dBrgIfIndex
	3 : i32 	Dot1dStpPortOperEdgePort
	4 : string 	Dot1dStpPortDesignatedPort
	5 : i32 	Dot1dStpPortAdminEdgePort
	6 : i32 	Dot1dStpPortForwardTransitions
	7 : i32 	Dot1dStpPortProtocolMigration
	8 : i32 	Dot1dStpPort
	9 : i32 	Dot1dStpPortPathCost
	10 : i32 	Dot1dStpPortPriority
	11 : string 	Dot1dStpPortDesignatedBridge
	12 : i32 	Dot1dStpPortAdminPointToPoint
	13 : i32 	Dot1dStpPortState
	14 : i32 	Dot1dStpPortEnable
	15 : string 	Dot1dStpPortDesignatedRoot
	16 : i32 	Dot1dStpPortDesignatedCost
	17 : i32 	Dot1dStpPortAdminPathCost
	18 : i32 	Dot1dStpPortPathCost32
	19 : i64 	StpInPkts
	20 : i64 	StpOutPkts
	21 : i64 	RstpInPkts
	22 : i64 	RstpOutPkts
	23 : i64 	TcInPkts
	24 : i64 	TcOutPkts
	25 : i64 	PvstInPkts
	26 : i64 	PvstOutPkts
	27 : i64 	BpduInPkts
	28 : i64 	BpduOutPkts
}
struct Dot1dStpPortEntryStateCountersGetInfo {
	1: int StartIdx
	2: int EndIdx
	3: int Count
	4: bool More
	5: list<Dot1dStpPortEntryStateCounters> Dot1dStpPortEntryStateCountersList
}
struct Dot1dStpBridgeConfig{
	1 : string 	Dot1dBridgeAddress
	2 : i16 	Dot1dStpVlan
	3 : i32 	Dot1dStpPriority
	4 : i32 	Dot1dStpBridgeMaxAge
	5 : i32 	Dot1dStpBridgeHelloTime
	6 : i32 	Dot1dStpBridgeForwardDelay
	7 : i32 	Dot1dStpBridgeForceVersion
	8 : i32 	Dot1dStpBridgeTxHoldCount
}
struct Dot1dStpBridgeState{
	1 : i32 	Dot1dStpBridgeForceVersion
	2 : string 	Dot1dBridgeAddress
	3 : i32 	Dot1dStpBridgeHelloTime
	4 : i32 	Dot1dStpBridgeTxHoldCount
	5 : i16 	Dot1dStpVlan
	6 : i32 	Dot1dStpBridgeForwardDelay
	7 : i32 	Dot1dStpBridgeMaxAge
	8 : i32 	Dot1dStpPriority
	9 : i32 	Dot1dBrgIfIndex
	10 : i32 	Dot1dStpProtocolSpecification
	11 : i32 	Dot1dStpTimeSinceTopologyChange
	12 : i32 	Dot1dStpTopChanges
	13 : string 	Dot1dStpDesignatedRoot
	14 : i32 	Dot1dStpRootCost
	15 : i32 	Dot1dStpRootPort
	16 : i32 	Dot1dStpMaxAge
	17 : i32 	Dot1dStpHelloTime
	18 : i32 	Dot1dStpHoldTime
	19 : i32 	Dot1dStpForwardDelay
}
struct Dot1dStpBridgeStateGetInfo {
	1: int StartIdx
	2: int EndIdx
	3: int Count
	4: bool More
	5: list<Dot1dStpBridgeState> Dot1dStpBridgeStateList
}
service STPDServices {
	bool CreateDot1dStpPortEntryConfig(1: Dot1dStpPortEntryConfig config);
	bool UpdateDot1dStpPortEntryConfig(1: Dot1dStpPortEntryConfig origconfig, 2: Dot1dStpPortEntryConfig newconfig, 3: list<bool> attrset);
	bool DeleteDot1dStpPortEntryConfig(1: Dot1dStpPortEntryConfig config);

	Dot1dStpPortEntryStateCountersGetInfo GetBulkDot1dStpPortEntryStateCounters(1: int fromIndex, 2: int count);
	bool CreateDot1dStpBridgeConfig(1: Dot1dStpBridgeConfig config);
	bool UpdateDot1dStpBridgeConfig(1: Dot1dStpBridgeConfig origconfig, 2: Dot1dStpBridgeConfig newconfig, 3: list<bool> attrset);
	bool DeleteDot1dStpBridgeConfig(1: Dot1dStpBridgeConfig config);

	Dot1dStpBridgeStateGetInfo GetBulkDot1dStpBridgeState(1: int fromIndex, 2: int count);
}