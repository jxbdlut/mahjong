package internal

import (
	"reflect"

	"github.com/jxbdlut/leaf/gate"
	"github.com/jxbdlut/leaf/log"
	"net"
	"server/proto"
	"server/userdata"
	"server/game/area_manager"
)

const (
	MinTableId uint32 = 10000
	MaxTableId uint32 = 100000
	MinRobotId uint64 = 100000
	MaxRobotId uint64 = 1000000
)

var (
	Tables       map[uint32]*Table
	robots       map[uint64]*gate.Agent
	curTableId   uint32 = 10000
	curRobotId   uint64 = 100000
	MapUidPlayer map[uint64]*Player
)

func init() {
	handler(&proto.CreateTableReq{}, handlerCreateTable)
	handler(&proto.JoinTableReq{}, handlerJoinTable)
	handler(&proto.OperatRsp{}, handlerOperatRsp)
	handler(&proto.TableOperatRsp{}, handlerTableOperatRsp)
	Tables = make(map[uint32]*Table)
	robots = make(map[uint64]*gate.Agent)
	MapUidPlayer = make(map[uint64]*Player)
}

func handler(m interface{}, h interface{}) {
	skeleton.RegisterChanRPC(reflect.TypeOf(m), h)
}

func checkLogin(a gate.Agent) bool {
	if a.UserData() == nil || 0 == a.UserData().(*userdata.UserData).Uid {
		return false
	} else {
		return true
	}
}

func handlerCreateTable(args []interface{}) {
	req := args[0].(*proto.CreateTableReq)
	a := args[1].(gate.Agent)
	seq := args[2].(uint32)
	if !checkLogin(a) {
		log.Error("no login!")
		a.Replay(&proto.CreateTableRsp{
			ErrCode: -1,
			ErrMsg:  "no login!",
		}, seq)
		return
	}
	uid := a.UserData().(*userdata.UserData).Uid
	tid := genTableId()
	log.Debug("uid:%v, create table, tid:%v, seq:%v", uid, tid, seq)
	table := NewTable(tid, proto.CreateTableReq_TableType(req.Type))
	table.rule = area_manager.GetArea(uint16(req.Area))
	log.Debug("tid:%v, rule:%v", tid, reflect.TypeOf(table.rule))
	a.SetUserData(&userdata.UserData{
		Uid: uid,
		Tid: tid,
	})
	table.AddAgent(a, true)
	if proto.CreateTableReq_TableType(req.Type) == proto.CreateTableReq_TableRobot {
		for i := 1; i < 4; i++ {
			rid := genRobotUid()
			agent := NewAgent(rid)
			robots[rid] = &agent
			table.AddAgent(agent, false)
		}
	}
	go table.Run()
	Tables[tid] = table
	a.Replay(&proto.CreateTableRsp{
		ErrCode: 0,
		ErrMsg:  "CreateTable success!",
		TableId: tid,
	}, seq)
}

func handlerJoinTable(args []interface{}) {
	req := args[0].(*proto.JoinTableReq)
	a := args[1].(gate.Agent)
	seq := args[2].(uint32)
	rsp := proto.JoinTableRsp{}
	if !checkLogin(a) {
		log.Error("no login!")
		a.Replay(&proto.CreateTableRsp{
			ErrCode: -1,
			ErrMsg:  "no login!",
		}, seq)
		return
	}
	uid := a.UserData().(*userdata.UserData).Uid
	tid := req.TableId
	log.Debug("uid:%v, join table, tid:%v, seq:%v", uid, tid, seq)
	if table, ok := Tables[tid]; ok {
		if _, err := table.GetPlayerIndex(uid); err == nil {
			rsp.ErrCode = 10000
			rsp.ErrMsg = "you areadly in table"
		} else {
			a.SetUserData(&userdata.UserData{
				Uid: uid,
				Tid: tid,
			})
			if pos, err := Tables[tid].AddAgent(a, false); err != nil {
				rsp.ErrCode = -1
				rsp.ErrMsg = err.Error()
			} else {
				rsp.ErrCode = 0
				rsp.ErrMsg = "join success!"
				rsp.Pos = int32(pos)
				joinTableMsg := proto.UserJoinTableMsg{Tid:tid}
				for i, player := range Tables[tid].players {
					seat := &proto.Seat{Uid:player.uid, Name:player.name, Pos:int32(i + 1)}
					joinTableMsg.Seats = append(joinTableMsg.Seats, seat)
				}
				table.Broadcast(&joinTableMsg)
			}
		}
	} else {
		log.Error("table is not exist, tid:%v", tid)
		rsp.ErrCode = -1
		rsp.ErrMsg = "table is not exist"
	}

	a.Replay(&rsp, seq)
}

func handlerTableOperatRsp(args []interface{}) {
	rsp := args[0]
	a := args[1].(gate.Agent)
	tid := a.UserData().(*userdata.UserData).Tid
	uid := a.UserData().(*userdata.UserData).Uid
	table := Tables[tid]
	if player, err := table.GetPlayer(uid); err == nil {
		player.HandlerTableOperatRsp(rsp)
	}
}

func handlerOperatRsp(args []interface{}) {
	rsp := args[0]
	a := args[1].(gate.Agent)
	tid := a.UserData().(*userdata.UserData).Tid
	uid := a.UserData().(*userdata.UserData).Uid
	table := Tables[tid]
	if player, err := table.GetPlayer(uid); err == nil {
		player.HandlerOperatRsp(rsp)
	}
}

func genTableId() uint32 {
	for {
		if _, ok := Tables[curTableId]; ok {
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
	userData interface{}
}

func NewAgent(uid uint64) gate.Agent {
	a := &agent{}
	a.SetUserData(&userdata.UserData{
		Uid: uid,
	})
	return a
}

func (a *agent) Replay(msg interface{}, seq uint32) {

}

func (a *agent) Send(msg interface{}) {

}

func (a *agent) SendRcv(msg interface{}) (interface{}, error) {
	// todo
	return nil, nil
}

func (a *agent) WriteMsg(msg interface{}, cbChan chan interface{}, seq uint32) {
	//log.Debug("uid:%v writemsg", a.UserData().(*userdata.UserData).Uid)
	return
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
