package lua_debugger

import (
	lua "github.com/yuin/gopher-lua"
	"lua_debugger/proto"
)

const (
	Lua_HookCall = iota
	Lua_HookRet
	Lua_HookLine
	Lua_HookCount
)

const (
	LUA_TNONE = iota - 1
	LUA_TNIL
	LUA_TBOOLEAN
	LUA_TLIGHTUSERDATA
	LUA_TNUMBER
	LUA_TSTRING
	LUA_TTABLE
	LUA_TFUNCTION
	LUA_TUSERDATA
	LUA_TTHREAD
)

type Ar struct {
	lua.Debug
	Event int
}

type BreakPoint struct {
	File      string
	Condition string
	PathParts []string
	Line      int
}

type Variable struct {
	Name          string
	NameType      int
	Value         string
	ValueType     int
	ValueTypeName string
	Children      []*Variable
	CacheId       int
}

func (v *Variable) toProto() *proto.Variable {
	res := &proto.Variable{
		Name:          v.Name,
		NameType:      v.NameType,
		Value:         v.Value,
		ValueType:     v.ValueType,
		ValueTypeName: v.ValueTypeName,
	}

	for _, c := range v.Children {
		res.Children = append(res.Children, c.toProto())
	}
	return res
}

type Stack struct {
	File             string
	FunctionName     string
	Level            int
	Line             int
	LocalVariables   []*Variable
	UpvalueVariables []*Variable
}

type EvalContext struct {
	Expr       string
	Error      string
	Seq        int
	StackLevel int
	Depth      int
	CacheId    int
	Result     *Variable
	Success    bool
}
