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
	tid        uint32
	players    []*Player
	play_count uint32
	play_turn  uint32
	left_cards []uint16
	drop_cards []uint16
	win_player *Player
}

func NewTable(tid uint32) *Table {
	t := new(Table)
	t.tid = tid
	return t
}

func (t *Table) getPlayerIndex(uid uint64) (int, error) {
	for index, player := range t.players {
		if player.Uid == uid {
			return index, nil
		}
	}
	return -1, errors.New("not in table")
}

func (t *Table) getPlayer(uid uint64) (*Player, error) {
	for _, player := range t.players {
		if player.Uid == uid {
			return player, nil
		}
	}
	return nil, errors.New("not in table")
}

func (t *Table) addAgent(agent gate.Agent, master bool) error {
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

func (t *Table) removeAgent(agent gate.Agent) error {
	uid := agent.UserData().(*userdata.UserData).Uid
	if index, err := t.getPlayerIndex(uid); err == nil {
		t.players = append(t.players[:index], t.players[index+1:]...)
		return nil
	}
	return errors.New("Agent not in table")
}

func (t *Table) Broadcast(msg interface{}) {
	for _, player := range t.players {
		player.WriteMsg(msg)
	}
}

func (t *Table) BroadcastExceptMe(msg interface{}, uid uint64) {
	for _, player := range t.players {
		if player.Uid != uid {
			player.WriteMsg(msg)
		}
	}
}

func (t *Table) shuffle() {
	each_cards := []uint16{101, 102, 103, 104, 105, 106, 107, 108, 109, 201, 202, 203, 204, 205, 206, 207, 208, 209,
		301, 302, 303, 304, 305, 306, 307, 308, 309, 401, 402, 403, 404, 405, 406, 407}
	var all_cards []uint16
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

func (t *Table) deal() {
	for _, player := range t.players {
		player.Cards = t.left_cards[:13]
		t.left_cards = append(t.left_cards[:0], t.left_cards[13:]...)

		log.Debug("Uid:%v Cards:%v", player.Uid, player.Cards)
	}
}

func (t *Table) draw_card() uint16 {
	card := t.left_cards[0]
	t.left_cards = append(t.left_cards[:0], t.left_cards[1:]...)
	return card
}

func (t *Table) play() {
	t.play_count++
	t.shuffle()
	t.deal()
	for len(t.left_cards) > 10 && t.win_player == nil && len(t.players) > 0 {
		player := t.players[t.play_turn]
		t.play_turn = (t.play_turn + 1) % uint32(len(t.players))
		player.draw()
		msg, err := player.WaitDrawRsp()
		if err != nil {
			player.SetOnline(false)
			log.Debug("err:%v", err)
		}
		log.Debug("msg:%v", msg)
		log.Debug("Table tid:%v runing, players num:%v, left_cards num:%d", t.tid, len(t.players), len(t.left_cards))
		time.Sleep(1 * time.Second)
	}
}

func (t *Table) run() {
	for {
		if len(t.players) == 0 {
			log.Debug("table:%v is over", t.tid)
			break
		} else if len(t.players) < 4 {
			log.Debug("waiting Agent join, table id:%v Agent num:%v", t.tid, len(t.players))
		} else {
			t.play()
		}
		time.Sleep(5 * time.Second)
	}
}
