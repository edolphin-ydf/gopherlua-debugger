package lua_debugger

import (
	"github.com/edolphin-ydf/gopherlua-debugger/proto"
	lua "github.com/yuin/gopher-lua"
	"sync"
)

func LuaError(L *lua.LState, msg string) int {
	msg = "[Emmy]" + msg
	f := L.GetGlobal("error")
	L.Push(f)
	L.Push(lua.LString(msg))
	L.Call(1, 0)
	return 0
}

type Facade struct {
	t               *Transport
	m               sync.Mutex
	cond            *sync.Cond
	isWaitingForIDE bool
	isIDEReady      bool
	helperCode      string

	states map[*lua.LState]struct{}
}

var Fcd = newFacade()

func newFacade() *Facade {
	res := &Facade{
		states: make(map[*lua.LState]struct{}),
	}
	res.cond = sync.NewCond(&res.m)

	return res
}

func (f *Facade) TcpConnect(L *lua.LState, host string, port int) error {
	f.states[L] = struct{}{}
	f.t = &Transport{}
	f.t.Handler = f.HandleMsg
	if err := f.t.Connect(host, port); err != nil {
		LuaError(L, err.Error())
		return err
	}

	f.WaiteIDE(true)
	return nil
}

func (f *Facade) WaiteIDE(force bool) {
	if f.t != nil && force && !f.isWaitingForIDE && !f.isIDEReady {
		f.isWaitingForIDE = true
		f.m.Lock()
		f.cond.Wait()
		f.m.Unlock()
		f.isWaitingForIDE = false
	}
}

func (f *Facade) HandleMsg(cmd int, req interface{}) {
	switch cmd {
	case proto.MsgIdInitReq:
		f.OnInitReq(req.(*proto.InitReq))
	case proto.MsgIdReadyReq:
		f.OnReadyReq()
	case proto.MsgIdAddBreakPointReq:
		f.OnAddBreakPointReq(req.(*proto.AddBreakPointReq))
	case proto.MsgIdRemoveBreakPointReq:
		f.OnRemoveBreakPointReq(req.(*proto.RemoveBreakPointReq))
	case proto.MsgIdActionReq:
		f.OnActionReq(req.(*proto.ActionReq))
	case proto.MsgIdEvalReq:
		f.OnEvalReq(req.(*proto.EvalReq))
	}
}

func (f *Facade) OnInitReq(req *proto.InitReq) {
	f.helperCode = req.EmmyHelper
	Dbg.Start(f.helperCode)

	for state := range f.states {
		Dbg.Attach(state)
	}

	Dbg.ExtNames = req.Ext
}

func (f *Facade) OnReadyReq() {
	f.isIDEReady = true
	f.cond.Broadcast()
}

func (f *Facade) OnAddBreakPointReq(req *proto.AddBreakPointReq) {
	if req.Clear {
		Dbg.RemoveAllBreakpoints()
	}

	for _, bpProto := range req.BreakPoints {
		bp := &BreakPoint{
			File:      bpProto.File,
			Condition: bpProto.Condition,
			Line:      bpProto.Line,
		}
		Dbg.AddBreakPoint(bp)
	}
}

func (f *Facade) OnRemoveBreakPointReq(req *proto.RemoveBreakPointReq) {
	for _, bp := range req.BreakPoints {
		Dbg.RemoveBreakPoint(bp.File, bp.Line)
	}
}

func (f *Facade) OnActionReq(req *proto.ActionReq) {
	Dbg.DoAction(req.Action)
}

func (f *Facade) OnEvalReq(req *proto.EvalReq) {
	context := &EvalContext{
		Expr:       req.Expr,
		Seq:        req.Seq,
		StackLevel: req.StackLevel,
		Depth:      req.Depth,
		CacheId:    req.CacheId,
		Success:    false,
	}

	Dbg.Eval(context)
}

func (f *Facade) OnBreak(L *lua.LState) {
	stacks := Dbg.GetStacks(L)

	notify := proto.BreakNotify{Cmd: proto.MsgIdBreakNotify}
	for _, stack := range stacks {
		s := proto.Stack{
			Level:            stack.Level,
			File:             stack.File,
			FunctionName:     stack.FunctionName,
			Line:             stack.Line,
			LocalVariables:   []*proto.Variable{},
			UpvalueVariables: []*proto.Variable{},
		}
		for _, variable := range stack.LocalVariables {
			s.LocalVariables = append(s.LocalVariables, variable.toProto())
		}
		for _, variable := range stack.UpvalueVariables {
			s.LocalVariables = append(s.LocalVariables, variable.toProto())
		}
		notify.Stacks = append(notify.Stacks, s)
	}
	f.t.Send(proto.MsgIdBreakNotify, notify)
}

func (f *Facade) OnEvalResult(ctx *EvalContext) {
	rsp := proto.EvalRsp{
		Seq:     ctx.Seq,
		Success: ctx.Success,
		Error:   ctx.Error,
	}
	if ctx.Success {
		rsp.Value = ctx.Result.toProto()
	}

	f.t.Send(proto.MsgIdEvalRsp, rsp)
}
