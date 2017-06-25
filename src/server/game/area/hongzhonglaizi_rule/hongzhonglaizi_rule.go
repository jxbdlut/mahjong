package hongzhonglaizi_rule

import (
	"server/game/area"
	"server/game/area/base_rule"
	"server/utils"
	"server/proto"
	"github.com/jxbdlut/leaf/log"
)

type HongZhongLaiZiRule struct {
	name          string
	base_rule     *base_rule.BaseRule
}

func NewHongZhongLaiZiRule() area.Rule {
	rule := new(HongZhongLaiZiRule)
	rule.base_rule = base_rule.NewBaseRule(true, true, true)
	return rule
}

func (m *HongZhongLaiZiRule) IsJiang(card int32) bool {
	return m.base_rule.IsJiang(card)
}

func (m *HongZhongLaiZiRule) HasHun() bool {
	return m.base_rule.HasHun()
}

func (m *HongZhongLaiZiRule) HasWind() bool {
	return m.base_rule.HasWind()
}


func (m *HongZhongLaiZiRule) CanHu(disCard utils.DisCard, player *proto.Player, req *proto.OperatReq) bool {
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

func (m *HongZhongLaiZiRule) CanEat(disCard utils.DisCard, player *proto.Player, req *proto.OperatReq) bool {
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

func (m *HongZhongLaiZiRule) CanAnGang(player *proto.Player, req *proto.OperatReq) bool {
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

func (m *HongZhongLaiZiRule) CanBuGang(player *proto.Player, req *proto.OperatReq) bool {
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

func (m *HongZhongLaiZiRule) CanMingGang(disCard utils.DisCard, player *proto.Player, req *proto.OperatReq) bool {
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

func (m *HongZhongLaiZiRule) CanPong(disCard utils.DisCard, player *proto.Player, req *proto.OperatReq) bool {
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

func (m *HongZhongLaiZiRule) Hu(player *proto.Player, huRsp *proto.HuRsp) {

}

func (m *HongZhongLaiZiRule) GetNeedHunInSub(sub_cards []int32, hun_num int32, need_hun_count int32) int32 {
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

func (m *HongZhongLaiZiRule) GetNeedHunInSubWithEye(cards []int32, min_need_num int32) int32 {
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
			tmp_cards = utils.DelCard(tmp_cards, cards_copy[i], 0, 0)
			min_need_num = utils.Min(min_need_num, m.GetNeedHunInSub(tmp_cards, 0, 4)+1)
		} else {
			if i+2 == len_cards || cards_copy[i]%10 != cards_copy[i+2]%10 {
				if m.base_rule.Check2Combine(cards_copy[i], cards_copy[i+1]) {
					tmp_cards := utils.Copy(cards_copy)
					tmp_cards = utils.DelCard(tmp_cards, cards_copy[i], cards_copy[i+1], 0)
					min_need_num = utils.Min(min_need_num, m.GetNeedHunInSub(tmp_cards, 0, 4))
				}
			}
			if cards_copy[i]%10 != cards_copy[i+1]%10 {
				tmp_cards := utils.Copy(cards_copy)
				tmp_cards = utils.DelCard(tmp_cards, cards_copy[i], 0, 0)
				min_need_num = utils.Min(min_need_num, m.GetNeedHunInSub(tmp_cards, 0, 4)+1)
			}
		}
	}
	return min_need_num
}

func (m *HongZhongLaiZiRule) SumNeedHun(need_hun_arr []int32) int32 {
	var sum int32
	for _, num := range need_hun_arr {
		sum = sum + num
	}
	return sum
}

func (m *HongZhongLaiZiRule) GetBestComb(separate_results [5][]int32, need_hun_arr []int32, need_hun_with_eye_arr []int32) (int32, []int) {
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

func (m *HongZhongLaiZiRule) IsHu(cards []int32, hun_num int32, hun_card int32) bool {
	cards_copy := utils.Copy(cards)
	len_cards := len(cards_copy)
	if len_cards == 0 {
		if hun_num >= 2 {
			return true
		} else {
			return false
		}
	}

	if hun_num < m.base_rule.GetModNeedNum(len_cards, true) {
		return false
	}
	utils.SortCards(cards_copy, hun_card)
	for i := 0; i < len_cards; i++ {
		// 如果是最后一张牌
		if i+1 == len_cards {
			if hun_num > 0 {
				tmp_cards := utils.Copy(cards_copy)
				tmp_cards = utils.DelCard(tmp_cards, cards_copy[i], 0, 0)
				if m.GetNeedHunInSub(tmp_cards, 0, 4) <= hun_num-1 {
					return true
				}
			}
		} else {
			if i+2 == len_cards || cards_copy[i]%10 != cards_copy[i+2]%10 {
				if m.base_rule.Check2Combine(cards_copy[i], cards_copy[i+1]) {
					tmp_cards := utils.Copy(cards_copy)
					tmp_cards = utils.DelCard(tmp_cards, cards_copy[i], cards_copy[i+1], 0)
					if m.GetNeedHunInSub(tmp_cards, 0, 4) <= hun_num {
						return true
					}
				}
			}
			if hun_num > 0 && cards_copy[i]%10 != cards_copy[i+1]%10 {
				tmp_cards := utils.Copy(cards_copy)
				tmp_cards = utils.DelCard(tmp_cards, cards_copy[i], 0, 0)
				if m.GetNeedHunInSub(tmp_cards, 0, 4) <= hun_num-1 {
					return true
				}
			}
		}
	}
	return false
}

func (m *HongZhongLaiZiRule) GetTingCards(player *proto.Player) ([]int32, []int32, map[int32]interface{}) {
	result := make(map[int32]interface{})
	separate_results := utils.SeparateCards(player.Cards, player.HunCard)
	var need_hun_arr []int32          // 每个分类需要混的数组
	var need_hun_with_eye_arr []int32 // 每个将分类需要混的数组
	cur_hun_num := int32(len(separate_results[0]))
	for _, cards := range separate_results[1:] {
		need_hun_arr = append(need_hun_arr, m.GetNeedHunInSub(cards, 0, 4))
		need_hun_with_eye_arr = append(need_hun_with_eye_arr, m.GetNeedHunInSubWithEye(cards, 4))
	}
	need_num, index := m.GetBestComb(separate_results, need_hun_arr, need_hun_with_eye_arr)
	if cur_hun_num-need_num >= 2 {
		result[1] = NewTing(1)
		return need_hun_arr, need_hun_with_eye_arr, result
	}
	if cur_hun_num-m.SumNeedHun(need_hun_arr) > 0 {
		result[2] = NewTing(2)
		return need_hun_arr, need_hun_with_eye_arr, result
	}
	if need_num > cur_hun_num+1 {
		return need_hun_arr, need_hun_with_eye_arr, result
	}
	log.Debug("uid:%v separate_results:%v", player.Uid, separate_results)
	log.Debug("uid:%v need_hun_arr:%v, need_hun_with_eye_arr:%v index:%v", player.Uid, need_hun_arr, need_hun_with_eye_arr, index)
	var cache_index []int32
	for _, i := range index {
		begin, end := utils.SearchRange(separate_results[i+1])
		for card := begin; card < end+1; card++ {
			if ok := result[card]; ok != nil {
				continue
			}
			tmp_cards := utils.Copy(separate_results[i+1])
			tmp_cards = append(tmp_cards, card)
			utils.SortCards(tmp_cards, player.HunCard)
			if m.IsHu(tmp_cards, need_hun_with_eye_arr[i]-1, player.HunCard) {
				result[card] = NewTing(card)
			}
		}
		for j := 0; j < 4; j++ {
			if j != i && len(separate_results[j+1]) != 0 && !utils.Contain(cache_index, int32(j)) {
				cache_index = append(cache_index, int32(j))
				begin, end := utils.SearchRange(separate_results[j+1])
				for card := begin; card < end+1; card++ {
					if ok := result[card]; ok != nil {
						continue
					}
					tmp_cards := utils.Copy(separate_results[j+1])
					tmp_cards = append(tmp_cards, card)
					utils.SortCards(tmp_cards, player.HunCard)
					if m.GetNeedHunInSub(tmp_cards, 0, 4) <= need_hun_arr[j]-1 {
						result[card] = NewTing(card)
					}
				}
			}
		}
	}
	return need_hun_arr, need_hun_with_eye_arr, result
}
