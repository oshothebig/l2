namespace go lldpd
typedef i32 int
typedef i16 uint16
struct LLDPIntf {
	1 : i32 IfIndex
	2 : bool Enable
}
service LLDPDServices {
	bool CreateLLDPIntf(1: LLDPIntf config);
	bool UpdateLLDPIntf(1: LLDPIntf origconfig, 2: LLDPIntf newconfig, 3: list<bool> attrset);
	bool DeleteLLDPIntf(1: LLDPIntf config);

}