package area

import (
	"server/proto"
	"server/mahjong"
	"github.com/jxbdlut/leaf/log"
)

type DefaultRule struct {
	name          string
	has_wind      bool
	has_hun       bool
	is_258		  bool
	sepical_cards []int32
}

func NewDefaultRule() Rule {
	rule := new(DefaultRule)
	rule.has_wind = true
	rule.has_hun = true
	rule.is_258 = true
	rule.sepical_cards = append(rule.sepical_cards, 407)
	return rule
}

func (m *DefaultRule) IsJiang(card int32) bool {
	if m.is_258 {
		t := card / 100
		v := card % 10
		if t != 4 && (v == 2 || v == 5 || v == 8) {
			return true
		} else {
			return false
		}
	}
	return true
}

func (m *DefaultRule) HasHun() bool {
	return m.has_hun
}

func (m *DefaultRule) HasWind() bool {
	return m.has_wind
}

func (m *DefaultRule) CanHu(disCard mahjong.DisCard, player *proto.Player, req *proto.OperatReq) bool {
	card := disCard.Card
	if player.CancelHu {
		return false
	}
	hu := false
	if _, ok := player.PrewinCards[1]; ok {
		hu = true
	}
	if _, ok := player.PrewinCards[2]; ok {
		if m.IsJiang(disCard.Card) {
			hu = true
		}
	}
	if _, ok := player.PrewinCards[card]; ok {
		hu = true
	}
	if (card == player.HunCard && len(player.PrewinCards) > 0) || hu {
		req.Type = req.Type | proto.OperatType_HuOperat
		req.HuReq.Card = card
		if disCard.FromUid == player.Uid {
			req.HuReq.Type = proto.HuType_Mo
			if disCard.DisType == mahjong.DisCard_SelfGang {
				req.HuReq.Type = proto.HuType_GangHua
			}
			if disCard.DisType == mahjong.DisCard_HaiDi {
				req.HuReq.Type = proto.HuType_HaiDiLao
			}
		} else {
			req.HuReq.Lose = disCard.FromUid
			if disCard.DisType == mahjong.DisCard_BuGang {
				req.HuReq.Type = proto.HuType_QiangGang
			}
		}
		return true
	}
	return false
}

func (m *DefaultRule) CanEat(disCard mahjong.DisCard, player *proto.Player, req *proto.OperatReq) bool {
	var eats []*proto.Eat
	card := disCard.Card
	t := card / 100
	if t == 4 || disCard.Card == player.HunCard {
		return false
	}

	i, err := player.GetPlayerIndex(disCard.FromUid)
	if err != nil {
		log.Error("uid:%v, fromuid:%v not in table, err:%v", player.Uid, disCard.FromUid, err)
		return false
	}
	j, err := player.GetPlayerIndex(player.Uid)
	if err != nil {
		log.Error("uid:%v, fromuid:%v not in table, err:%v", player.Uid, disCard.FromUid, err)
		return false
	}
	if (i + 1)%len(player.Pos) != j  {
		return false
	}

	c_1 := mahjong.Count(player.Cards, card-1)
	c_2 := mahjong.Count(player.Cards, card-2)
	c1 := mahjong.Count(player.Cards, card+1)
	c2 := mahjong.Count(player.Cards, card+2)

	if c_1 > 0 && c_2 > 0 && (player.HunCard < card-2 || player.HunCard > card) {
		var eat proto.Eat
		eat.HandCard = []int32{card - 2, card - 1}
		eat.WaveCard = []int32{card - 2, card - 1, card}
		eats = append(eats, &eat)
	}
	if c_1 > 0 && c1 > 0 && (player.HunCard < card-1 || player.HunCard > card+1) {
		var eat proto.Eat
		eat.HandCard = []int32{card - 1, card + 1}
		eat.WaveCard = []int32{card - 1, card, card + 1}
		eats = append(eats, &eat)
	}
	if c1 > 0 && c2 > 0 && (player.HunCard < card || player.HunCard > card+2) {
		var eat proto.Eat
		eat.HandCard = []int32{card + 1, card + 2}
		eat.WaveCard = []int32{card, card + 1, card + 2}
		eats = append(eats, &eat)
	}
	if len(eats) > 0 {
		req.Type = req.Type | proto.OperatType_EatOperat
		req.EatReq.Eat = eats
		return true
	}
	return false
}

func (m *DefaultRule) CanAnGang(player *proto.Player, req *proto.OperatReq) bool {
	ret := false
	separate_result := mahjong.SeparateCards(player.Cards, player.HunCard)
	for _, m := range separate_result {
		if len(m) < 4 {
			continue
		}
		record := []int32{}
		for _, card := range m {
			count := mahjong.Count(player.Cards, card)
			if count < 4 {
				continue
			}
			if mahjong.Contain(record, card) {
				continue
			}
			req.Type = req.Type | proto.OperatType_GangOperat
			gang := &proto.Gang{
				Cards: []int32{card, card, card, card},
				Type: proto.GangType_AnGang,
			}
			req.GangReq.Gang = append(req.GangReq.Gang, gang)
			record = append(record, card)
			ret = true
		}
	}
	return ret
}

func (m *DefaultRule) CanBuGang(player *proto.Player, req *proto.OperatReq) bool {
	ret := false
	for _, wave := range player.Waves {
		if wave.WaveType != proto.Wave_PongWave {
			continue
		}
		if mahjong.Count(player.Cards, wave.Cards[0]) > 0 {
			req.Type = req.Type | proto.OperatType_GangOperat
			gang := &proto.Gang{
				Cards: []int32{wave.Cards[0]},
				Type: proto.GangType_BuGang,
			}
			req.GangReq.Gang = append(req.GangReq.Gang, gang)
			ret = true
		}
	}
	return ret
}

func (m *DefaultRule) CanMingGang(disCard mahjong.DisCard, player *proto.Player, req *proto.OperatReq) bool {
	if disCard.FromUid == player.Uid {
		return false
	}
	card := disCard.Card
	count := mahjong.Count(player.Cards, card)
	if count > 2 {
		req.Type = req.Type | proto.OperatType_GangOperat
		gang := &proto.Gang{
			Cards: []int32{card, card, card},
			Type: proto.GangType_MingGang,
		}
		req.GangReq.Gang = append(req.GangReq.Gang, gang)
		return true
	}
	return false
}

func (m *DefaultRule) CanPong(disCard mahjong.DisCard, player *proto.Player, req *proto.OperatReq) bool {
	if disCard.FromUid == player.Uid {
		return false
	}
	card := disCard.Card
	count := mahjong.Count(player.Cards, card)
	if count > 1 {
		req.Type = req.Type | proto.OperatType_PongOperat
		req.PongReq.Card = card
		return true
	}
	return false
}

func (m *DefaultRule) Hu(player *proto.Player, huRsp *proto.HuRsp) {

}
