package main

import (
	lua_debugger "github.com/edolphin-ydf/gopherlua-debugger"
	lua "github.com/yuin/gopher-lua"
	"log"
)

func main() {
	L := lua.NewState()
	lua_debugger.Preload(L)

	err := L.DoFile("main.lua")
	log.Println(err)
}
