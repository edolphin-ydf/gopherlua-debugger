package lua_debugger

import (
	"container/list"
	"github.com/edolphin-ydf/gopherlua-debugger/proto"
	lua "github.com/yuin/gopher-lua"
	"log"
	"strings"
	"sync"
)

func Hook(L *lua.LState) int {
	ar := &Ar{}

	event := L.CheckString(1)
	switch event {
	case "line":
		ar.Event = Lua_HookLine
		line := L.CheckNumber(2)
		ar.CurrentLine = int(line)
	case "count":
		ar.Event = Lua_HookCount
	case "call":
		ar.Event = Lua_HookCall
	case "return":
		ar.Event = Lua_HookRet
	}

	Dbg.Hook(L, ar)
	return 0
}

type Debugger struct {
	SkipHook     bool
	LineSet      map[int]struct{}
	BreakPoints  []*BreakPoint
	ExtNames     []string
	CurrentState *lua.LState
	HelperCode   string
	States       map[*lua.LState]struct{}
	HookState    HookStateInter

	stateBreak    HookStateInter
	stateStepOver HookStateInter
	stateStepIn   HookStateInter
	stateStepOut  HookStateInter
	stateContinue HookStateInter
	stateStop     HookStateInter

	mutexBP   sync.Mutex
	mutexRun  sync.Mutex
	condRun   *sync.Cond
	blocking  bool
	mutexEval sync.Mutex
	evalQueue list.List
	running   bool
}

var Dbg = newDebugger()

func newDebugger() *Debugger {
	res := &Debugger{}
	res.LineSet = make(map[int]struct{})
	res.States = make(map[*lua.LState]struct{})
	res.condRun = sync.NewCond(&res.mutexRun)
	res.stateBreak = &HookStateBreak{}
	res.stateStepOver = &HookStateStepOver{}
	res.stateStepIn = &HookStateStepIn{}
	res.stateStepOut = &HookStateStepOut{}
	res.stateContinue = &HookStateContinue{}
	res.stateStop = &HookStateStop{}
	return res
}

func (d *Debugger) Start(code string) {
	d.HelperCode = code
	d.SkipHook = false
	d.blocking = false
	d.running = true
}

func (d *Debugger) Attach(L *lua.LState) {
	if !d.running {
		return
	}

	d.States[L] = struct{}{}

	if d.HelperCode != "" {
		t := L.GetTop()
		err := L.DoString(d.HelperCode)
		if err != nil {
			log.Fatal("do helper code fail:", err)
		}
		L.SetTop(t)
	}
	d.UpdateHook(L, "clr")
}

func (d *Debugger) DoAction(action proto.DebugAction) {
	L := d.CurrentState
	switch action {
	case proto.Break:
		d.SetHookState(L, d.stateBreak)
	case proto.Continue:
		d.SetHookState(L, d.stateContinue)
	case proto.StepOver:
		d.SetHookState(L, d.stateStepOver)
	case proto.StepIn:
		d.SetHookState(L, d.stateStepIn)
	case proto.StepOut:
		d.SetHookState(L, d.stateStepOut)
	case proto.Stop:
		d.SetHookState(L, d.stateStop)
	}
}

func (d *Debugger) SetHookState(L *lua.LState, newState HookStateInter) {
	d.HookState = nil
	if newState.Start(d, L) {
		d.HookState = newState
	}
}

func (d *Debugger) GetStackLevel(L *lua.LState, skipGo bool) int {
	level := 0
	i := 0
	ar, ok := L.GetStack(i)
	for ok {
		_, err := L.GetInfo("l", ar, nil)
		if err != nil {
			log.Println("get info fail", err)
			return level
		}

		if ar.CurrentLine >= 0 || !skipGo {
			level++
		}
		i++
		ar, ok = L.GetStack(i)
	}
	return level
}

func (d *Debugger) UpdateHook(L *lua.LState, mask string) {
	if mask == "" {
		_ = L.SetHook(L.NewFunction(Hook), mask, 0)
		return
	}
	_ = L.SetHook(L.NewFunction(Hook), mask, 0)
}

