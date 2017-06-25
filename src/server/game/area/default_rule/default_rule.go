package default_rule

import (
	"github.com/jxbdlut/leaf/log"
	"server/game/area"
	"server/game/area/base_rule"
	"server/proto"
	"server/utils"
)

type DefaultRule struct {
	name        string
	qin_yi_se   bool
	jiang_yi_se bool
	wind_yi_se  bool
	base_rule   *base_rule.BaseRule
}

func NewDefaultRule() area.Rule {
	rule := new(DefaultRule)
	rule.base_rule = base_rule.NewBaseRule(true, true, true)
	return rule
}

func (m *DefaultRule) IsJiang(card int32) bool {
	if m.qin_yi_se {
		return true
	}
	return m.base_rule.IsJiang(card)
}

func (m *DefaultRule) HasHun() bool {
	return m.base_rule.HasHun()
}

func (m *DefaultRule) HasWind() bool {
	return m.base_rule.HasWind()
}

func (m *DefaultRule) CanHu(disCard utils.DisCard, player *proto.Player, req *proto.OperatReq) bool {
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
			if disCard.DisType == utils.DisCard_SelfGang {
				req.HuReq.Type = proto.HuType_GangHua
			}
			if disCard.DisType == utils.DisCard_HaiDi {
				req.HuReq.Type = proto.HuType_HaiDiLao
			}
		} else {
			req.HuReq.Lose = disCard.FromUid
			if disCard.DisType == utils.DisCard_BuGang {
				req.HuReq.Type = proto.HuType_QiangGang
			}
		}
		return true
	}
	return false
}

