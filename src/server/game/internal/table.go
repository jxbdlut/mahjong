package internal

import (
	"errors"
	"github.com/name5566/leaf/gate"
	"github.com/name5566/leaf/log"
	"math/rand"
	"server/mahjong"
	"server/userdata"
	"time"
)

type Table struct {
	tid         uint32
	players     []*Player
	play_count  uint32
	play_turn   int
	left_cards  []int32
	drop_cards  []int32
	win_player  *Player
	fan_card    int32
	hun_card    int32
	round       int
	drop_record map[*Player][]int32
	call_time   int
}

func NewTable(tid uint32) *Table {
	t := new(Table)
	t.tid = tid
	t.play_turn = 0
	t.drop_record = make(map[*Player][]int32)
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
		delete(t.drop_record, player)
	}
}

func GetIndex(cards []int32, card int32) (int, error) {
	for i, c := range cards {
		if c == card {
			return i, nil
		}
	}
	return -1, errors.New("not found")
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

func (t *Table) AddAgent(agent gate.Agent, master bool) error {
	if len(t.players) < 4 {
		uid := agent.UserData().(*userdata.UserData).Uid
		player := NewPlayer(agent, uid)
		player.SetMaster(master)
		player.SetTable(t)
		player.SetOnline(true)
		MapUidPlayer[uid] = player
		t.players = append(t.players, player)
		return nil
	} else {
		return errors.New("this table is full!")
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
		player.WriteMsg(msg)
	}
}

func (t *Table) BroadcastExceptMe(msg interface{}, uid uint64) {
	for _, player := range t.players {
		if player.uid != uid {
			player.WriteMsg(msg)
		}
	}
}

func (t *Table) Shuffle() {
	each_cards := []int32{101, 102, 103, 104, 105, 106, 107, 108, 109, 201, 202, 203, 204, 205, 206, 207, 208, 209,
		301, 302, 303, 304, 305, 306, 307, 308, 309, 401, 402, 403, 404, 405, 406, 407}
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
	t.fan_card = t.DrawCard()
	t.hun_card = t.NextCard(t.fan_card)

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

func (t *Table) DropRecord(p *Player, dis_card int32) {
	t.drop_record[p] = append(t.drop_record[p], dis_card)
}

func (t *Table) DisCard(p *Player, dis_card int32) {
	t.DropRecord(p, dis_card)
	next_pos, err := t.GetPlayerIndex(p.uid)
	if err != nil {
		log.Error("next_pos err:%v", err)
		return
	}
	for i := 0; i < len(t.players); i++ {
		player := t.players[(next_pos+i)%len(t.players)]
		if player.CheckHu(dis_card, false) {
			player.Hu(dis_card, false)
			t.win_player = player
			return
		}
	}

	for i := 0; i < len(t.players); i++ {
		player := t.players[(t.play_turn+i)%len(t.players)]
		if dis_card, count := player.CheckPong(dis_card); count > 0 {
			pos, err := t.GetPlayerIndex(player.uid)
			if err != nil {
				log.Error("GetPlayerIndex err:", err)
				return
			}
			t.play_turn = (pos + 1) % len(t.players)
			t.DisCard(player, dis_card)
			return
		}
	}

	player := t.players[t.play_turn]
	if dis_card, ok := player.CheckEat(dis_card); ok {
		t.play_turn = (t.play_turn + 1) % len(t.players)
		t.DisCard(player, dis_card)
		return
	}
}

func (t *Table) GetModNeedNum(len int, eye bool) int {
	var need_hun_arr []int
	if eye {
		need_hun_arr = []int{2, 1, 0}
	} else {
		need_hun_arr = []int{0, 2, 1}
	}

	if len == 0 {
		return 0
	} else {
		return need_hun_arr[len%3]
	}
}

func min(a int, b int) int {
	if a < b {
		return a
	} else {
		return b
	}
}

func (t *Table) Check3Combine(card1 int32, card2 int32, card3 int32) bool {
	m1, m2, m3 := card1/100, card2/100, card3/100

	if m1 != m2 || m1 != m3 {
		return false
	}
	v1, v2, v3 := card1%10, card2%10, card3%10
	if v1 == v2 && v2 == v3 {
		return true
	}
	if m3 == 4 {
		return false
	}
	if v1+1 == v2 && v1+2 == v3 {
		return true
	}
	return false
}

func (t *Table) Check2Combine(card1 int32, card2 int32) bool {
	if card1 == card2 {
		return true
	}
	return false
}

func (t *Table) Remove(cards []int32, card1 int32, card2 int32, card3 int32) []int32 {
	if card1 != 0 {
		if i, err := GetIndex(cards, card1); err == nil {
			cards = append(cards[:i], cards[i+1:]...)
		}
	}
	if card2 != 0 {
		if i, err := GetIndex(cards, card2); err == nil {
			cards = append(cards[:i], cards[i+1:]...)
		}
	}
	if card3 != 0 {
		if i, err := GetIndex(cards, card3); err == nil {
			cards = append(cards[:i], cards[i+1:]...)
		}
	}
	return cards
}

func (t *Table) GetNeedHunInSub(sub_cards []int32, hun_num int, need_hun_count int) int {
	t.call_time++
	if need_hun_count == 0 {
		return need_hun_count
	}

	len_sub_cards := len(sub_cards)
	if hun_num+t.GetModNeedNum(len_sub_cards, false) >= need_hun_count {
		return need_hun_count
	}

	if len_sub_cards == 0 {
		return min(hun_num, need_hun_count)
	} else if len_sub_cards == 1 {
		return min(hun_num+2, need_hun_count)
	} else if len_sub_cards == 2 {
		m, v0, v1 := sub_cards[0]/100, sub_cards[0]%10, sub_cards[1]%10
		if m == 4 {
			if v0 == v1 {
				return min(hun_num+1, need_hun_count)
			}
		} else if v1-v0 < 3 {
			return min(hun_num+1, need_hun_count)
		}
	} else if len_sub_cards >= 3 {
		m, v0 := sub_cards[0]/100, sub_cards[0]%10

		// 第一个和后两个一铺
		for i := 1; i < len_sub_cards; i++ {
			if hun_num+t.GetModNeedNum(len_sub_cards-3, false) >= need_hun_count {
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
				if t.Check3Combine(tmp1, tmp2, tmp3) {
					tmp_cards := mahjong.Copy(sub_cards)
					tmp_cards = t.Remove(tmp_cards, tmp1, tmp2, tmp3)
					need_hun_count = t.GetNeedHunInSub(tmp_cards, hun_num, need_hun_count)
				}
			}
		}

		// 第一个和第二个一铺
		v1 := sub_cards[1] % 10
		if hun_num+t.GetModNeedNum(len_sub_cards-2, false)+1 < need_hun_count {
			if m == 4 {
				if v0 == v1 {
					tmp_cards := mahjong.Copy(sub_cards[2:])
					need_hun_count = t.GetNeedHunInSub(tmp_cards, hun_num+1, need_hun_count)
				}
			} else {
				for i := 1; i < len_sub_cards; i++ {
					if hun_num+t.GetModNeedNum(len_sub_cards-2, false)+1 >= need_hun_count {
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
						tmp_cards := mahjong.Copy(sub_cards)
						tmp_cards = t.Remove(tmp_cards, tmp1, tmp2, 0)
						need_hun_count = t.GetNeedHunInSub(tmp_cards, hun_num+1, need_hun_count)
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
		if hun_num+t.GetModNeedNum(len_sub_cards-1, false)+2 < need_hun_count {
			tmp_cards := mahjong.Copy(sub_cards[1:])
			need_hun_count = t.GetNeedHunInSub(tmp_cards, hun_num+2, need_hun_count)
		}
	}
	return need_hun_count
}

func (t *Table) GetNeedHunInSubWithEye(cards []int32, min_need_num int) int {
	// 拷贝
	cards_copy := mahjong.Copy(cards)
	len_cards := len(cards_copy)
	if len_cards == 0 {
		return 2
	}
	if min_need_num < t.GetModNeedNum(len_cards, true) {
		return min_need_num
	}
	for i := 0; i < len_cards; i++ {
		if i == len_cards-1 { // 如果是最后一张牌
			tmp_cards := mahjong.Copy(cards_copy)
			tmp_cards = t.Remove(tmp_cards, cards_copy[i], 0, 0)
			min_need_num = min(min_need_num, t.GetNeedHunInSub(tmp_cards, 0, 4)+1)
		} else {
			if i+2 == len_cards || cards_copy[i]%10 != cards_copy[i+2]%10 {
				if t.Check2Combine(cards_copy[i], cards_copy[i+1]) {
					tmp_cards := mahjong.Copy(cards_copy)
					tmp_cards = t.Remove(tmp_cards, cards_copy[i], cards_copy[i+1], 0)
					min_need_num = min(min_need_num, t.GetNeedHunInSub(tmp_cards, 0, 4))
				}
			}
			if cards_copy[i]%10 != cards_copy[i+1]%10 {
				tmp_cards := mahjong.Copy(cards_copy)
				tmp_cards = t.Remove(tmp_cards, cards_copy[i], 0, 0)
				min_need_num = min(min_need_num, t.GetNeedHunInSub(tmp_cards, 0, 4)+1)
			}
		}
	}
	return min_need_num
}

func (t *Table) GetBestComb(separate_results [5][]int32, need_hun_arr []int, need_hun_with_eye_arr []int) (int, []int) {
	min_need_num := 5
	sum_num := t.SumNeedHun(need_hun_arr)
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

func (t *Table) SumNeedHun(need_hun_arr []int) int {
	var sum int
	for _, num := range need_hun_arr {
		sum = sum + num
	}
	return sum
}

func (t *Table) SearchRange(cards []int32) (int32, int32) {
	m := cards[0] / 100
	begin := int32(10)
	end := int32(0)
	for _, card := range cards {
		if card < begin {
			begin = card
		}
		if card > end {
			end = card
		}
	}
	if begin-2 < 100*m+1 {
		begin = 100*m + 1
	} else {
		begin = begin - 2
	}

	if m == 4 {
		if end+2 > 100*m+7 {
			end = 100*m + 7
		} else {
			end = end + 2
		}
	} else {
		if end+2 > 100*m+9 {
			end = 100*m + 9
		} else {
			end = end + 2
		}
	}

	return begin, end
}

func (t *Table) CanHu(cards []int32, hun_num int) bool {
	cards_copy := mahjong.Copy(cards)
	len_cards := len(cards_copy)
	if len_cards == 0 {
		if hun_num >= 2 {
			return true
		} else {
			return false
		}
	}

	if hun_num < t.GetModNeedNum(len_cards, true) {
		return false
	}
	mahjong.SortCards(cards_copy, t.hun_card)
	for i := 0; i < len_cards; i++ {
		// 如果是最后一张牌
		if i+1 == len_cards {
			if hun_num > 0 {
				tmp_cards := mahjong.Copy(cards_copy)
				tmp_cards = t.Remove(tmp_cards, cards_copy[i], 0, 0)
				if t.GetNeedHunInSub(tmp_cards, 0, 4) <= hun_num-1 {
					return true
				}
			}
		} else {
			if i+2 == len_cards || cards_copy[i]%10 != cards_copy[i+2]%10 {
				if t.Check2Combine(cards_copy[i], cards_copy[i+1]) {
					tmp_cards := mahjong.Copy(cards_copy)
					tmp_cards = t.Remove(tmp_cards, cards_copy[i], cards_copy[i+1], 0)
					if t.GetNeedHunInSub(tmp_cards, 0, 4) <= hun_num {
						return true
					}
				}
			}
			if hun_num > 0 && cards_copy[i]%10 != cards_copy[i+1]%10 {
				tmp_cards := mahjong.Copy(cards_copy)
				tmp_cards = t.Remove(tmp_cards, cards_copy[i], 0, 0)
				if t.GetNeedHunInSub(tmp_cards, 0, 4) <= hun_num-1 {
					return true
				}
			}
		}
	}
	return false
}

func (t *Table) Contain(elems []int, elem int) bool {
	for _, e := range elems {
		if e == elem {
			return true
		}
	}
	return false
}

func (t *Table) GetTingCards(p *Player) map[int32]interface{} {
	result := make(map[int32]interface{})
	separate_results := p.separate_result
	var need_hun_arr []int          // 每个分类需要混的数组
	var need_hun_with_eye_arr []int // 每个将分类需要混的数组
	cur_hun_num := len(separate_results[0])

	for _, cards := range separate_results[1:] {
		need_hun_arr = append(need_hun_arr, t.GetNeedHunInSub(cards, 0, 4))
		need_hun_with_eye_arr = append(need_hun_with_eye_arr, t.GetNeedHunInSubWithEye(cards, 4))
	}
	need_num, index := t.GetBestComb(separate_results, need_hun_arr, need_hun_with_eye_arr)
	if cur_hun_num-need_num >= 2 {
		result[1] = 1
		return result
	}
	if cur_hun_num-t.SumNeedHun(need_hun_arr) > 0 {
		result[2] = 2
		return result
	}
	if need_num > cur_hun_num+1 {
		return result
	}
	//log.Debug("uid:%v separate_results:%v", p.uid, separate_results)
	//log.Debug("uid:%v need_hun_arr:%v, need_hun_with_eye_arr:%v index:%v", p.uid, need_hun_arr, need_hun_with_eye_arr, index)
	var cache_index []int
	for _, i := range index {
		begin, end := t.SearchRange(separate_results[i+1])
		for card := begin; card < end+1; card++ {
			if ok := result[card]; ok != nil {
				continue
			}
			tmp_cards := mahjong.Copy(separate_results[i+1])
			tmp_cards = append(tmp_cards, card)
			mahjong.SortCards(tmp_cards, t.hun_card)
			if t.CanHu(tmp_cards, need_hun_with_eye_arr[i]-1) {
				result[card] = card
			}
		}
		for j := 0; j < 4; j++ {
			if j != i && len(separate_results[j+1]) != 0 && !t.Contain(cache_index, j) {
				cache_index = append(cache_index, j)
				begin, end := t.SearchRange(separate_results[j+1])
				for card := begin; card < end+1; card++ {
					if ok := result[card]; ok != nil {
						continue
					}
					tmp_cards := mahjong.Copy(separate_results[j+1])
					tmp_cards = append(tmp_cards, card)
					mahjong.SortCards(tmp_cards, t.hun_card)
					if t.GetNeedHunInSub(tmp_cards, 0, 4) <= need_hun_arr[j]-1 {
						result[card] = card
					}
				}
			}
		}
	}

	return result
}

func (t *Table)GetOnlineNum() int {
	num := 0
	for _, player := range t.players {
		if player.online {
			num++
		}
	}
	return num
}

func (t *Table) Play() {
	t.play_count++
	t.Shuffle()
	t.Deal()
	for len(t.left_cards) > 10 && t.win_player == nil && len(t.players) == 4 {
		player := t.players[t.play_turn]
		t.play_turn = (t.play_turn + 1) % len(t.players)
		discard := player.Draw(false)
		if discard != 0 {
			t.DisCard(player, discard)
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

func (t *Table) Run() {
	start := time.Now().UTC()
	for {
		if len(t.players) == 0 {
			break
		} else if len(t.players) < 4 {
			log.Debug("tid:%v, waiting agent join, agent num:%v", t.tid, len(t.players))
			time.Sleep(time.Second)
			//todo
		} else if t.GetOnlineNum() > 3 {
			t.Clear()
			t.Play()
			log.Debug("tid:%v, running, play_count:%v, players num:%v, left_cards num:%d", t.tid, t.play_count, len(t.players), len(t.left_cards))
			time.Sleep(1 * time.Millisecond)
			start = time.Now().UTC()
		}
		time.Sleep(50 * time.Millisecond)
		now := time.Now().UTC()
		if start.Add(100 * time.Second).Before(now) {
			log.Error("tid:%v, timeout, start:%v, now:%v", t.tid, start, now)
			break
		}
	}
	for _, player := range t.players {
		t.RemoveAgent(player)
	}
	log.Debug("tid:%v, is over", t.tid)
}
