package proto

import "reflect"

const (
	MsgIdUnknown = iota

	MsgIdInitReq
	MsgIdInitRsp

	MsgIdReadyReq
	MsgIdReadyRsq

	MsgIdAddBreakPointReq
	MsgIdAddBreakPointRsp

	MsgIdRemoveBreakPointReq
	MsgIdRemoveBreakPointRsp

	MsgIdActionReq
	MsgIdActionRsp

	MsgIdEvalReq
	MsgIdEvalRsp

	// debugger -> ide
	MsgIdBreakNotify
	MsgIdAttachedNotify

	MsgIdStartHookReq
	MsgIdStartHookRsp

	// debugger -> ide
	MsgIdLogNotify
)

type Variable struct {
	Name          string      `json:"name"`
	NameType      int         `json:"nameType"`
	Value         string      `json:"value"`
	ValueType     int         `json:"valueType"`
	ValueTypeName string      `json:"valueTypeName"`
	Children      []*Variable `json:"children"`
}

type Stack struct {
	Level            int         `json:"level"`
	File             string      `json:"file"`
	FunctionName     string      `json:"functionName"`
	Line             int         `json:"line"`
	LocalVariables   []*Variable `json:"localVariables"`
	UpvalueVariables []*Variable `json:"upvalueVariables"`
}

type BreakPoint struct {
	File      string `json:"file"`
	Line      int    `json:"line"`
	Condition string `json:"condition"`
	HitCount  int    `json:"hitCount"`
}

type InitReq struct {
	EmmyHelper string   `json:"emmyHelper"`
	Ext        []string `json:"ext"`
}

type InitRsp struct {
	Version string `json:"version"`
}

type ReadyReq struct {
}

type AddBreakPointReq struct {
	Clear       bool         `json:"clear"`
	BreakPoints []BreakPoint `json:"breakPoints"`
}

type AddBreakPointRsp struct {
}

type RemoveBreakPointReq struct {
	BreakPoints []BreakPoint `json:"breakPoints"`
}

type RemoveBreakPointRsp struct {
}

type DebugAction int

const (
	Break DebugAction = iota
	Continue
	StepOver
	StepIn
	StepOut
	Stop
)

type ActionReq struct {
	Action DebugAction `json:"action"`
}

type ActionRsp struct {
}

type BreakNotify struct {
	Cmd    int     `json:"cmd"`
	Stacks []Stack `json:"stacks"`
}

type EvalReq struct {
	Seq        int    `json:"seq"`
	Expr       string `json:"expr"`
	StackLevel int    `json:"stackLevel"`
	Depth      int    `json:"depth"`
	CacheId    int    `json:"cacheId"`
}

type EvalRsp struct {
	Seq     int       `json:"seq"`
	Success bool      `json:"success"`
	Error   string    `json:"error"`
	Value   *Variable `json:"value"`
}

var msgIdToReqMap = map[int]reflect.Type{
	MsgIdInitReq:             reflect.TypeOf(&InitReq{}),
	MsgIdReadyReq:            reflect.TypeOf(&ReadyReq{}),
	MsgIdAddBreakPointReq:    reflect.TypeOf(&AddBreakPointReq{}),
	MsgIdRemoveBreakPointReq: reflect.TypeOf(&RemoveBreakPointReq{}),
	MsgIdActionReq:           reflect.TypeOf(&ActionReq{}),
	MsgIdEvalReq:             reflect.TypeOf(&EvalReq{}),
}

func GetMsg(msgId int) interface{} {
	t := msgIdToReqMap[msgId]
	if t == nil {
		return nil
	}

	return reflect.New(t.Elem()).Interface()
}
