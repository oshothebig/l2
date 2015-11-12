namespace go lacpd
typedef i32 int
//typedef byte uint8
typedef i16 uint16
service LaService 
{
    int CreateLaAggregator(1: int aggId, 2: uint16 actorAdminKey, 3: string actorSystemId, 4: string aggMacAddr);
    int DeleteLaAggregator(1: int aggId);
	int CreateLaAggPort(1: uint16 Id , 2: uint16 Prio, 3: uint16 Key, 4: int AggId, 5: bool Enable, 6: string Mode, 
	7: string Timeout, 8: string Mac, 9: int Speed, 10: int Duplex, 11: int Mtu, 12: string SysId, 13: string IntfId);
	int DeleteLaAggPort(1: uint16 Id);
	int SetLaAggPortMode(1: uint16 Id, 2: string m);
	int SetLaAggPortTimeout(1: uint16 Id, 2: string t);
	int SetPortLacpLogEnable(1: uint16 Id, 2: string modStr, 3: bool ena);
}