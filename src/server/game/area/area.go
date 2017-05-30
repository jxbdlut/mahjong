package area

import (
	"server/proto"
	"server/mahjong"
)

type Rule interface {
	HasHun() bool
	HasWind() bool
	IsJiang(card int32) bool
	CanHu(disCard mahjong.DisCard, player *proto.Player, req *proto.OperatReq) bool
	CanEat(disCard mahjong.DisCard, player *proto.Player, req *proto.OperatReq) bool
	CanAnGang(player *proto.Player, req *proto.OperatReq) bool
	CanBuGang(player *proto.Player, req *proto.OperatReq) bool
	CanMingGang(disCard mahjong.DisCard, player *proto.Player, req *proto.OperatReq) bool
	CanPong(disCard mahjong.DisCard, player *proto.Player, req *proto.OperatReq) bool
	Hu(player *proto.Player, huRsp *proto.HuRsp)
}

