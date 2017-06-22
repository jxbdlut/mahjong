package area

import (
	"server/proto"
	"server/utils"
)

type Rule interface {
	HasHun() bool
	HasWind() bool
	IsJiang(card int32) bool
	CanHu(disCard utils.DisCard, player *proto.Player, req *proto.OperatReq) bool
	CanEat(disCard utils.DisCard, player *proto.Player, req *proto.OperatReq) bool
	CanAnGang(player *proto.Player, req *proto.OperatReq) bool
	CanBuGang(player *proto.Player, req *proto.OperatReq) bool
	CanMingGang(disCard utils.DisCard, player *proto.Player, req *proto.OperatReq) bool
	CanPong(disCard utils.DisCard, player *proto.Player, req *proto.OperatReq) bool
	Hu(player *proto.Player, huRsp *proto.HuRsp)
	GetTingCards(player *proto.Player) map[int32]interface{}
}

type Ting interface {
	Info() string
}
