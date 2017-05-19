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
	proto.Processor.SetRouter(&proto.DealCardRsp{}, game.ChanRPC)
	proto.Processor.SetRouter(&proto.DrawCardRsp{}, game.ChanRPC)
	proto.Processor.SetRouter(&proto.HuRsp{}, game.ChanRPC)
	proto.Processor.SetRouter(&proto.EatRsp{}, game.ChanRPC)
	proto.Processor.SetRouter(&proto.PongRsp{}, game.ChanRPC)
}
