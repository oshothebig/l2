namespace go lacpdServices
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
	2 : i16 	OperKey
	3 : string 	PartnerId
	4 : string 	Interface
	5 : i32 	Synchronization
	6 : bool 	Aggregatable
	7 : i16 	Mtu
	8 : string 	NameKey
	9 : i32 	LacpMode
	10 : i16 	PartnerKey
	11 : string 	Description
	12 : string 	SystemIdMac
	13 : i32 	LagType
	14 : string 	SystemId
	15 : i32 	Interval
	16 : bool 	Enabled
	17 : i32 	LagHash
	18 : bool 	Distributing
	19 : i32 	Timeout
	20 : i32 	Activity
	21 : i16 	SystemPriority
	22 : string 	Type
	23 : i16 	MinLinks
	24 : i64 	LacpInPkts
	25 : i64 	LacpOutPkts
	26 : i64 	LacpRxErrors
	27 : i64 	LacpTxErrors
	28 : i64 	LacpUnknownErrors
	29 : i64 	LacpErrors
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
	bool UpdateEthernetConfig(1: EthernetConfig origconfig, 2: EthernetConfig newconfig, 3: list<byte> attrset);
	bool DeleteEthernetConfig(1: EthernetConfig config);

	bool CreateAggregationLacpConfig(1: AggregationLacpConfig config);
	bool UpdateAggregationLacpConfig(1: AggregationLacpConfig origconfig, 2: AggregationLacpConfig newconfig, 3: list<byte> attrset);
	bool DeleteAggregationLacpConfig(1: AggregationLacpConfig config);

	AggregationLacpStateGetInfo GetBulkAggregationLacpState(1: int fromIndex, 2: int count);
	AggregationLacpMemberStateCountersGetInfo GetBulkAggregationLacpMemberStateCounters(1: int fromIndex, 2: int count);
}