func (d *Debugger) Hook(L *lua.LState, ar *Ar) {
	if d.SkipHook {
		return
	}
	if ar.Event == Lua_HookLine {
		ar2, _ := L.GetStack(1)
		ar2.CurrentLine = ar.CurrentLine
		ar.Debug = *ar2
		bp := d.FindBreakPoint(L, ar)
		if bp != nil {
			d.HandleBreak(L)
			return
		}
		if d.HookState != nil {
			d.HookState.ProcessHook(d, L, ar)
		}
	}
}

func GoLuaTypeToCLuaType(t lua.LValueType) (int, string) {

	switch t {
	case lua.LTNil:
		return LUA_TNIL, "nil"
	case lua.LTBool:
		return LUA_TBOOLEAN, "boolean"
	case lua.LTNumber:
		return LUA_TNUMBER, "number"
	case lua.LTString:
		return LUA_TSTRING, "string"
	case lua.LTFunction:
		return LUA_TFUNCTION, "function"
	case lua.LTUserData:
		return LUA_TUSERDATA, "userdata"
	case lua.LTThread:
		return LUA_TTHREAD, "thread"
	case lua.LTTable:
		return LUA_TTABLE, "table"
	case lua.LTChannel:
		return LUA_TUSERDATA, "userdata"
	default:
		return LUA_TNONE, "no value"
	}
}

func (d *Debugger) GetVariable(name string, value lua.LValue, depth int) *Variable {
	variable := &Variable{
		Name:          name,
		NameType:      LUA_TSTRING,
		ValueTypeName: value.Type().String(),
		Children:      nil,
		CacheId:       0,
	}
	variable.ValueType, variable.ValueTypeName = GoLuaTypeToCLuaType(value.Type())

	switch conv := value.(type) {
	case *lua.LTable:
		if depth > 0 {
			conv.ForEach(func(key lua.LValue, value lua.LValue) {
				varia := d.GetVariable(key.String(), value, depth-1)
				varia.NameType, _ = GoLuaTypeToCLuaType(key.Type())
				variable.Children = append(variable.Children, varia)
			})
		}
	default:
		variable.Value = value.String()
	}
	return variable
}

func (d *Debugger) GetStacks(L *lua.LState) []*Stack {
	var stacks []*Stack
	level := 0
	for {
		ar, ok := L.GetStack(level)
		if !ok {
			break
		}
		if _, err := L.GetInfo("nSlu", ar, nil); err != nil {
			log.Println("Debugger:GetStacks get info fail:", err)
			break
		}

		stack := &Stack{}
		stack.File = d.GetFile(L, &Ar{Debug: *ar})
		stack.FunctionName = ar.Name
		stack.Level = level
		stack.Line = ar.CurrentLine
		stacks = append(stacks, stack)

		for i := 1; ; i++ {
			name, value := L.GetLocal(ar, i)
			if name == "" {
				break
			}
			if name[0] == '(' {
				continue
			}

			variable := d.GetVariable(name, value, 1)
			stack.LocalVariables = append(stack.LocalVariables, variable)
		}

		if f, _ := L.GetInfo("f", ar, nil); f != lua.LNil {
			for i := 1; ; i++ {
				name, value := L.GetUpvalue(f.(*lua.LFunction), i)
				if name == "" {
					break
				}

				variable := d.GetVariable(name, value, 1)
				stack.UpvalueVariables = append(stack.UpvalueVariables, variable)
			}
		}
		level++
	}
	return stacks
}

func (d *Debugger) HandleBreak(L *lua.LState) {
	d.CurrentState = L
	d.UpdateHook(L, "l") // TODO
	Fcd.OnBreak(L)
	d.EnterDebugMode(L)
}

