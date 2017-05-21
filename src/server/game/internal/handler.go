package internal

import (
	"reflect"

	"github.com/name5566/leaf/gate"
	"github.com/name5566/leaf/log"
	"server/proto"
	"server/userdata"
)

var (
	tables       map[uint32]*Table
	curTableId   uint32 = 10000
	MapUidPlayer map[uint64]*Player
)

func init() {
	handler(&proto.CreateTableReq{}, handlerCreateTable)
	handler(&proto.JoinTableReq{}, handlerJoinTable)
	handler(&proto.OperatRsp{}, handlerOperatRsp)
	tables = make(map[uint32]*Table)
	MapUidPlayer = make(map[uint64]*Player)
}

func handler(m interface{}, h interface{}) {
	skeleton.RegisterChanRPC(reflect.TypeOf(m), h)
}

func handlerCreateTable(args []interface{}) {
	req := args[0].(*proto.CreateTableReq)
	a := args[1].(gate.Agent)

	tid := genTableId()
	table := NewTable(tid)
	a.SetUserData(&userdata.UserData{
		Uid: req.Uid,
		Tid: tid,
	})
	table.AddAgent(a, true)
	go table.Run()
	tables[tid] = table
	log.Debug("create table uid:%v", req.Uid)
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

	tid := req.TableId
	if table, ok := tables[tid]; ok {
		if _, ok := table.GetPlayerIndex(req.Uid); ok == nil {
			rsp.ErrCode = 10000
			rsp.ErrMsg = "you areadly in table"
		} else {
			a.SetUserData(&userdata.UserData{
				Uid: req.Uid,
				Tid: tid,
			})
			tables[tid].AddAgent(a, false)
			rsp.ErrCode = 0
			rsp.ErrMsg = "join successed!"
			table.BroadcastExceptMe(&proto.UserJoinTableMsg{
				Uid: req.Uid,
				Tid: tid,
			}, req.Uid)
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
			if curTableId > 100000 {
				curTableId = 10000
			}
		} else {
			return curTableId
		}
	}
}