func (m *DefaultRule) CanEat(disCard utils.DisCard, player *proto.Player, req *proto.OperatReq) bool {
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
	if (i+1)%len(player.Pos) != j {
		return false
	}

	c_1 := utils.Count(player.Cards, card-1)
	c_2 := utils.Count(player.Cards, card-2)
	c1 := utils.Count(player.Cards, card+1)
	c2 := utils.Count(player.Cards, card+2)

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
	separate_result := utils.SeparateCards(player.Cards, player.HunCard)
	for _, m := range separate_result {
		if len(m) < 4 {
			continue
		}
		record := []int32{}
		for _, card := range m {
			count := utils.Count(player.Cards, card)
			if count < 4 {
				continue
			}
			if utils.Contain(record, card) {
				continue
			}
			req.Type = req.Type | proto.OperatType_GangOperat
			gang := &proto.Gang{
				Cards: []int32{card, card, card, card},
				Type:  proto.GangType_AnGang,
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
		if utils.Count(player.Cards, wave.Cards[0]) > 0 {
			req.Type = req.Type | proto.OperatType_GangOperat
			gang := &proto.Gang{
				Cards: []int32{wave.Cards[0]},
				Type:  proto.GangType_BuGang,
			}
			req.GangReq.Gang = append(req.GangReq.Gang, gang)
			ret = true
		}
	}
	return ret
}

func (m *DefaultRule) CanMingGang(disCard utils.DisCard, player *proto.Player, req *proto.OperatReq) bool {
	if disCard.FromUid == player.Uid {
		return false
	}
	card := disCard.Card
	count := utils.Count(player.Cards, card)
	if count > 2 {
		req.Type = req.Type | proto.OperatType_GangOperat
		gang := &proto.Gang{
			Cards: []int32{card, card, card},
			Type:  proto.GangType_MingGang,
		}
		req.GangReq.Gang = append(req.GangReq.Gang, gang)
		return true
	}
	return false
}

func (m *DefaultRule) CanPong(disCard utils.DisCard, player *proto.Player, req *proto.OperatReq) bool {
	if disCard.FromUid == player.Uid {
		return false
	}
	card := disCard.Card
	count := utils.Count(player.Cards, card)
	if count > 1 {
		req.Type = req.Type | proto.OperatType_PongOperat
		req.PongReq.Card = card
		return true
	}
	return false
}

func (m *DefaultRule) Hu(player *proto.Player, huRsp *proto.HuRsp) {

}
func (m *DefaultRule) Check2Combine(card1 int32, card2 int32) bool {
	if card1 == card2 {
		if m.IsJiang(card1) {
			return true
		} else {
			return false
		}
	}
	return false
}

func (m *DefaultRule) GetNeedHunInSub(sub_cards []int32, hun_num int32, need_hun_count int32) int32 {
	if need_hun_count == 0 {
		return need_hun_count
	}

	len_sub_cards := len(sub_cards)
	if hun_num+m.base_rule.GetModNeedNum(len_sub_cards, false) >= need_hun_count {
		return need_hun_count
	}

	if len_sub_cards == 0 {
		return utils.Min(hun_num, need_hun_count)
	} else if len_sub_cards == 1 {
		return utils.Min(hun_num+2, need_hun_count)
	} else if len_sub_cards == 2 {
		m, v0, v1 := sub_cards[0]/100, sub_cards[0]%10, sub_cards[1]%10
		if m == 4 {
			if v0 == v1 {
				return utils.Min(hun_num+1, need_hun_count)
			}
		} else if v1-v0 < 3 {
			return utils.Min(hun_num+1, need_hun_count)
		}
	} else if len_sub_cards >= 3 {
		t, v0 := sub_cards[0]/100, sub_cards[0]%10

		// 第一个和后两个一铺
		for i := 1; i < len_sub_cards; i++ {
			if hun_num+m.base_rule.GetModNeedNum(len_sub_cards-3, false) >= need_hun_count {
				break
			}
			v1 := sub_cards[i] % 10
			// 13444   134不可能连一起
			if v1-v0 > 1 {
				break
			}
			if i+2 < len_sub_cards {
				if sub_cards[i+2]%10 == v1 {
					continue
				}
			}
			if i+1 < len_sub_cards {
				tmp1, tmp2, tmp3 := sub_cards[0], sub_cards[i], sub_cards[i+1]
				if m.base_rule.Check3Combine(tmp1, tmp2, tmp3) {
					tmp_cards := utils.Copy(sub_cards)
					tmp_cards = utils.DelCard(tmp_cards, tmp1, tmp2, tmp3)
					need_hun_count = m.GetNeedHunInSub(tmp_cards, hun_num, need_hun_count)
				}
			}
		}

		// 第一个和第二个一铺
		v1 := sub_cards[1] % 10
		if hun_num+m.base_rule.GetModNeedNum(len_sub_cards-2, false)+1 < need_hun_count {
			if t == 4 {
				if v0 == v1 {
					tmp_cards := utils.Copy(sub_cards[2:])
					need_hun_count = m.GetNeedHunInSub(tmp_cards, hun_num+1, need_hun_count)
				}
			} else {
				for i := 1; i < len_sub_cards; i++ {
					if hun_num+m.base_rule.GetModNeedNum(len_sub_cards-2, false)+1 >= need_hun_count {
						break
					}
					v1 = sub_cards[i] % 10
					// 如果当前的value不等于下一个value则和下一个结合避免重复
					if i+1 != len_sub_cards {
						v2 := sub_cards[i+1] % 10
						if v1 == v2 {
							continue
						}
					}
					mius := v1 - v0
					if mius < 3 {
						tmp1, tmp2 := sub_cards[0], sub_cards[i]
						tmp_cards := utils.Copy(sub_cards)
						tmp_cards = utils.DelCard(tmp_cards, tmp1, tmp2, 0)
						need_hun_count = m.GetNeedHunInSub(tmp_cards, hun_num+1, need_hun_count)
						if mius >= 1 {
							break
						}
					} else {
						break
					}
				}
			}
		}
		// 第一个自己一铺
		if hun_num+m.base_rule.GetModNeedNum(len_sub_cards-1, false)+2 < need_hun_count {
			tmp_cards := utils.Copy(sub_cards[1:])
			need_hun_count = m.GetNeedHunInSub(tmp_cards, hun_num+2, need_hun_count)
		}
	}
	return need_hun_count
}

func (m *DefaultRule) GetNeedHunInSubWithEye(cards []int32, min_need_num int32) int32 {
	// 拷贝
	cards_copy := utils.Copy(cards)
	len_cards := len(cards_copy)
	if len_cards == 0 {
		return 2
	}
	if min_need_num < m.base_rule.GetModNeedNum(len_cards, true) {
		return min_need_num
	}
	for i := 0; i < len_cards; i++ {
		if i == len_cards-1 { // 如果是最后一张牌
			tmp_cards := utils.Copy(cards_copy)
			if m.IsJiang(cards_copy[i]) {
				tmp_cards = utils.DelCard(tmp_cards, cards_copy[i], 0, 0)
				min_need_num = utils.Min(min_need_num, m.GetNeedHunInSub(tmp_cards, 0, 4)+1)
			} else {
				min_need_num = utils.Min(min_need_num, m.GetNeedHunInSub(tmp_cards, 0, 4)+2)
			}
		} else {
			if i+2 == len_cards || cards_copy[i]%10 != cards_copy[i+2]%10 {
				tmp_cards := utils.Copy(cards_copy)
				if m.Check2Combine(cards_copy[i], cards_copy[i+1]) {
					tmp_cards = utils.DelCard(tmp_cards, cards_copy[i], cards_copy[i+1], 0)
					min_need_num = utils.Min(min_need_num, m.GetNeedHunInSub(tmp_cards, 0, 4))
				} else {
					min_need_num = utils.Min(min_need_num, m.GetNeedHunInSub(tmp_cards, 0, 4)+2)
				}
			}
			if cards_copy[i]%10 != cards_copy[i+1]%10 {
				tmp_cards := utils.Copy(cards_copy)
				if m.IsJiang(tmp_cards[i]) {
					tmp_cards = utils.DelCard(tmp_cards, cards_copy[i], 0, 0)
					min_need_num = utils.Min(min_need_num, m.GetNeedHunInSub(tmp_cards, 0, 4)+1)
				} else if m.IsJiang(tmp_cards[i+1]) {
					tmp_cards = utils.DelCard(tmp_cards, cards_copy[i+1], 0, 0)
					min_need_num = utils.Min(min_need_num, m.GetNeedHunInSub(tmp_cards, 0, 4)+1)
				} else {
					min_need_num = utils.Min(min_need_num, m.GetNeedHunInSub(tmp_cards, 0, 4)+2)
				}
			}
		}
	}
	return min_need_num
}

func (m *DefaultRule) SumNeedHun(need_hun_arr []int32) int32 {
	var sum int32
	for _, num := range need_hun_arr {
		sum = sum + num
	}
	return sum
}

func (m *DefaultRule) GetBestComb(separate_results [5][]int32, need_hun_arr []int32, need_hun_with_eye_arr []int32) (int32, []int) {
	min_need_num := int32(5)
	sum_num := m.SumNeedHun(need_hun_arr)
	var result []int

	for i := 0; i < 4; i++ {
		need_num := sum_num - need_hun_arr[i] + need_hun_with_eye_arr[i]
		if need_num < min_need_num && len(separate_results[i+1]) != 0 {
			min_need_num = need_num
			result = append(result[:0], result[:0]...)
			result = append(result, i)
		} else if need_num == min_need_num && len(separate_results[i+1]) != 0 {
			result = append(result, i)
		}
	}
	return min_need_num, result
}

func (m *DefaultRule) CheckQingYiSe(player *proto.Player) map[int32]interface{} {
	result := make(map[int32]interface{})
	var se_count []int32
	separate_results := utils.SeparateCards(player.Cards, player.HunCard)

	for _, wave := range player.Waves {
		t := wave.Cards[0] / 100
		if utils.Contain(se_count, t) {
			continue
		}
		se_count = append(se_count, t)
		if len(se_count) >= 2 {
			return result
		}
	}

	for _, cards := range separate_results[1:] {
		if len(cards) != 0 {
			t := cards[0] / 100
			if utils.Contain(se_count, t) {
				continue
			}
			se_count = append(se_count, t)
			if len(se_count) >= 2 {
				return result
			}
		}
	}

	m.qin_yi_se = true
	t := se_count[0]
	cur_hun_num := int32(len(separate_results[0]))
	need_num := m.GetNeedHunInSub(separate_results[t], 0, 4)
	if need_num < cur_hun_num {
		result[2] = m.NewTing(2)
		m.qin_yi_se = false
		return result
	}
	for i := int32(1); i < 10; i++ {
		card := int32(t*100 + i)
		if card == player.HunCard {
			continue
		}
		tmp_cards := utils.Copy(separate_results[t])
		tmp_cards = append(tmp_cards, card)
		utils.SortCards(tmp_cards, player.HunCard)
		if m.GetNeedHunInSubWithEye(tmp_cards, 4) <= cur_hun_num {
			result[card] = m.NewTing(card)
		}
	}
	m.qin_yi_se = false
	return result
}

func (m *DefaultRule) NewTing(card int32) *Ting {
	return NewTing(card, m.qin_yi_se, m.jiang_yi_se, m.wind_yi_se)
}

func (m *DefaultRule) CheckPengPengHu(player *proto.Player, result map[int32]interface{}) map[int32]interface{} {
	separate_results := utils.SeparateCards(player.Cards, player.HunCard)
	for _, wave := range player.Waves {
		if wave.WaveType == proto.Wave_EatWave {
			return result
		}
	}
	cur_hun_num := len(separate_results[0])
	var need_hun int
	eye := false
	for _, cards := range separate_results[1:] {
		cache_cards := []int32{}
		for _, card := range cards {
			if utils.Contain(cache_cards, card) {
				continue
			}
			cache_cards = append(cache_cards, card)
			count := utils.Count(cards, card)
			switch count {
			case 1:
				if eye {
					need_hun = need_hun + 2
					result[card] = m.NewTing(card).SetPengPengHu()
				} else {
					eye = true
					need_hun = need_hun + 1
					result[card] = m.NewTing(card).SetPengPengHu()
				}
			case 2:
				if eye {
					need_hun = need_hun + 1
					result[card] = m.NewTing(card).SetPengPengHu()
				} else {
					eye = true
					result[card] = m.NewTing(card).SetPengPengHu()
				}
			case 3:
			case 4:
				if eye {
					need_hun = need_hun + 2
					result[card] = m.NewTing(card).SetPengPengHu()
				} else {
					eye = true
					need_hun = need_hun + 1
					result[card] = m.NewTing(card).SetPengPengHu()
				}
			}
			if cur_hun_num+1 < need_hun {
				result = make(map[int32]interface{})
				return result
			}
		}
	}
	if eye && cur_hun_num > need_hun+1 || !eye && cur_hun_num > need_hun {
		result = make(map[int32]interface{})
		result[1] = m.NewTing(1).SetPengPengHu()
	}
	return result
}

func (m *DefaultRule) CheckJiangYiSe(player *proto.Player) bool {
	for _, wave := range player.Waves {
		if wave.WaveType == proto.Wave_EatWave {
			return false
		}
		if !m.base_rule.IsJiang(wave.Cards[0]) {
			return false
		}
	}
	for _, card := range player.Cards {
		if !m.base_rule.IsJiang(card) {
			return false
		}
	}
	return true
}

func (m *DefaultRule) CheckWindYiSe(player *proto.Player) bool {
	for _, wave := range player.Waves {
		if wave.WaveType == proto.Wave_EatWave {
			return false
		}
		if wave.Cards[0]/100 != 4 {
			return false
		}
	}
	for _, card := range player.Cards {
		if card/100 != 4 {
			return false
		}
	}
	return true
}

func (m *DefaultRule) GetTingCards(player *proto.Player) ([]int32, []int32, map[int32]interface{}) {
	separate_results := utils.SeparateCards(player.Cards, player.HunCard)
	result := m.CheckQingYiSe(player)
	m.jiang_yi_se = m.CheckJiangYiSe(player)
	m.wind_yi_se = m.CheckWindYiSe(player)
	result = m.CheckPengPengHu(player, result)
	if m.jiang_yi_se {
		result[2] = m.NewTing(2).SetJiangYiSe()
		return player.NeedHun, player.NeedHunWithEye, result
	}
	if m.wind_yi_se {
		result[4] = m.NewTing(4).SetWindYiSe()
		return player.NeedHun, player.NeedHunWithEye, result
	}
	if ok := result[1]; ok != nil {
		return player.NeedHun, player.NeedHunWithEye, result
	}
	cur_hun_num := int32(len(separate_results[0]))
	for i, update_flag := range player.IsNeedUpdate {
		if update_flag {
			player.NeedHun[i] = m.GetNeedHunInSub(separate_results[i+1], 0, 4)
			player.NeedHunWithEye[i] = m.GetNeedHunInSubWithEye(separate_results[i+1], 4)
		}
	}
	need_num, index := m.GetBestComb(separate_results, player.NeedHun, player.NeedHunWithEye)
	log.Debug("uid:%v separate_results:%v", player.Uid, separate_results)
	log.Debug("uid:%v need_hun_arr:%v, need_hun_with_eye_arr:%v index:%v", player.Uid, player.NeedHun, player.NeedHunWithEye, index)
	if cur_hun_num-need_num >= 2 {
		result[1] = m.NewTing(1)
		return player.NeedHun, player.NeedHunWithEye, result
	}
	if cur_hun_num-m.SumNeedHun(player.NeedHun) > 0 {
		result[2] = m.NewTing(2)
		return player.NeedHun, player.NeedHunWithEye, result
	}
	if need_num > cur_hun_num+1 {
		return player.NeedHun, player.NeedHunWithEye, result
	}
	var cache_index []int32
	for _, i := range index {
		begin, end := utils.SearchRange(separate_results[i+1])
		for card := begin; card < end+1; card++ {
			if card == player.HunCard {
				continue
			}
			if ok := result[card]; ok != nil {
				continue
			}
			tmp_cards := utils.Copy(separate_results[i+1])
			tmp_cards = append(tmp_cards, card)
			utils.SortCards(tmp_cards, player.HunCard)
			if m.GetNeedHunInSubWithEye(tmp_cards, 4) <= player.NeedHunWithEye[i]-1 {
				result[card] = m.NewTing(card)
			}
		}
		for j := 0; j < 4; j++ {
			if j != i && len(separate_results[j+1]) != 0 && !utils.Contain(cache_index, int32(j)) {
				cache_index = append(cache_index, int32(j))
				begin, end := utils.SearchRange(separate_results[j+1])
				for card := begin; card < end+1; card++ {
					if card == player.HunCard {
						continue
					}
					if ok := result[card]; ok != nil {
						continue
					}
					tmp_cards := utils.Copy(separate_results[j+1])
					tmp_cards = append(tmp_cards, card)
					utils.SortCards(tmp_cards, player.HunCard)
					if m.GetNeedHunInSub(tmp_cards, 0, 4) <= player.NeedHun[j]-1 {
						result[card] = m.NewTing(card)
					}
				}
			}
		}
	}
	return player.NeedHun, player.NeedHunWithEye, result
}