func (d *Debugger) EnterDebugMode(L *lua.LState) {
	d.mutexRun.Lock()
	defer d.mutexRun.Unlock()

	d.blocking = true
	for {
		d.mutexEval.Lock()
		if d.evalQueue.Len() == 0 && d.blocking {
			d.mutexEval.Unlock()
			d.condRun.Wait()
			d.mutexEval.Lock()
		}

		if d.evalQueue.Len() > 0 {
			evalContextNode := d.evalQueue.Front()
			d.evalQueue.Remove(evalContextNode)
			d.mutexEval.Unlock()

			skip := d.SkipHook
			d.SkipHook = true

			evalContext, _ := evalContextNode.Value.(*EvalContext)
			evalContext.Success = d.DoEval(evalContext)
			d.SkipHook = skip
			Fcd.OnEvalResult(evalContext)
			continue
		}
		d.mutexEval.Unlock()
		break
	}
}

func (d *Debugger) ExitDebugMode() {
	d.blocking = false
	d.condRun.Broadcast()
}

func (d *Debugger) Eval(ctx *EvalContext) {
	if !d.blocking {
		return
	}

	d.mutexEval.Lock()
	d.evalQueue.PushBack(ctx)
	d.mutexEval.Unlock()
	d.condRun.Broadcast()
}

func (d *Debugger) DoEval(evalContext *EvalContext) bool {
	L := d.CurrentState
	statement := "return " + evalContext.Expr
	f, err := L.LoadString(statement)
	if err != nil {
		log.Println("Debugger:DoEval loadstring fail:", err)
		evalContext.Error = err.Error()
		return false
	}

	env, ok := d.CreateEnv(evalContext.StackLevel)
	if !ok {
		log.Println("Debugger:DoEval create env fail")
		return false
	}
	L.SetFEnv(f, env)

	L.Push(f)
	if err := L.PCall(0, 1, nil); err != nil {
		log.Println("Debugger:DoEval call fail", err)
		evalContext.Error = err.Error()
		return false
	}

	evalContext.Result = d.GetVariable(evalContext.Expr, L.Get(L.GetTop()), evalContext.Depth)
	return true
}

func EnvIndexFunction(L *lua.LState) int {
	localsIdx := lua.UpvalueIndex(1)
	upvaluesIdx := lua.UpvalueIndex(2)
	name := L.CheckString(2)
	upvalues := L.Get(upvaluesIdx)

	v := L.RawGet(upvalues.(*lua.LTable), lua.LString(name))
	if v != lua.LNil {
		L.Push(v)
		return 1
	}

	locals := L.Get(localsIdx)
	v = L.RawGet(locals.(*lua.LTable), lua.LString(name))
	if v != lua.LNil {
		L.Push(v)
		return 1
	}

	v = L.GetGlobal(name)
	if v != lua.LNil {
		L.Push(v)
		return 1
	}
	return 0
}

func (d *Debugger) CreateEnv(stackLevel int) (*lua.LTable, bool) {
	L := d.CurrentState
	ar, ok := L.GetStack(stackLevel)
	if !ok {
		return nil, false
	}
	if _, err := L.GetInfo("nSlu", ar, nil); err != nil {
		log.Println(err)
		return nil, false
	}

	env := L.NewTable()
	envMetatable := L.NewTable()
	locals := L.NewTable()
	upvalues := L.NewTable()

	for i := 1; ; i++ {
		name, value := L.GetLocal(ar, i)
		if name == "" {
			break
		}
		if name[0] == '(' {
			continue
		}
		locals.RawSetString(name, value)
	}

	if f, _ := L.GetInfo("f", ar, nil); f != nil {
		for i := 1; ; i++ {
			name, value := L.GetUpvalue(f.(*lua.LFunction), i)
			if name == "" {
				break
			}
			upvalues.RawSetString(name, value)
		}
	}

	cl := L.NewClosure(EnvIndexFunction, locals, upvalues)
	envMetatable.RawSetString("__index", cl)
	L.SetMetatable(env, envMetatable)

	return env, true
}

func FixPath(L *lua.LState) int {
	path := L.ToString(1)
	emmy := L.GetGlobal("emmy")
	if _, ok := emmy.(*lua.LTable); ok {
		f := L.GetField(emmy, "fixPath")
		L.Push(f)
		L.Push(lua.LString(path))
		L.Call(1, 1)
		return 1
	}
	return 0
}

