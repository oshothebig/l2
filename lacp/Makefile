RM=rm -f
RMFORCE=rm -rf
GENERATED_IPC=$(SR_CODE_BASE)/generated/src
IPC_GEN_CMD=thrift 
MAIN_SRCS=lahandler.go main.go
LACP_SRCS=aggrigator.go churnDetection.go config.go debugEvent.go \
defs.go global.go lacppdu.go markerpdu.go muxmachine.go \
periodictx.go port.go rx.go rxmachine.go
IPC_SRCS=lacpd.thrift
MAIN_OBJS=$(subst .cpp,.o,$(MAIN_SRCS))
LACP_OBJS=$(subst .cpp,.o,$(LACP_SRCS))
COMP_NAME=lacpd
PKG_NAME=lacp
all:ipc $(COMP_NAME)
ipc:
	 $(IPC_GEN_CMD) --gen go -out $(GENERATED_IPC) $(IPC_SRCS)

lacpd: $(MAIN_OBJS)
	 go build -o $@ $(MAIN_SRCS)

lacp: %(LACP_OBJS)
	go build -o $@ (LACP_SRCS)

test:
	go test lacp -test.v

guard:
ifndef SR_CODE_BASE
	 $(error SR_CODE_BASE is not set)
endif

clean:guard
	 $(RM) $(COMP_NAME) 
	 $(RMFORCE) $(GENERATED_IPC)/$(COMP_NAME)