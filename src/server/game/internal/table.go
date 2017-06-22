package internal

import (
	"errors"
	"github.com/jxbdlut/leaf/gate"
	"github.com/jxbdlut/leaf/log"
	"math/rand"
	"server/game/area"
	"server/utils"
	"server/proto"
	"server/userdata"
	"time"
)

type Table struct {
	tid         uint32
	tableType   proto.CreateTableReq_TableType
	players     []*Player
	rule        area.Rule
	play_count  uint32
	play_turn   int
	left_cards  []int32
	drop_cards  []int32
	win_player  *Player
	fan_card    int32
	hun_card    int32
	round       int
	drop_record map[uint64][]int32
	avail_count int
}

func NewTable(tid uint32, tableType proto.CreateTableReq_TableType) *Table {
	t := new(Table)
	t.tid = tid
	t.tableType = tableType
	t.play_turn = 0
	t.fan_card = 0
	t.hun_card = 0
	t.drop_record = make(map[uint64][]int32)
	if tableType == proto.CreateTableReq_TableRobot {
		t.avail_count = 1
	} else if tableType == proto.CreateTableReq_TableNomal {
		t.avail_count = 10
	}
	return t
}

func (t *Table) Clear() {
	t.play_turn = 0
	t.left_cards = append(t.left_cards[:0], t.left_cards[:0]...)
	t.win_player = nil
	t.fan_card = 0
	t.hun_card = 0
	t.round = 0
	for _, player := range t.players {
		delete(t.drop_record, player.uid)
	}
}

func (t *Table) GetPlayerIndex(uid uint64) (int, error) {
	for index, player := range t.players {
		if player.uid == uid {
			return index, nil
		}
	}
	return -1, errors.New("not in table")
}

func (t *Table) GetPlayer(uid uint64) (*Player, error) {
	for _, player := range t.players {
		if player.uid == uid {
			return player, nil
		}
	}
	return nil, errors.New("not in table")
}

func (t *Table) AddAgent(agent gate.Agent, master bool) (int, error) {
	if len(t.players) < 4 {
		uid := agent.UserData().(*userdata.UserData).Uid
		player := NewPlayer(agent, uid)
		player.SetMaster(master)
		player.SetTable(t)
		player.SetOnline(true)
		MapUidPlayer[uid] = player
		t.players = append(t.players, player)
		return len(t.players), nil
	} else {
		return 0, errors.New("this table is full!")
	}
}

func (t *Table) RemoveAgent(player *Player) error {
	uid := player.uid
	if index, err := t.GetPlayerIndex(uid); err == nil {
		player.agent.Destroy()
		t.players = append(t.players[:index], t.players[index+1:]...)
		delete(MapUidPlayer, uid)
		return nil
	}
	return errors.New("agent not in table")
}

func (t *Table) OfflineAgent(agent gate.Agent) error {
	uid := agent.UserData().(*userdata.UserData).Uid
	if index, err := t.GetPlayerIndex(uid); err == nil {
		t.players[index].online = false
		return nil
	}
	return errors.New("agent not in table")
}

func (t *Table) Broadcast(msg interface{}) {
	for _, player := range t.players {
		player.Send(msg)
	}
}

func (t *Table) BroadcastExceptMe(msg interface{}, uid uint64) {
	for _, player := range t.players {
		if player.uid != uid {
			player.Send(msg)
		}
	}
}

