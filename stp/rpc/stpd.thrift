namespace go stpd
typedef i32 int
typedef i16 uint16
struct Dot1dBaseDot1dBasePortEntry{
	1 : i32 Dot1dBaseType
	2 : string Dot1dBaseBridgeAddress
	3 : i32 Dot1dBasePort
	4 : i32 Dot1dBaseNumPorts
	5 : i32 Dot1dBasePortIfIndex
	6 : string Dot1dBasePortCircuit
	7 : i32 Dot1dBasePortDelayExceededDiscards
	8 : i32 Dot1dBasePortMtuExceededDiscards
}
struct Dot1dBaseDot1dBasePortEntryGetInfo {
	1: int StartIdx
	2: int EndIdx
	3: int Count
	4: bool More
	5: list<Dot1dBaseDot1dBasePortEntry> Dot1dBaseDot1dBasePortEntryList
}
struct Dot1dTpDot1dTpPortEntry{
	1 : i32 Dot1dTpPort
	2 : i32 Dot1dTpLearnedEntryDiscards
	3 : i32 Dot1dTpAgingTime
	4 : i32 Dot1dTpPortMaxInfo
	5 : i32 Dot1dTpPortInFrames
	6 : i32 Dot1dTpPortOutFrames
	7 : i32 Dot1dTpPortInDiscards
}
struct Dot1dTpDot1dTpPortEntryGetInfo {
	1: int StartIdx
	2: int EndIdx
	3: int Count
	4: bool More
	5: list<Dot1dTpDot1dTpPortEntry> Dot1dTpDot1dTpPortEntryList
}
struct Dot1dStaticDot1dStaticEntry{
	1 : string Dot1dStaticAddress
	2 : i32 Dot1dStaticReceivePort
	3 : string Dot1dStaticAllowedToGoTo
	4 : i32 Dot1dStaticStatus
}
struct Dot1dStpPortEntryStateCountersFsmStatesPortTimer{
	1 : i64 RstpOutPkts
	2 : string PtxmPrevState
	3 : i64 RstpInPkts
	4 : string PpmCurrState
	5 : string Dot1dStpPortDesignatedPort
	6 : i32 Dot1dStpBridgePortForwardDelay
	7 : i32 Dot1dBrgIfIndex
	8 : string PimPrevState
	9 : i32 Dot1dStpPortPathCost
	10 : string PrxmPrevState
	11 : string PpmPrevState
	12 : string PrtmCurrState
	13 : string Dot1dStpPortDesignatedBridge
	14 : i64 TcOutPkts
	15 : i32 Dot1dStpPortAdminPointToPoint
	16 : string PtimPrevState
	17 : i64 PvstInPkts
	18 : string PstmPrevState
	19 : i32 Dot1dStpPortProtocolMigration
	20 : i32 Dot1dStpPortState
	21 : i32 Dot1dStpPortEnable
	22 : string Dot1dStpPortDesignatedRoot
	23 : i64 BpduOutPkts
	24 : i32 Dot1dStpPortDesignatedCost
	25 : i32 Dot1dStpBridgePortMaxAge
	26 : string TcmCurrState
	27 : i32 Dot1dStpPortOperPointToPoint
	28 : string TcmPrevState
	29 : i32 Dot1dStpPortOperEdgePort
	30 : string PstmCurrState
	31 : i32 Dot1dStpPortAdminEdgePort
	32 : i64 PvstOutPkts
	33 : string BdmCurrState
	34 : string PtimCurrState
	35 : string PimCurrState
	36 : i32 Dot1dStpPort
	37 : i32 Dot1dStpBridgePortHelloTime
	38 : i64 BpduInPkts
	39 : i32 Dot1dStpPortForwardTransitions
	40 : string BdmPrevState
	41 : string PrxmCurrState
	42 : i32 Dot1dStpPortPriority
	43 : string PrtmPrevState
	44 : i64 StpInPkts
	45 : string PtxmCurrState
	46 : i64 TcInPkts
	47 : i32 Dot1dStpPortAdminPathCost
	48 : i64 StpOutPkts
	49 : i32 Dot1dStpPortPathCost32
	50 : i32 EdgeDelayWhile
	51 : i32 FdWhile
	52 : i32 HelloWhen
	53 : i32 MdelayWhile
	54 : i32 RbWhile
	55 : i32 RcvdInfoWhile
	56 : i32 RrWhile
	57 : i32 TcWhile
}
struct Dot1dStpPortEntryStateCountersFsmStatesPortTimerGetInfo {
	1: int StartIdx
	2: int EndIdx
	3: int Count
	4: bool More
	5: list<Dot1dStpPortEntryStateCountersFsmStatesPortTimer> Dot1dStpPortEntryStateCountersFsmStatesPortTimerList
}
struct Dot1dStpPortEntryConfig{
	1 : i32 Dot1dStpPort
	2 : i32 Dot1dBrgIfIndex
	3 : i32 Dot1dStpPortPriority
	4 : i32 Dot1dStpPortEnable
	5 : i32 Dot1dStpPortPathCost
	6 : i32 Dot1dStpPortPathCost32
	7 : i32 Dot1dStpPortProtocolMigration
	8 : i32 Dot1dStpPortAdminPointToPoint
	9 : i32 Dot1dStpPortAdminEdgePort
	10 : i32 Dot1dStpPortAdminPathCost
}
struct Dot1dStpBridgeState{
	1 : i32 Dot1dStpBridgeForceVersion
	2 : string Dot1dBridgeAddress
	3 : i32 Dot1dStpBridgeHelloTime
	4 : i32 Dot1dStpBridgeTxHoldCount
	5 : i16 Dot1dStpVlan
	6 : i32 Dot1dStpBridgeForwardDelay
	7 : i32 Dot1dStpBridgeMaxAge
	8 : i32 Dot1dStpPriority
	9 : i32 Dot1dBrgIfIndex
	10 : i32 Dot1dStpProtocolSpecification
	11 : i32 Dot1dStpTimeSinceTopologyChange
	12 : i32 Dot1dStpTopChanges
	13 : string Dot1dStpDesignatedRoot
	14 : i32 Dot1dStpRootCost
	15 : i32 Dot1dStpRootPort
	16 : i32 Dot1dStpMaxAge
	17 : i32 Dot1dStpHelloTime
	18 : i32 Dot1dStpHoldTime
	19 : i32 Dot1dStpForwardDelay
}
struct Dot1dStpBridgeStateGetInfo {
	1: int StartIdx
	2: int EndIdx
	3: int Count
	4: bool More
	5: list<Dot1dStpBridgeState> Dot1dStpBridgeStateList
}
struct Dot1dTpDot1dTpFdbEntry{
	1 : i32 Dot1dTpAgingTime
	2 : i32 Dot1dTpLearnedEntryDiscards
	3 : string Dot1dTpFdbAddress
	4 : i32 Dot1dTpFdbPort
	5 : i32 Dot1dTpFdbStatus
}
struct Dot1dTpDot1dTpFdbEntryGetInfo {
	1: int StartIdx
	2: int EndIdx
	3: int Count
	4: bool More
	5: list<Dot1dTpDot1dTpFdbEntry> Dot1dTpDot1dTpFdbEntryList
}
struct Dot1dStpBridgeConfig{
	1 : string Dot1dBridgeAddress
	2 : i16 Dot1dStpVlan
	3 : i32 Dot1dStpPriority
	4 : i32 Dot1dStpBridgeMaxAge
	5 : i32 Dot1dStpBridgeHelloTime
	6 : i32 Dot1dStpBridgeForwardDelay
	7 : i32 Dot1dStpBridgeForceVersion
	8 : i32 Dot1dStpBridgeTxHoldCount
}
service STPDServices {
	Dot1dBaseDot1dBasePortEntryGetInfo GetBulkDot1dBaseDot1dBasePortEntry(1: int fromIndex, 2: int count);
	Dot1dTpDot1dTpPortEntryGetInfo GetBulkDot1dTpDot1dTpPortEntry(1: int fromIndex, 2: int count);
	bool CreateDot1dStaticDot1dStaticEntry(1: Dot1dStaticDot1dStaticEntry config);
	bool UpdateDot1dStaticDot1dStaticEntry(1: Dot1dStaticDot1dStaticEntry origconfig, 2: Dot1dStaticDot1dStaticEntry newconfig, 3: list<bool> attrset);
	bool DeleteDot1dStaticDot1dStaticEntry(1: Dot1dStaticDot1dStaticEntry config);

	Dot1dStpPortEntryStateCountersFsmStatesPortTimerGetInfo GetBulkDot1dStpPortEntryStateCountersFsmStatesPortTimer(1: int fromIndex, 2: int count);
	bool CreateDot1dStpPortEntryConfig(1: Dot1dStpPortEntryConfig config);
	bool UpdateDot1dStpPortEntryConfig(1: Dot1dStpPortEntryConfig origconfig, 2: Dot1dStpPortEntryConfig newconfig, 3: list<bool> attrset);
	bool DeleteDot1dStpPortEntryConfig(1: Dot1dStpPortEntryConfig config);

	Dot1dStpBridgeStateGetInfo GetBulkDot1dStpBridgeState(1: int fromIndex, 2: int count);
	Dot1dTpDot1dTpFdbEntryGetInfo GetBulkDot1dTpDot1dTpFdbEntry(1: int fromIndex, 2: int count);
	bool CreateDot1dStpBridgeConfig(1: Dot1dStpBridgeConfig config);
	bool UpdateDot1dStpBridgeConfig(1: Dot1dStpBridgeConfig origconfig, 2: Dot1dStpBridgeConfig newconfig, 3: list<bool> attrset);
	bool DeleteDot1dStpBridgeConfig(1: Dot1dStpBridgeConfig config);

}