package internal

import (
	"reflect"

	"github.com/name5566/leaf/gate"
	"github.com/name5566/leaf/log"
	"server/proto"
	"server/userdata"
)

var (
	tables     map[uint32]*Table
	curTableId uint32 = 10000
)

func init() {
	handler(&proto.CreateTableReq{}, handlerCreateTable)
	handler(&proto.JoinTableReq{}, handlerJoinTable)
	tables = make(map[uint32]*Table)
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
	table.addAgent(a, true)
	go table.run()
	tables[tid] = table
	log.Debug("create table Uid:%v", req.Uid)
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
		if _, ok := table.getPlayerIndex(req.Uid); ok == nil {
			rsp.ErrCode = 10000
			rsp.ErrMsg = "you areadly in table"
		} else {
			a.SetUserData(&userdata.UserData{
				Uid: req.Uid,
				Tid: tid,
			})
			tables[tid].addAgent(a, false)
			rsp.ErrCode = 0
			rsp.ErrMsg = "join successed!"
			//table.Broadcast(&proto.UserJoinTableMsg{
			//	Uid:req.Uid,
			//	Name:a.UserData().(*userdata.UserData).Name,
			//	Tid:tid,
			//})
		}
	} else {
		log.Error("table is not exist, tid:%v", tid)
		rsp.ErrCode = -1
		rsp.ErrMsg = "table is not exist"
	}

	a.WriteMsg(&rsp)
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