func (d *Debugger) GetFile(L *lua.LState, ar *Ar) string {
	file := ar.Source
	if ar.CurrentLine < 0 {
		return file
	}

	L.Push(L.NewFunction(FixPath))
	L.Push(lua.LString(file))
	err := L.PCall(1, 1, nil)
	if err == nil {
		p := L.ToString(-1)
		L.Pop(1)
		if p != "" {
			return p
		}
	}
	return file
}

func ParsePathParts(file string, paths []string) []string {
	idx := 0
	for i, c := range file {
		if c == '/' || c == '\\' {
			part := file[idx:i]
			idx = i + 1

			// ../a/b/c
			if part == ".." {
				if len(paths) > 0 {
					paths = paths[:len(paths)-1]
					continue
				}
			}

			// ./a/b/c
			if (part == "." || len(part) == 0) && len(paths) == 0 {
				continue
			}
			paths = append(paths, part)
		}
	}

	paths = append(paths, file[idx:])
	return paths
}

func (d *Debugger) MatchFileName(chunkName string, fileName string) bool {
	if chunkName == fileName {
		return true
	}

	for _, ext := range d.ExtNames {
		if chunkName+ext == fileName {
			return true
		}
	}
	return false
}

func (d *Debugger) FindBreakPoint(L *lua.LState, ar *Ar) *BreakPoint {
	if ar.CurrentLine >= 0 {
		_, lineExist := d.LineSet[ar.CurrentLine]
		if lineExist {
			_, err := L.GetInfo("S", &ar.Debug, nil)
			if err != nil {
				log.Println("find break point fail:", err)
				return nil
			}

			file := d.GetFile(L, ar)
			return d.FindBreakPointByFile(file, ar.CurrentLine)
		}
	}
	return nil
}

func (d *Debugger) FindBreakPointByFile(file string, line int) *BreakPoint {
	d.mutexBP.Lock()
	defer d.mutexBP.Unlock()

	var pathParts []string
	lowerCaseFile := strings.ToLower(file)
	pathParts = ParsePathParts(lowerCaseFile, pathParts)

	for _, bp := range d.BreakPoints {
		if bp.Line == line {
			if bp.File == lowerCaseFile {
				return bp
			}

			if len(bp.PathParts) >= len(pathParts) && d.MatchFileName(pathParts[len(pathParts)-1], bp.PathParts[len(bp.PathParts)-1]) {
				match := true
				for i := 0; i < len(pathParts); i++ {
					p := bp.PathParts[len(bp.PathParts)-1-i]
					f := pathParts[len(pathParts)-1-i]
					if p != f {
						match = false
						break
					}
				}
				if match {
					return bp
				}
			}
		}
	}
	return nil
}

func (d *Debugger) RemoveBreakPoint(file string, line int) {
	lowerCaseFile := strings.ToLower(file)
	d.mutexBP.Lock()
	defer d.mutexBP.Unlock()

	for i, bp := range d.BreakPoints {
		if bp.File == lowerCaseFile && bp.Line == line {
			d.BreakPoints = append(d.BreakPoints[:i], d.BreakPoints[i+1:]...)
			break
		}
	}
	d.RefreshLineSet()
}

func (d *Debugger) RemoveAllBreakpoints() {
	d.LineSet = make(map[int]struct{})
	d.BreakPoints = []*BreakPoint{}
}

func (d *Debugger) AddBreakPoint(bp *BreakPoint) {
	d.mutexBP.Lock()
	defer d.mutexBP.Unlock()

	bp.File = strings.ToLower(bp.File)
	bp.PathParts = ParsePathParts(bp.File, bp.PathParts)
	d.BreakPoints = append(d.BreakPoints, bp)

	d.RefreshLineSet()
}

func (d *Debugger) RefreshLineSet() {
	d.LineSet = make(map[int]struct{})

	for _, bp := range d.BreakPoints {
		d.LineSet[bp.Line] = struct{}{}
	}
}
