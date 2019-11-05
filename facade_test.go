package lua_debugger

import (
	lua "github.com/yuin/gopher-lua"
	"testing"
	"time"
)

func TestFacade_TcpConnect(t *testing.T) {
	L := lua.NewState()
	f := Facade{}
	if err:= f.TcpConnect(L, "localhost", 9966); err != nil {
		t.Fatal(err)
	}

	time.Sleep(1000 * time.Second)
}