func (t *Table) Shuffle() {
	each_cards := []int32{101, 102, 103, 104, 105, 106, 107, 108, 109, 201, 202, 203, 204, 205, 206, 207, 208, 209}
	wind_cards := []int32{301, 302, 303, 304, 305, 306, 307, 308, 309, 401, 402, 403, 404, 405, 406, 407}
	if t.rule.HasWind() {
		each_cards = append(each_cards, wind_cards...)
	}
	var all_cards []int32
	for i := 0; i < 4; i++ {
		all_cards = append(all_cards, each_cards[:]...)
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for len(all_cards) > 0 {
		index := r.Intn(len(all_cards))
		t.left_cards = append(t.left_cards, all_cards[index])
		all_cards = append(all_cards[:index], all_cards[index+1:]...)
	}

	t.drop_cards = t.drop_cards[:0]
}

func (t *Table) Deal() {
	for _, player := range t.players {
		player.Clear()
		player.FeedCard(t.left_cards[:13])
		t.left_cards = append(t.left_cards[:0], t.left_cards[13:]...)
		log.Release("%v", player)
	}
	if t.rule.HasHun() {
		t.fan_card = t.DrawCard()
		t.hun_card = t.NextCard(t.fan_card)
	}

	for _, player := range t.players {
		player.Deal()
	}
}

func (t *Table) NextCard(card int32) int32 {
	m := t.fan_card / 100
	v := t.fan_card % 10
	if 0 < m && m < 4 {
		if v == 9 {
			v = 1
		} else {
			v = v + 1
		}
	} else if m == 4 {
		if v == 7 {
			v = 1
		} else {
			v = v + 1
		}
	}

	return 100*m + v
}

func (t *Table) DrawCard() int32 {
	card := t.left_cards[0]
	t.left_cards = append(t.left_cards[:0], t.left_cards[1:]...)
	return card
}

func (t *Table) DropRecord(uid uint64, dis_card int32) {
	t.drop_record[uid] = append(t.drop_record[uid], dis_card)
}

func (t *Table) CheckHu(disCard utils.DisCard) bool {
	pos, err := t.GetPlayerIndex(disCard.FromUid)
	if err != nil {
		log.Error("next_pos err:%v", err)
		return false
	}

	for i := 1; i < len(t.players); i++ {
		player := t.players[(pos+i)%len(t.players)]
		if player.CheckHu(disCard) {
			return true
		}
	}
	return false
}

func (t *Table) DisCard(disCard utils.DisCard) {
	pos, err := t.GetPlayerIndex(disCard.FromUid)
	if err != nil {
		log.Error("next_pos err:%v", err)
		return
	}

	for i := 1; i < len(t.players); i++ {
		player := t.players[(pos+i)%len(t.players)]
		if player.CheckHu(disCard) {
			return
		}
	}

	t.DropRecord(disCard.FromUid, disCard.Card)

	for i := 1; i < len(t.players); i++ {
		player := t.players[(pos+i)%len(t.players)]

		if disCard, ok := player.CheckGangOrPong(disCard); ok {
			pos, err := t.GetPlayerIndex(player.uid)
			if err != nil {
				log.Error("GetPlayerIndex err:", err)
				return
			}
			t.play_turn = (pos + 1) % len(t.players)
			//dis_card等于0的情况是杠上开花
			if disCard.Card == 0 {
				return
			}
			t.DisCard(disCard)
			return
		}
	}

	player := t.players[t.play_turn]
	if disCard, ok := player.CheckEat(disCard); ok {
		t.play_turn = (t.play_turn + 1) % len(t.players)
		t.DisCard(disCard)
		return
	}
}

func (t *Table) GetModNeedNum(len int, eye bool) *Ting {
	var need_hun_arr []int
	if eye {
		need_hun_arr = []int{2, 1, 0}
	} else {
		need_hun_arr = []int{0, 2, 1}
	}

	ting := MaxTing(t)
	if len == 0 {
		ting.need_hun_num = 0
	} else {
		ting.need_hun_num = need_hun_arr[len%3]
	}
	return ting
}

func (t *Table) Check3Combine(card1 int32, card2 int32, card3 int32, ting *Ting) bool {
	m1, m2, m3 := card1/100, card2/100, card3/100

	if m1 != m2 || m1 != m3 {
		return false
	}
	v1, v2, v3 := card1%10, card2%10, card3%10
	if v1 == v2 && v2 == v3 {
		ting.kezi_count = ting.kezi_count + 1
		ting.AddHunNoEye(0, []int32{card1, card2, card3})
		return true
	}
	if m3 == 4 {
		return false
	}
	if v1+1 == v2 && v1+2 == v3 {
		ting.kezi_count = ting.shunzi_count + 1
		ting.AddHunNoEye(0, []int32{card1, card2, card3})
		return true
	}
	return false
}

func (t *Table) Check2Combine(card1 int32, card2 int32, ting *Ting) bool {
	if card1 == card2 {
		if t.IsJiang(card1) {
			ting.AddHunEye(0, []int32{card1, card2})
			return true
		} else {
			return false
		}
	}
	return false
}

func (t *Table) GetNeedHunInSub(sub_cards []int32, min_need *Ting, max_need *Ting) *Ting {
	min_ting, max_ting := min_need.Copy(), max_need.Copy()
	if max_ting.need_hun_num == 0 {
		return max_ting
	}

	len_sub_cards := len(sub_cards)
	ting1 := min_ting.Copy()
	if ting1.AddTing(t.GetModNeedNum(len_sub_cards, false)).Bigger(max_ting) {
		return max_ting
	}

	if len_sub_cards == 0 {
		return GetMin(min_ting, max_ting)
	} else if len_sub_cards == 1 {
		return GetMin(min_ting.AddHunNoEye(2, []int32{sub_cards[0]}).AddNum(1), max_ting)
	} else if len_sub_cards == 2 {
		m, v0, v1 := sub_cards[0]/100, sub_cards[0]%10, sub_cards[1]%10
		if m == 4 {
			if v0 == v1 {
				return GetMin(min_ting.AddHunNoEye(1, []int32{sub_cards[0], sub_cards[1]}).AddKezi(1), max_ting)
			}
		} else if v1-v0 < 3 {
			return GetMin(min_ting.AddHunNoEye(1, []int32{sub_cards[0], sub_cards[1]}).AddShunzi(1), max_ting)
		}
	} else if len_sub_cards >= 3 {
		m, v0 := sub_cards[0]/100, sub_cards[0]%10

		// 第一个和后两个一铺
		for i := 1; i < len_sub_cards; i++ {
			ting1 := min_ting.Copy().AddTing(t.GetModNeedNum(len_sub_cards-3, false))
			if ting1.BiggerOrEqual(max_ting) {
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
				if t.Check3Combine(tmp1, tmp2, tmp3, min_ting) {
					tmp_cards := utils.Copy(sub_cards)
					tmp_cards = utils.DelCard(tmp_cards, tmp1, tmp2, tmp3)
					max_ting = t.GetNeedHunInSub(tmp_cards, min_ting, max_ting)
				}
			}
		}

		// 第一个和第二个一铺
		v1 := sub_cards[1] % 10
		ting2 := min_ting.Copy().AddTing(t.GetModNeedNum(len_sub_cards-2, false))
		if ting2.AddNum(1).Smaller(max_ting) {
			if m == 4 {
				if v0 == v1 {
					tmp_cards := utils.Copy(sub_cards[2:])
					max_ting = t.GetNeedHunInSub(tmp_cards, min_ting.AddHunNoEye(1, []int32{sub_cards[0], sub_cards[1]}).AddKezi(1), max_ting)
				}
			} else {
				for i := 1; i < len_sub_cards; i++ {
					ting4 := min_ting.Copy()
					if ting4.AddTing(t.GetModNeedNum(len_sub_cards-2, false)).AddNum(1).BiggerOrEqual(max_ting) {
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
						max_ting = t.GetNeedHunInSub(tmp_cards, min_ting.AddHunNoEye(1, []int32{tmp1, tmp2}), max_ting)
						if mius >= 1 {
							break
						}
						if mius == 0 {
							max_ting.AddKezi(1)
						}
					} else {
						break
					}
				}
			}
		}
		// 第一个自己一铺
		if min_ting.AddTing(t.GetModNeedNum(len_sub_cards-1, false)).AddNum(2).Smaller(max_ting) {
			tmp_cards := utils.Copy(sub_cards[1:])
			max_ting = t.GetNeedHunInSub(tmp_cards, min_ting.AddHunNoEye(2, []int32{sub_cards[0]}).AddKezi(1), max_ting)
		}
	}
	return max_ting
}

func (t *Table) IsJiang(card int32) bool {
	return t.rule.IsJiang(card)
}

func (t *Table) GetNeedHunInSubWithEye(cards []int32, max_need *Ting) *Ting {
	// 拷贝
	ting := MaxTing(t)
	cards_copy := utils.Copy(cards)
	len_cards := len(cards_copy)
	if len_cards == 0 {
		ting.need_hun_num = 2
		return ting
	}
	min_ting := max_need.Copy()
	if min_ting.Smaller(t.GetModNeedNum(len_cards, true)) {
		return min_ting
	}
	for i := 0; i < len_cards; i++ {
		if i == len_cards-1 { // 如果是最后一张牌
			tmp_cards := utils.Copy(cards_copy)
			if t.IsJiang(cards_copy[i]) {
				tmp_cards = utils.DelCard(tmp_cards, cards_copy[i], 0, 0)
				min_ting = GetMin(min_ting, t.GetNeedHunInSub(tmp_cards, Minting(t), MaxTing(t)).Copy().AddHunEye(1, []int32{cards_copy[i]}))
			} else {
				min_ting = GetMin(min_ting, t.GetNeedHunInSub(tmp_cards, Minting(t), MaxTing(t)).Copy().AddHunEye(2, []int32{}))
			}
		} else {
			if i+2 == len_cards || cards_copy[i]%10 != cards_copy[i+2]%10 {
				tmp_cards := utils.Copy(cards_copy)
				if t.Check2Combine(cards_copy[i], cards_copy[i+1], min_ting) {
					tmp_cards = utils.DelCard(tmp_cards, cards_copy[i], cards_copy[i+1], 0)
					min_ting = GetMin(min_ting, t.GetNeedHunInSub(tmp_cards, Minting(t), MaxTing(t)))
				} else {
					min_ting = GetMin(min_ting, t.GetNeedHunInSub(tmp_cards, Minting(t), MaxTing(t)).Copy().AddHunEye(2, []int32{}))
				}
			}
			if cards_copy[i]%10 != cards_copy[i+1]%10 {
				tmp_cards := utils.Copy(cards_copy)
				if t.IsJiang(tmp_cards[i]) {
					tmp_cards = utils.DelCard(tmp_cards, cards_copy[i], 0, 0)
					min_ting = GetMin(min_ting, t.GetNeedHunInSub(tmp_cards, Minting(t), MaxTing(t)).AddHunEye(1, []int32{cards_copy[i]}))
				} else if t.IsJiang(tmp_cards[i+1]) {
					tmp_cards = utils.DelCard(tmp_cards, cards_copy[i+1], 0, 0)
					min_ting = GetMin(min_ting, t.GetNeedHunInSub(tmp_cards, Minting(t), MaxTing(t)).AddHunEye(1, []int32{cards_copy[i+1]}))
				}
			}
		}
	}
	return min_ting
}

func (t *Table) GetBestComb(separate_results [5][]int32, need_hun_arr []*Ting, need_hun_with_eye_arr []*Ting) (*Ting, []int) {
	min_need_num := MaxTing(t)
	sum_num := t.SumNeedHun(need_hun_arr)
	var result []int

	for i := 0; i < 4; i++ {
		tmp_sum := sum_num.Copy()
		need_num := tmp_sum.SubTing(need_hun_arr[i]).AddTing(need_hun_with_eye_arr[i])
		if need_num.Smaller(min_need_num) && len(separate_results[i+1]) != 0 {
			min_need_num = need_num
			result = append(result[:0], result[:0]...)
			result = append(result, i)
		} else if need_num.Equal(min_need_num) && len(separate_results[i+1]) != 0 {
			result = append(result, i)
		}
	}
	return min_need_num, result
}

func (t *Table) SumNeedHun(need_hun_arr []*Ting) *Ting {
	sum := Minting(t)
	for _, t := range need_hun_arr {
		sum.AddTing(t)
	}
	return sum
}

func (t *Table) SumKeZiNum(need_hun_arr []*Ting) int {
	sum := 0
	for _, t := range need_hun_arr {
		sum = sum + t.kezi_count
	}
	return sum
}

func (t *Table) IsHu(cards []int32, hun_num_ting *Ting) bool {
	cards_copy := utils.Copy(cards)
	len_cards := len(cards_copy)
	if len_cards == 0 {
		if hun_num_ting.BiggerOrEqualNum(2) {
			return true
		} else {
			return false
		}
	}

	if hun_num_ting.Smaller(t.GetModNeedNum(len_cards, true)) {
		return false
	}
	utils.SortCards(cards_copy, t.hun_card)
	for i := 0; i < len_cards; i++ {
		// 如果是最后一张牌
		if i+1 == len_cards {
			if hun_num_ting.BiggerNum(0) {
				tmp_cards := utils.Copy(cards_copy)
				tmp_ting := hun_num_ting.Copy()
				if t.IsJiang(cards_copy[i]) {
					tmp_cards = utils.DelCard(tmp_cards, cards_copy[i], 0, 0)
					if t.GetNeedHunInSub(tmp_cards, Minting(t), MaxTing(t)).SmallerOrEqual(tmp_ting.SubNum(1)) {
						return true
					}
				} else {
					if t.GetNeedHunInSub(tmp_cards, Minting(t), MaxTing(t)).SmallerOrEqual(tmp_ting.SubNum(2)) {
						return true
					}
				}
			}
		} else {
			if i+2 == len_cards || cards_copy[i]%10 != cards_copy[i+2]%10 {
				tmp_cards := utils.Copy(cards_copy)
				tmp_ting := hun_num_ting.Copy()
				if t.Check2Combine(cards_copy[i], cards_copy[i+1], tmp_ting) {
					tmp_cards = utils.DelCard(tmp_cards, cards_copy[i], cards_copy[i+1], 0)
					if t.GetNeedHunInSub(tmp_cards, Minting(t), MaxTing(t)).SmallerOrEqual(tmp_ting) {
						return true
					}
				} else {
					if t.GetNeedHunInSub(tmp_cards, Minting(t), MaxTing(t)).SmallerOrEqual(tmp_ting.SubNum(2)) {
						return true
					}
				}
			}
			if hun_num_ting.BiggerNum(0) && cards_copy[i]%10 != cards_copy[i+1]%10 {
				tmp_cards := utils.Copy(cards_copy)
				tmp_ting := hun_num_ting.Copy()
				if t.IsJiang(cards_copy[i]) {
					tmp_cards = utils.DelCard(tmp_cards, cards_copy[i], 0, 0)
					if t.GetNeedHunInSub(tmp_cards, Minting(t), MaxTing(t)).SmallerOrEqual(tmp_ting.SubNum(1)) {
						return true
					}
				} else {
					if t.GetNeedHunInSub(tmp_cards, Minting(t), MaxTing(t)).SmallerOrEqual(tmp_ting.SubNum(2)) {
						return true
					}
				}
			}
		}
	}
	return false
}

func (t *Table) HasDuiJiang(cards []int32) bool {
	var cache []int32
	for _, card := range cards {
		if utils.Contain(cache, card) {
			continue
		}
		cache = append(cache, card)
		if utils.Count(cards, card) >= 2 && t.rule.IsJiang(card) {
			return true
		}
	}
	return false
}

func (t *Table) GetTingCards(p *Player) map[int32]interface{} {
	result := make(map[int32]interface{})
	separate_results := p.separate_result
	var need_hun_arr []*Ting          // 每个分类需要混的数组
	var need_hun_with_eye_arr []*Ting // 每个将分类需要混的数组
	cur_hun := NewTing(len(separate_results[0]), t)

	for _, cards := range separate_results[1:] {
		need_hun_arr = append(need_hun_arr, t.GetNeedHunInSub(cards, Minting(t), MaxTing(t)))
		need_hun_with_eye_arr = append(need_hun_with_eye_arr, t.GetNeedHunInSubWithEye(cards, MaxTing(t)))
	}
	need_hun, index := t.GetBestComb(separate_results, need_hun_arr, need_hun_with_eye_arr)
	log.Debug("uid:%v, separate_results:%v", p.uid, separate_results)
	log.Debug("uid:%v, need_hun_arr:%v, need_hun_with_eye_arr:%v index:%v", p.uid, need_hun_arr, need_hun_with_eye_arr, index)
	if cur_hun.Copy().SubTing(need_hun).BiggerOrEqualNum(2) {
		result[1] = &Ting{tengkong:true}
		return result
	}
	if cur_hun.Copy().SubTing(t.SumNeedHun(need_hun_arr)).BiggerNum(0) {
		result[2] = &Ting{piaojiang:true}
	}
	if need_hun.Bigger(cur_hun.Copy().AddNum(1)) {
		return result
	}
	var cache_index []int32
	for _, i := range index {
		begin, end := utils.SearchRange(separate_results[i+1])
		for card := begin; card < end+1; card++ {
			if ok := result[card]; ok != nil {
				continue
			}
			if ok := result[2]; ok != nil && need_hun_arr[i].EqualNum(0) && !t.HasDuiJiang(separate_results[i+1]) {
				continue
			}
			tmp_cards := utils.Copy(separate_results[i+1])
			tmp_cards = append(tmp_cards, card)
			utils.SortCards(tmp_cards, t.hun_card)
			//tmp_ting := need_hun_with_eye_arr[i].Copy()
			ting := t.GetNeedHunInSubWithEye(tmp_cards, MaxTing(t))
			if ting.SmallerOrEqual(need_hun_with_eye_arr[i].Copy().SubNum(1)) {
				ting.card = card
				if ting.IsPiaoMen() {
					result[3] = &Ting{piaomen:true}
				}
				result[card] = ting
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
					utils.SortCards(tmp_cards, t.hun_card)
					ting := t.GetNeedHunInSub(tmp_cards, Minting(t), MaxTing(t))
					if ting.SmallerOrEqual(need_hun_arr[j].Copy().SubNum(1)) {
						ting.card = card
						if ting.IsPiaoMen() {
							result[3] = &Ting{piaomen:true}
						}
						result[card] = ting
					}
				}
			}
		}
	}

	return result
}

