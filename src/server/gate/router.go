package gate

import (
	"server/game"
	"server/login"
	"server/proto"
)

func init() {
	proto.Processor.SetRouter(&proto.LoginReq{}, login.ChanRPC)
	proto.Processor.SetRouter(&proto.CreateTableReq{}, game.ChanRPC)
	proto.Processor.SetRouter(&proto.JoinTableReq{}, game.ChanRPC)
	proto.Processor.SetRouter(&proto.OperatRsp{}, game.ChanRPC)
	proto.Processor.SetRouter(&proto.ContinueRsp{}, game.ChanRPC)
}
