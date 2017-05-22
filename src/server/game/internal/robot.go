package internal

import (
	"server/proto"
	"server/mahjong"
	"math/rand"
	"time"
)

type robot interface {
	HandlerOperatMsg(req *proto.OperatReq) (*proto.OperatRsp, error)
}

type Agent struct {
	player  *Player
	rand    *rand.Rand
}

func NewAgent(p *Player) *Agent{
	a := new(Agent)
	a.player = p
	a.rand = rand.New(rand.NewSource(time.Now().UnixNano()))
	return a
}

func (a *Agent)HandlerOperatMsg(req *proto.OperatReq) (*proto.OperatRsp, error) {
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

func (a *Agent)Hu(req *proto.HuReq, rsp *proto.HuRsp) bool {
	rsp.Ok = true
	return true
}

func (a *Agent) Deal(req *proto.DealReq, rsp *proto.DealRsp) bool {
	return true
}

func (a *Agent) Draw(req *proto.DrawReq, rsp *proto.DrawRsp) bool {
	return true
}

func (a *Agent)Drop(req *proto.DropReq, rsp *proto.DropRsp) bool {
	cards_copy := mahjong.Copy(a.player.cards)
	separate_result := mahjong.SeparateCards(cards_copy, a.player.table.hun_card)
	discard := DropSingle(separate_result)
	if discard == 0 {
		discard = DropRand(cards_copy, a.player.table.hun_card)
	}
	rsp.DisCard = discard
	return true
}

func (a *Agent)Eat(req *proto.EatReq, rsp *proto.EatRsp) bool {
	rsp.Eat = req.Eat[0]
	return true
}

func (a *Agent)Pong(req *proto.PongReq, rsp *proto.PongRsp) bool {
	rsp.Card, rsp.Count = req.Card, req.Count
	return true
}

func DropSingle(separate_result [5][]int32) int32 {
	wind_cards := separate_result[4]
	if len(wind_cards) == 1 {
		return wind_cards[0]
	} else {
		for _, card := range wind_cards {
			if mahjong.Count(wind_cards, card) == 1 {
				return card
			}
		}
	}

	for i := 1; i < 4; i++ {
		min_card, max_card := int32(i*100+1), int32(i*100+9)
		if mahjong.Count(separate_result[i], min_card) == 1 && mahjong.Count(separate_result[i], min_card+1) == 0 && mahjong.Count(separate_result[i], min_card+2) == 0 {
			return min_card
		}
		if mahjong.Count(separate_result[i], max_card) == 1 && mahjong.Count(separate_result[i], max_card-1) == 0 && mahjong.Count(separate_result[i], max_card-2) == 0 {
			return max_card
		}
	}

	for i := 1; i < 4; i++ {
		for _, card := range separate_result[i] {
			if mahjong.Count(separate_result[i], card) > 1 {
				continue
			} else if mahjong.Count(separate_result[i], card+1) > 0 || mahjong.Count(separate_result[i], card-1) > 0 {
				continue
			} else {
				return card
			}
		}
	}

	return 0
}

func DropRand(cards []int32, hun_card int32) int32 {
	for {
		index := rand.Intn(len(cards))
		if hun_card != cards[index] {
			return cards[index]
		}
	}
}
