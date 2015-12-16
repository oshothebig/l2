COMPS=lacp

IPCS=lacp

all: ipc exe install 

exe: $(COMPS)
	 $(foreach f,$^, make -C $(f) exe;)

ipc: $(IPCS)
	 $(foreach f,$^, make -C $(f) ipc;)

clean: $(COMPS)
	$(foreach f,$^, make -C $(f) clean;)

install:
	$(foreach f,$^, make -C $(f) install;)
