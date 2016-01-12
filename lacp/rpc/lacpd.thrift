namespace go lacpd
typedef i32 int
typedef i16 uint16
struct EthernetConfig{
	1 : string 	NameKey
	2 : bool 	Enabled
	3 : string 	Description
	4 : i16 	Mtu
	5 : string 	Type
	6 : string 	MacAddress
	7 : i32 	DuplexMode
	8 : bool 	Auto
	9 : string 	Speed
	10 : bool 	EnableFlowControl
	11 : string 	AggregateId
}
struct AggregationLacpConfig{
	1 : i32 	LagType
	2 : bool 	Enabled
	3 : string 	Description
	4 : i16 	Mtu
	5 : string 	Type
	6 : i16 	MinLinks
	7 : string 	NameKey
	8 : i32 	Interval
	9 : i32 	LacpMode
	10 : string 	SystemIdMac
	11 : i16 	SystemPriority
	12 : i32 	LagHash
}
struct AggregationLacpState{
	1 : i32 	LagType
	2 : bool 	Enabled
	3 : string 	Description
	4 : i16 	Mtu
	5 : string 	Type
	6 : i16 	MinLinks
	7 : string 	NameKey
	8 : i32 	Interval
	9 : i32 	LacpMode
	10 : string 	SystemIdMac
	11 : i16 	SystemPriority
	12 : i32 	LagHash
}
struct AggregationLacpStateGetInfo {
	1: int StartIdx
	2: int EndIdx
	3: int Count
	4: bool More
	5: list<AggregationLacpState> AggregationLacpStateList
}
struct AggregationLacpMemberStateCounters{
	1 : bool 	Collecting
	2 : i32 	PartnerCdsChurnMachine
	3 : i16 	OperKey
	4 : string 	MuxReason
	5 : string 	PartnerId
	6 : bool 	Aggregatable
	7 : i64 	PartnerCdsChurnCount
	8 : i32 	LagHash
	9 : i16 	PartnerKey
	10 : i32 	RxMachine
	11 : i64 	ActorSyncTransitionCount
	12 : string 	SystemId
	13 : i64 	ActorChangeCount
	14 : i32 	LacpMode
	15 : bool 	Distributing
	16 : i32 	MuxMachine
	17 : i64 	ActorCdsChurnCount
	18 : i64 	PartnerSyncTransitionCount
	19 : i64 	ActorChurnCount
	20 : i16 	SystemPriority
	21 : string 	Type
	22 : i16 	MinLinks
	23 : string 	Description
	24 : i64 	PartnerChangeCount
	25 : i32 	Synchronization
	26 : string 	Interface
	27 : i32 	DebugId
	28 : i32 	RxTime
	29 : i32 	ActorChurnMachine
	30 : i32 	LagType
	31 : i32 	ActorCdsChurnMachine
	32 : string 	NameKey
	33 : string 	SystemIdMac
	34 : i32 	Interval
	35 : bool 	Enabled
	36 : i16 	Mtu
	37 : i32 	Timeout
	38 : i32 	Activity
	39 : i32 	PartnerChurnMachine
	40 : i64 	PartnerChurnCount
	41 : i64 	LacpInPkts
	42 : i64 	LacpOutPkts
	43 : i64 	LacpRxErrors
	44 : i64 	LacpTxErrors
	45 : i64 	LacpUnknownErrors
	46 : i64 	LacpErrors
}
struct AggregationLacpMemberStateCountersGetInfo {
	1: int StartIdx
	2: int EndIdx
	3: int Count
	4: bool More
	5: list<AggregationLacpMemberStateCounters> AggregationLacpMemberStateCountersList
}
service LACPDServices {
	bool CreateEthernetConfig(1: EthernetConfig config);
	bool UpdateEthernetConfig(1: EthernetConfig origconfig, 2: EthernetConfig newconfig, 3: list<bool> attrset);
	bool DeleteEthernetConfig(1: EthernetConfig config);

	bool CreateAggregationLacpConfig(1: AggregationLacpConfig config);
	bool UpdateAggregationLacpConfig(1: AggregationLacpConfig origconfig, 2: AggregationLacpConfig newconfig, 3: list<bool> attrset);
	bool DeleteAggregationLacpConfig(1: AggregationLacpConfig config);

	AggregationLacpStateGetInfo GetBulkAggregationLacpState(1: int fromIndex, 2: int count);
	AggregationLacpMemberStateCountersGetInfo GetBulkAggregationLacpMemberStateCounters(1: int fromIndex, 2: int count);
}