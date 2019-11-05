package lua_debugger

import "testing"

func TestTransport_Connect(t *testing.T) {
	trans := Transport{}
	if err := trans.Connect("localhost", 9966); err != nil {
		t.Fatal(err)
	}
}
