package main

import (
	lua "github.com/yuin/gopher-lua"
	"log"
	"lua_debugger"
)

func main() {
	L := lua.NewState()
	lua_debugger.Preload(L)

	err := L.DoFile("main.lua")
	log.Println(err)
}
