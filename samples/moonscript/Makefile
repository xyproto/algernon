.PHONY: all clean serve

MOONFILES=$(wildcard *.moon)
LUAFILES=$(MOONFILES:.moon=.lua)

all: hello clean

%.lua: %.moon
	@moonc -o "$@" "$<"

hello: hello.lua
	@algernon -o -s -z --theme=dracula hello.lua

index: index.lua
	@algernon -o -e .

clean:
	@rm -f $(LUAFILES)
