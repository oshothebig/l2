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
struct AggregationConfig{
	1 : string 	NameKey
	2 : bool 	Enabled
	3 : string 	Description
	4 : string 	Type
	5 : i16 	Mtu
	6 : i32 	LagType
	7 : i16 	MinLinks
}
struct AggregationLacpConfig{
	1 : i32 	LagType
	2 : string 	Description
	3 : bool 	Enabled
	4 : i16 	Mtu
	5 : i16 	MinLinks
	6 : string 	Type
	7 : string 	NameKey
	8 : i32 	Interval
	9 : i32 	LacpMode
	10 : string 	SystemIdMac
	11 : i16 	SystemPriority
}
service LACPDServer {
	bool CreateEthernetConfig(1: EthernetConfig config);
	bool UpdateEthernetConfig(1: EthernetConfig config);
	bool DeleteEthernetConfig(1: EthernetConfig config);

	bool CreateAggregationConfig(1: AggregationConfig config);
	bool UpdateAggregationConfig(1: AggregationConfig config);
	bool DeleteAggregationConfig(1: AggregationConfig config);

	bool CreateAggregationLacpConfig(1: AggregationLacpConfig config);
	bool UpdateAggregationLacpConfig(1: AggregationLacpConfig config);
	bool DeleteAggregationLacpConfig(1: AggregationLacpConfig config);

}