package internal

import (
	"reflect"

	"github.com/name5566/leaf/gate"
	"github.com/name5566/leaf/log"
	"server/proto"
	"server/userdata"
	"net"
)

const (
	MinTableId uint32 = 10000
	MaxTableId uint32 = 100000
	MinRobotId uint64 = 100000
	MaxRobotId uint64 = 1000000
)

var (
	tables       map[uint32]*Table
	robots       map[uint64]*gate.Agent
	curTableId   uint32 = 10000
	curRobotId   uint64 = 100000
	MapUidPlayer map[uint64]*Player
)

func init() {
	handler(&proto.CreateTableReq{}, handlerCreateTable)
	handler(&proto.JoinTableReq{}, handlerJoinTable)
	handler(&proto.OperatRsp{}, handlerOperatRsp)
	tables = make(map[uint32]*Table)
	robots = make(map[uint64]*gate.Agent)
	MapUidPlayer = make(map[uint64]*Player)
}

func handler(m interface{}, h interface{}) {
	skeleton.RegisterChanRPC(reflect.TypeOf(m), h)
}

func checkLogin(a gate.Agent) bool {
	if 	a.UserData() == nil || 0 == a.UserData().(*userdata.UserData).Uid {
		return false
	} else {
		return true
	}
}

func handlerCreateTable(args []interface{}) {
	req := args[0].(*proto.CreateTableReq)
	a := args[1].(gate.Agent)
	if !checkLogin(a) {
		log.Error("no login!")
		a.WriteMsg(&proto.CreateTableRsp{
			ErrCode: -1,
			ErrMsg:  "no login!",
		})
		return
	}
	uid := a.UserData().(*userdata.UserData).Uid
	tid := genTableId()
	table := NewTable(tid, proto.CreateTableReq_TableType(req.Type))
	a.SetUserData(&userdata.UserData{
		Uid: uid,
		Tid: tid,
	})
	table.AddAgent(a, true)
	if proto.CreateTableReq_TableType(req.Type) == proto.CreateTableReq_TableRobot {
		for i := 0; i < 3; i++ {
			rid := genRobotUid()
			agent := NewAgent(rid)
			robots[rid] = &agent
			table.AddAgent(agent, false)
		}
	}
	go table.Run()
	tables[tid] = table
	log.Debug("uid:%v, create table", uid)
	a.WriteMsg(&proto.CreateTableRsp{
		ErrCode: 0,
		ErrMsg:  "successed!",
		TableId: tid,
	})
}

func handlerJoinTable(args []interface{}) {
	req := args[0].(*proto.JoinTableReq)
	a := args[1].(gate.Agent)
	rsp := proto.JoinTableRsp{}
	if !checkLogin(a) {
		log.Error("no login!")
		a.WriteMsg(&proto.CreateTableRsp{
			ErrCode: -1,
			ErrMsg:  "no login!",
		})
		return
	}
	uid := a.UserData().(*userdata.UserData).Uid
	tid := req.TableId
	if table, ok := tables[tid]; ok {
		if _, ok := table.GetPlayerIndex(uid); ok == nil {
			rsp.ErrCode = 10000
			rsp.ErrMsg = "you areadly in table"
		} else {
			a.SetUserData(&userdata.UserData{
				Uid: uid,
				Tid: tid,
			})
			tables[tid].AddAgent(a, false)
			rsp.ErrCode = 0
			rsp.ErrMsg = "join successed!"
			table.BroadcastExceptMe(&proto.UserJoinTableMsg{
				Uid: uid,
				Tid: tid,
			}, uid)
		}
	} else {
		log.Error("table is not exist, tid:%v", tid)
		rsp.ErrCode = -1
		rsp.ErrMsg = "table is not exist"
	}

	a.WriteMsg(&rsp)
}

func handlerOperatRsp(args []interface{}) {
	rsp := args[0]
	a := args[1].(gate.Agent)
	tid := a.UserData().(*userdata.UserData).Tid
	uid := a.UserData().(*userdata.UserData).Uid
	table := tables[tid]
	if player, err := table.GetPlayer(uid); err == nil {
		player.HandlerOperatRsp(rsp)
	}
}

func genTableId() uint32 {
	for {
		if _, ok := tables[curTableId]; ok {
			curTableId++
			if curTableId > MaxTableId {
				curTableId = MinTableId
			}
		} else {
			return curTableId
		}
	}
}

func genRobotUid() uint64 {
	for {
		if _, ok := robots[curRobotId]; ok {
			curRobotId++
			if curRobotId > MaxRobotId {
				curRobotId = MinRobotId
			}
		} else {
			return curRobotId
		}
	}
}

type agent struct {
	userData        interface{}
}

func NewAgent(uid uint64) gate.Agent {
	a := &agent{}
	a.SetUserData(&userdata.UserData{
		Uid: uid,
	})
	return a
}

func (a *agent) WriteMsg(msg interface{}) {
	log.Debug("uid:%v writemsg", a.UserData().(*userdata.UserData).Uid)
}

func (a *agent) UserData() interface{} {
	return a.userData
}

func (a *agent) SetUserData(data interface{}) {
	a.userData = data
}

func (a *agent) LocalAddr() net.Addr {
	return nil
}

func (a *agent) RemoteAddr() net.Addr {
	return nil
}

func (a *agent) Close() {

}

func (a *agent) Destroy() {

}