func (t *Table) GetOnlineNum() int {
	num := 0
	log.Debug("tid:%v players num:%v", t.tid, len(t.players))
	for _, player := range t.players {
		if player.online && !player.isRobot {
			num++
		}
	}
	return num
}

func (t *Table) TableOperat(tableOperat proto.TableOperat) bool {
	rsp := make(chan int, 4)
	result := make([]bool, 4)
	ret := true
	for i := range t.players {
		go func(index int) {
			player := t.players[index]
			ok := player.CheckTableOperat(tableOperat)
			result[index] = ok
			rsp <- index
		}(i)
	}
	for i := 0; i < 4; i++ {
		index := <-rsp
		player := t.players[index]
		tableOperatMsg := proto.TableOperatMsg{Uid: player.uid, Type: tableOperat, OK: result[index]}
		t.players[index].BoardCastMsg(&tableOperatMsg)
		ret = ret && result[index]
	}
	// todo 需要添加超时
	return ret
}

func (t *Table) Play() {
	t.play_count++
	t.avail_count--
	t.Shuffle()
	t.Deal()
	for len(t.left_cards) > 10 && t.win_player == nil && len(t.players) == 4 {
		player := t.players[t.play_turn]
		t.play_turn = (t.play_turn + 1) % len(t.players)
		discard := player.Draw(utils.DisCard_Mo)
		if discard.Card != 0 {
			if discard.DisType == utils.DisCard_BuGang {
				t.DisCard(discard)
				discard = player.Draw(utils.DisCard_SelfGang)
			}
			t.DisCard(discard)
			t.round += 1
		} else {
			t.win_player = player
		}
	}
	if t.win_player != nil {
		log.Release("tid:%v, uid :%v win the game, round:%v", t.tid, t.win_player.uid, t.round)
	} else {
		log.Release("tid:%v, 流局..., play_count:%v", t.tid, t.play_count)
	}
}

