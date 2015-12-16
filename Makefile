COMPS=lacp
IPCS=lacp
all: ipc exe install

exe: $(COMPS)
	 @echo "ignoring $^"
	 @#$(foreach f,$^, make -C $(f) exe;)

ipc: $(IPCS)
	 @echo "ignoring $^"
	 @#$(foreach f,$^, make -C $(f) ipc;)

clean: $(COMPS)
	 @echo "ignoring $^"
	 @#$(foreach f,$^, make -C $(f) clean;)

install: $(COMPS)
	 @echo "ignoring $^"
	 @#$(foreach f,$^, make -C $(f) install;)

