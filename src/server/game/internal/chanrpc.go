package internal

import (
	"github.com/name5566/leaf/gate"
	"github.com/name5566/leaf/log"
	"server/userdata"
)

func init() {
	skeleton.RegisterChanRPC("NewAgent", rpcNewAgent)
	skeleton.RegisterChanRPC("CloseAgent", rpcCloseAgent)
}

func rpcNewAgent(args []interface{}) {
	a := args[0].(gate.Agent)
	_ = a
}

func rpcCloseAgent(args []interface{}) {
	if args == nil {
		return
	}
	a := args[0].(gate.Agent)
	if a == nil {
		return
	}
	uid := a.UserData().(*userdata.UserData).Uid
	tid := a.UserData().(*userdata.UserData).Tid
	if table, ok := tables[tid]; ok {
		table.removeAgent(a)
		a.Destroy()
	}
	log.Debug("close Agent Uid: %v, tid:%v", uid, tid)
}
