namespace go lacpd
typedef i32 int
typedef i16 uint16
struct LaPortChannel {
	1 : i32 LagId
	2 : i32 LagType
	3 : i16 MinLinks
	4 : i32 Interval
	5 : i32 LacpMode
	6 : string SystemIdMac
	7 : i16 SystemPriority
	8 : i32 LagHash
	9 : string AdminState
	10 : list<i32> Members
}
struct LaPortChannelMemberState {
	1 : i32 IfIndex
	2 : i32 LagId
	3 : string OperState
	4 : i32 LagIfIndex
	5 : i32 Activity
	6 : i32 Timeout
	7 : i32 Synchronization
	8 : bool Aggregatable
	9 : bool Collecting
	10 : bool Distributing
	11 : bool Defaulted
	12 : string SystemId
	13 : i16 OperKey
	14 : string PartnerId
	15 : i16 PartnerKey
	16 : i32 DebugId
	17 : i32 RxMachine
	18 : i32 RxTime
	19 : i32 MuxMachine
	20 : string MuxReason
	21 : i32 ActorChurnMachine
	22 : i32 PartnerChurnMachine
	23 : i64 ActorChurnCount
	24 : i64 PartnerChurnCount
	25 : i64 ActorSyncTransitionCount
	26 : i64 PartnerSyncTransitionCount
	27 : i64 ActorChangeCount
	28 : i64 PartnerChangeCount
	29 : i32 ActorCdsChurnMachine
	30 : i32 PartnerCdsChurnMachine
	31 : i64 ActorCdsChurnCount
	32 : i64 PartnerCdsChurnCount
	33 : i64 LacpInPkts
	34 : i64 LacpOutPkts
	35 : i64 LacpRxErrors
	36 : i64 LacpTxErrors
	37 : i64 LacpUnknownErrors
	38 : i64 LacpErrors
	39 : i64 LampInPdu
	40 : i64 LampInResponsePdu
	41 : i64 LampOutPdu
	42 : i64 LampOutResponsePdu
}
struct LaPortChannelMemberStateGetInfo {
	1: int StartIdx
	2: int EndIdx
	3: int Count
	4: bool More
	5: list<LaPortChannelMemberState> LaPortChannelMemberStateList
}
struct LaPortChannelState {
	1 : i32 LagId
	2 : i32 IfIndex
	3 : string Name
	4 : i32 LagType
	5 : i16 MinLinks
	6 : i32 Interval
	7 : i32 LacpMode
	8 : string SystemIdMac
	9 : i16 SystemPriority
	10 : i32 LagHash
	11 : string AdminState
	12 : string OperState
	13 : list<i32> Members
	14 : list<i32> MembersUpInBundle
}
struct LaPortChannelStateGetInfo {
	1: int StartIdx
	2: int EndIdx
	3: int Count
	4: bool More
	5: list<LaPortChannelState> LaPortChannelStateList
}
service LACPDServices {
	bool CreateLaPortChannel(1: LaPortChannel config);
	bool UpdateLaPortChannel(1: LaPortChannel origconfig, 2: LaPortChannel newconfig, 3: list<bool> attrset);
	bool DeleteLaPortChannel(1: LaPortChannel config);

	LaPortChannelMemberStateGetInfo GetBulkLaPortChannelMemberState(1: int fromIndex, 2: int count);
	LaPortChannelStateGetInfo GetBulkLaPortChannelState(1: int fromIndex, 2: int count);
}