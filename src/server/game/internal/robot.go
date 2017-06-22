package internal

import (
	"errors"
	"reflect"
	"server/utils"
	"server/proto"
)

type robot interface {
	HandlerMsg(req interface{}) (interface{}, error)
}

type BaseRobot struct {
	player *Player
}

func NewRobot(p *Player) *BaseRobot {
	a := new(BaseRobot)
	a.player = p
	return a
}

func (a *BaseRobot) HandlerMsg(req interface{}) (interface{}, error) {
	if reflect.TypeOf(req) == reflect.TypeOf(&proto.OperatReq{}) {
		return a.HandlerOperatMsg(req.(*proto.OperatReq))
	} else if reflect.TypeOf(req) == reflect.TypeOf(&proto.TableOperatReq{}) {
		return a.HandlerTableOperatMsg(req.(*proto.TableOperatReq))
	}
	return nil, errors.New("err msg type")
}

func (a *BaseRobot) HandlerTableOperatMsg(req *proto.TableOperatReq) (*proto.TableOperatRsp, error) {
	var ok bool
	if req.Type == proto.TableOperat_TableStart {
		ok = true
	} else if req.Type == proto.TableOperat_TableContinue {
		ok = false
	}
	rsp := &proto.TableOperatRsp{Ok: ok, Type: req.Type}
	return rsp, nil
}

func (a *BaseRobot) HandlerOperatMsg(req *proto.OperatReq) (*proto.OperatRsp, error) {
	rsp := proto.NewOperatRsp()
	if req.Type&proto.OperatType_DealOperat != 0 {
		rsp.Type = proto.OperatType_DealOperat
		a.Deal(req.DealReq, rsp.DealRsp)
	} else if req.Type&proto.OperatType_HuOperat != 0 {
		rsp.Type = proto.OperatType_HuOperat
		a.Hu(req.HuReq, rsp.HuRsp)
	} else if req.Type&proto.OperatType_DrawOperat != 0 {
		rsp.Type = proto.OperatType_DrawOperat
		a.Draw(req.DrawReq, rsp.DrawRsp)
	} else if req.Type&proto.OperatType_GangOperat != 0 {
		rsp.Type = proto.OperatType_GangOperat
		a.Gang(req.GangReq, rsp.GangRsp)
	} else if req.Type&proto.OperatType_PongOperat != 0 {
		rsp.Type = proto.OperatType_PongOperat
		a.Pong(req.PongReq, rsp.PongRsp)
	} else if req.Type&proto.OperatType_EatOperat != 0 {
		rsp.Type = proto.OperatType_EatOperat
		a.Eat(req.EatReq, rsp.EatRsp)
	} else if req.Type&proto.OperatType_DropOperat != 0 {
		rsp.Type = proto.OperatType_DropOperat
		a.Drop(req.DropReq, rsp.DropRsp)
	}
	return rsp, nil
}

func (a *BaseRobot) Hu(req *proto.HuReq, rsp *proto.HuRsp) bool {
	rsp.Ok = true
	rsp.Card = req.Card
	rsp.Type = req.Type
	rsp.Lose = req.Lose
	return true
}

func (a *BaseRobot) Deal(req *proto.DealReq, rsp *proto.DealRsp) bool {
	return true
}

func (a *BaseRobot) Draw(req *proto.DrawReq, rsp *proto.DrawRsp) bool {
	return true
}

func (a *BaseRobot) Drop(req *proto.DropReq, rsp *proto.DropRsp) bool {
	cards_copy := utils.Copy(a.player.cards)
	separate_result := utils.SeparateCards(cards_copy, a.player.table.hun_card)
	discard := utils.DropSingle(separate_result)
	if discard == 0 {
		discard = utils.DropRand(cards_copy, a.player.table.hun_card)
	}
	rsp.DisCard = discard
	return true
}

func (a *BaseRobot) Eat(req *proto.EatReq, rsp *proto.EatRsp) bool {
	rsp.Eat = req.Eat[0]
	return true
}

func (a *BaseRobot) Gang(req *proto.GangReq, rsp *proto.GangRsp) bool {
	rsp.Ok = true
	rsp.Gang = req.Gang[0]
	return true
}

func (a *BaseRobot) Pong(req *proto.PongReq, rsp *proto.PongRsp) bool {
	rsp.Card, rsp.Ok = req.Card, true
	return true
}
