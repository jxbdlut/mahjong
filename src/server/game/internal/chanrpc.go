package internal

import (
	"github.com/jxbdlut/leaf/gate"
	"github.com/jxbdlut/leaf/log"
	"server/userdata"
	"server/game/area_manager"
)

func init() {
	skeleton.RegisterChanRPC("NewRobot", rpcNewAgent)
	skeleton.RegisterChanRPC("CloseAgent", rpcCloseAgent)
	area_manager.Init()
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
	if a.UserData() == nil {
		return
	}
	uid := a.UserData().(*userdata.UserData).Uid
	tid := a.UserData().(*userdata.UserData).Tid
	if table, ok := Tables[tid]; ok {
		//table.RemoveAgent(a)
		//a.Destroy()
		table.OfflineAgent(a)
	}
	log.Debug("close agent uid: %v, tid:%v", uid, tid)
}