func (t *Table) waitPlayer() bool {
	if t.tableType == proto.CreateTableReq_TableNomal {
		if len(t.players) < 4 {
			return true
		} else {
			return false
		}
	} else if t.tableType == proto.CreateTableReq_TableRobot && t.GetOnlineNum() > 0 {
		//todo 等待恢复才能运行
		return false
	}
	return true
}

func (t *Table) Run() {
	for {
		if len(t.players) == 0 {
			break
		} else if t.waitPlayer() {
			log.Debug("tid:%v, waiting agent join, agent num:%v", t.tid, len(t.players))
			time.Sleep(time.Second)
			//todo
		} else {
			if !t.TableOperat(proto.TableOperat_TableStart) {
				break
			}
			t.Clear()
			t.Play()
			log.Debug("tid:%v, running, play_count:%v, players num:%v, left_cards num:%d", t.tid, t.play_count, len(t.players), len(t.left_cards))
			if t.avail_count == 0 {
				break
			}
			if !t.TableOperat(proto.TableOperat_TableContinue) {
				break
			}
			time.Sleep(1 * time.Millisecond)
		}

		//time.Sleep(50 * time.Millisecond)
		//now := time.Now().UTC()
		//if start.Add(100 * time.Second).Before(now) {
		//	log.Error("tid:%v, timeout, start:%v, now:%v", t.tid, start, now)
		//	break
		//}
	}
	for _, player := range t.players {
		t.RemoveAgent(player)
		delete(MapUidPlayer, player.uid)
	}
	delete(Tables, t.tid)
	log.Debug("tid:%v, is over", t.tid)

}
