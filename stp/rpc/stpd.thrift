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
struct Dot1dStpPortEntryStateCountersFsmStates{
	1 : i64 	RstpOutPkts
	2 : i64 	RstpInPkts
	3 : string 	Dot1dStpPortDesignatedPort
	4 : i32 	Dot1dStpBridgePortForwardDelay
	5 : i32 	Dot1dStpPortPathCost
	6 : string 	Dot1dStpPortDesignatedBridge
	7 : i64 	TcOutPkts
	8 : i32 	Dot1dStpPortAdminPointToPoint
	9 : i64 	PvstInPkts
	10 : i32 	Dot1dStpPortProtocolMigration
	11 : i32 	Dot1dStpPortState
	12 : i32 	Dot1dStpPortEnable
	13 : string 	Dot1dStpPortDesignatedRoot
	14 : i64 	BpduOutPkts
	15 : i32 	Dot1dStpBridgePortMaxAge
	16 : i32 	Dot1dStpPortOperPointToPoint
	17 : i32 	Dot1dBrgIfIndex
	18 : i32 	Dot1dStpPortOperEdgePort
	19 : i32 	Dot1dStpPortAdminEdgePort
	20 : i32 	Dot1dStpPortForwardTransitions
	21 : i64 	PvstOutPkts
	22 : i64 	StpOutPkts
	23 : i32 	Dot1dStpPort
	24 : i32 	Dot1dStpBridgePortHelloTime
	25 : i64 	BpduInPkts
	26 : i32 	Dot1dStpPortPriority
	27 : i64 	StpInPkts
	28 : i64 	TcInPkts
	29 : i32 	Dot1dStpPortAdminPathCost
	30 : i32 	Dot1dStpPortPathCost32
	31 : i32 	Dot1dStpPortDesignatedCost
	32 : string 	PimPrevState
	33 : string 	PimCurrState
	34 : string 	PrtmPrevState
	35 : string 	PrtmCurrState
	36 : string 	PrxmPrevState
	37 : string 	PrxmCurrState
	38 : string 	PstmPrevState
	39 : string 	PstmCurrState
	40 : string 	TcmPrevState
	41 : string 	TcmCurrState
	42 : string 	PpmPrevState
	43 : string 	PpmCurrState
	44 : string 	PtxmPrevState
	45 : string 	PtxmCurrState
	46 : string 	PtimPrevState
	47 : string 	PtimCurrState
	48 : string 	BdmPrevState
	49 : string 	BdmCurrState
}
struct Dot1dStpPortEntryStateCountersFsmStatesGetInfo {
	1: int StartIdx
	2: int EndIdx
	3: int Count
	4: bool More
	5: list<Dot1dStpPortEntryStateCountersFsmStates> Dot1dStpPortEntryStateCountersFsmStatesList
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

	Dot1dStpPortEntryStateCountersFsmStatesGetInfo GetBulkDot1dStpPortEntryStateCountersFsmStates(1: int fromIndex, 2: int count);
	bool CreateDot1dStpBridgeConfig(1: Dot1dStpBridgeConfig config);
	bool UpdateDot1dStpBridgeConfig(1: Dot1dStpBridgeConfig origconfig, 2: Dot1dStpBridgeConfig newconfig, 3: list<bool> attrset);
	bool DeleteDot1dStpBridgeConfig(1: Dot1dStpBridgeConfig config);

	Dot1dStpBridgeStateGetInfo GetBulkDot1dStpBridgeState(1: int fromIndex, 2: int count);
}