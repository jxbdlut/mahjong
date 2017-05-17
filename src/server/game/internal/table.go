package internal

import (
	"errors"
	"time"

	"github.com/name5566/leaf/gate"
	"github.com/name5566/leaf/log"
	"math/rand"
	"server/userdata"
)

type Table struct {
	tid         uint32
	players     []*Player
	play_count  uint32
	play_turn   int
	left_cards  []int
	drop_cards  []int
	win_player  *Player
	fan_card    int
	hun_card    int
	round       int
	drop_record map[*Player][]int
}

func NewTable(tid uint32) *Table {
	t := new(Table)
	t.tid = tid
	t.play_turn = 0
	t.drop_record = make(map[*Player][]int)
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
		player := NewPlayer(agent, agent.UserData().(*userdata.UserData).Uid)
		player.SetMaster(master)
		player.SetTable(t)
		player.SetOnline(true)
		t.players = append(t.players, player)
		return nil
	} else {
		return errors.New("this table is full!")
	}
}

func (t *Table) RemoveAgent(agent gate.Agent) error {
	uid := agent.UserData().(*userdata.UserData).Uid
	if index, err := t.GetPlayerIndex(uid); err == nil {
		t.players = append(t.players[:index], t.players[index+1:]...)
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

func (t *Table) SeparateCards(cards []int) [5][]int {
	var result = [5][]int{}
	for _, card := range cards {
		m := int(0)
		if int(card) != t.hun_card {
			m = card / 100
		} else {
			m = 0
		}
		result[m] = append(result[m], int(card))
	}
	return result
}

func (t *Table) Shuffle() {
	each_cards := []int{101, 102, 103, 104, 105, 106, 107, 108, 109, 201, 202, 203, 204, 205, 206, 207, 208, 209,
		301, 302, 303, 304, 305, 306, 307, 308, 309, 401, 402, 403, 404, 405, 406, 407}
	var all_cards []int
	for i := 0; i < 4; i++ {
		all_cards = append(all_cards, each_cards[:]...)
	}

	for len(all_cards) > 0 {
		index := rand.Intn(len(all_cards))
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

		log.Debug("uid:%v cards:%v", player.uid, player.cards)
	}
	t.fan_card = t.DrawCard()
	t.hun_card = t.NextCard(t.fan_card)
}

func (t *Table) NextCard(card int) int {
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

func (t *Table) DrawCard() int {
	card := t.left_cards[0]
	t.left_cards = append(t.left_cards[:0], t.left_cards[1:]...)
	return card
}

func (t *Table) DropRecord(p *Player, dis_card int) {
	t.drop_record[p] = append(t.drop_record[p], dis_card)
}

func (t *Table) DisCard(p *Player, dis_card int) {
	t.DropRecord(p, dis_card)
	for index := range t.players {
		player := t.players[(t.play_turn+index)%len(t.players)]
		if player.CheckHu(dis_card) {
			return
		}
	}

	for index := range t.players {
		player := t.players[(t.play_turn+index)%len(t.players)]
		if dis_card, count := player.CheckPong(dis_card); count > 0 {
			pos, err := t.GetPlayerIndex(player.uid)
			if err != nil {
				log.Error("GetPlayerIndex err:%v", err)
				return
			}
			t.play_turn = (pos + 1) % len(t.players)
			t.DisCard(player, dis_card)
			return
		}
	}

	player := t.players[t.play_turn]
	if eat, dis_card, ok := player.CheckEat(dis_card); ok {
		player.Eat(eat)
		t.DisCard(player, dis_card)
		return
	}
}

func (t *Table) Play() {
	t.play_count++
	t.Shuffle()
	t.Deal()
	for len(t.left_cards) > 10 && t.win_player == nil && len(t.players) > 0 {
		player := t.players[t.play_turn]
		t.play_turn = (t.play_turn + 1) % len(t.players)
		discard := player.draw()
		if discard != 0 {
			t.DisCard(player, discard)
			t.round += 1
		}
		time.Sleep(2 * time.Second)
	}
}

func (t *Table) Run() {
	for {
		if len(t.players) == 0 {
			log.Debug("table:%v is over", t.tid)
			break
		} else if len(t.players) < 4 {
			log.Debug("waiting agent join, table id:%v agent num:%v", t.tid, len(t.players))
		} else {
			t.Clear()
			t.Play()
		}
		log.Debug("Table tid:%v runing, play_count:%v, players num:%v, left_cards num:%d", t.tid, t.play_count, len(t.players), len(t.left_cards))
		time.Sleep(time.Second)
	}
}
