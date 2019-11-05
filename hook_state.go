package lua_debugger

import (
	"github.com/edolphin-ydf/gopherlua-debugger/proto"
	lua "github.com/yuin/gopher-lua"
	"log"
)

type HookStateInter interface {
	Start(debugger *Debugger, current *lua.LState) bool
	ProcessHook(debugger *Debugger, L *lua.LState, ar *Ar)
}

type HookState struct {
	currentL *lua.LState
}

func (h *HookState) Start(debugger *Debugger, current *lua.LState) bool {
	h.currentL = current
	return true
}

func (h *HookState) ProcessHook(debugger *Debugger, L *lua.LState, ar *Ar) {
}

type HookStateContinue struct {
	HookState
}

func (h *HookStateContinue) Start(debugger *Debugger, current *lua.LState) bool {
	debugger.ExitDebugMode()
	return true
}

type StackLevelBasedState struct {
	HookState
	oriStackLevel int
	newStackLevel int
}

func (h *StackLevelBasedState) Start(debugger *Debugger, current *lua.LState) bool {
	if current == nil {
		return false
	}
	h.currentL = current
	h.oriStackLevel = debugger.GetStackLevel(current, false)
	h.newStackLevel = h.oriStackLevel
	return true
}

func (h *StackLevelBasedState) UpdateStackLevel(debugger *Debugger, L *lua.LState, ar *Ar) {
	if L != h.currentL {
		return
	}

	for i := h.newStackLevel + 1; i >= 0; i-- {
		_, ok := L.GetStack(i)
		if ok {
			h.newStackLevel = i + 1
			break
		}
	}
}

type HookStateStepIn struct {
	StackLevelBasedState
	file string
	line int
}

func (h *HookStateStepIn) Start(debugger *Debugger, current *lua.LState) bool {
	if !h.StackLevelBasedState.Start(debugger, current) {
		return false
	}

	ar, ok := current.GetStack(0)
	if !ok {
		return false
	}

	_, err := current.GetInfo("nSl", ar, nil)
	if err != nil {
		log.Println("StepIn:Start, get info fail:", err)
		return false
	}

	h.file = ar.Source
	h.line = ar.CurrentLine
	debugger.ExitDebugMode()
	return true
}

func (h *HookStateStepIn) ProcessHook(debugger *Debugger, L *lua.LState, ar *Ar) {
	h.UpdateStackLevel(debugger, L, ar)
	if ar.Event == Lua_HookLine && ar.CurrentLine != h.line {
		debugger.HandleBreak(L)
	} else {
		h.StackLevelBasedState.ProcessHook(debugger, L, ar)
	}
}

type HookStateStepOut struct {
	StackLevelBasedState
}

func (h *HookStateStepOut) Start(debugger *Debugger, current *lua.LState) bool {
	if !h.StackLevelBasedState.Start(debugger, current) {
		return false
	}

	debugger.ExitDebugMode()
	return true
}

func (h *HookStateStepOut) ProcessHook(debugger *Debugger, L *lua.LState, ar *Ar) {
	h.UpdateStackLevel(debugger, L, ar)
	if h.newStackLevel < h.oriStackLevel {
		debugger.HandleBreak(L)
	} else {
		h.StackLevelBasedState.ProcessHook(debugger, L, ar)
	}
}

type HookStateStepOver struct {
	StackLevelBasedState
	file string
	line int
}

func (h *HookStateStepOver) Start(debugger *Debugger, current *lua.LState) bool {
	if !h.StackLevelBasedState.Start(debugger, current) {
		return false
	}
	ar, ok := current.GetStack(0)
	if !ok {
		log.Println("HookStateStepOver:Start, get stack fail")
		return false
	}
	_, err := current.GetInfo("nSl", ar, nil)
	if err != nil {
		log.Println("HookStateStepOver:Start, get info fail:", err)
	}

	h.file = ar.Source
	h.line = ar.CurrentLine
	debugger.ExitDebugMode()
	return true
}

func (h *HookStateStepOver) ProcessHook(debugger *Debugger, L *lua.LState, ar *Ar) {
	h.UpdateStackLevel(debugger, L, ar)

	if h.newStackLevel < h.oriStackLevel {
		debugger.HandleBreak(L)
		return
	}

	if ar.Event == Lua_HookLine && ar.CurrentLine != h.line && h.newStackLevel == h.oriStackLevel {
		_, err := L.GetInfo("Sl", &ar.Debug, nil)
		if err != nil {
			log.Fatal("HookStateStepOver:ProcessHook, getinfo fail", err)
		}
		if ar.Source == h.file || h.line == -1 {
			debugger.HandleBreak(L)
			return
		}
	}

	h.StackLevelBasedState.ProcessHook(debugger, L, ar)
}

type HookStateBreak struct {
	HookState
}

func (h *HookStateBreak) ProcessHook(debugger *Debugger, L *lua.LState, ar *Ar) {
	if ar.Event == Lua_HookLine {
		debugger.HandleBreak(L)
	} else {
		h.HookState.ProcessHook(debugger, L, ar)
	}
}

type HookStateStop struct {
	HookState
}

func (h *HookStateStop) Start(debugger *Debugger, current *lua.LState) bool {
	if current == nil {
		return false
	}
	debugger.UpdateHook(current, "")
	debugger.DoAction(proto.Continue)
	return true
}
