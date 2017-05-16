package internal

import (
	"net"

	"errors"
	"github.com/name5566/leaf/gate"
	"github.com/name5566/leaf/log"
	"server/proto"
	"time"
)

type Player struct {
	agent           gate.Agent
	uid             uint64
	Name            string
	cards           []int
	separate_result [5][]int
	pong_cards      []int
	prewin_cards    map[int]interface{}
	master          bool
	online          bool
	table           *Table
}

var (
	draw_rsp_chan = make(chan interface{})
	hu_rsp_chan   = make(chan interface{})
	eat_rsp_chan  = make(chan interface{})
	pong_rsp_chan = make(chan interface{})
)

func NewPlayer(agent gate.Agent, uid uint64) *Player {
	p := new(Player)
	p.agent = agent
	p.uid = uid

	return p
}

func (p *Player) Clear() {
	p.cards = append(p.cards[:0], p.cards[:0]...)
	p.pong_cards = append(p.pong_cards[:0], p.pong_cards[:0]...)
	p.separate_result = [5][]int{}
}

func (p *Player) FeedCard(cards []int) {
	p.cards = append(p.cards, cards...)
	p.separate_result = p.table.SeparateCards(p.cards)
}

func (p *Player) AddPongCards(cards []int32) {
	for _, card := range cards {
		p.pong_cards = append(p.pong_cards, int(card))
	}
}

func (p *Player) GetCardIndex(card int) (int, error) {
	for i, c := range p.cards {
		if c == card {
			return i, nil
		}
	}
	return -1, errors.New("not found")
}

func (p *Player) DelCards(cards []int32) {
	for _, card := range cards {
		if index, err := p.GetCardIndex(int(card)); err == nil {
			p.cards = append(p.cards[:index], p.cards[index+1:]...)
		}
	}
}

func (p *Player) count(card int) int {
	count := 0
	for _, c := range p.cards {
		if int(c) == card {
			count++
		}
	}
	return count
}

func (p *Player) analyze_gang() []int {
	var result []int
	for _, m := range p.separate_result {
		if len(m) < 4 {
			continue
		}
		for _, card := range m {
			count := p.count(card)
			if count >= 4 {
				result = append(result, card)
			}
		}
	}
	return result
}

func (p *Player) discard(card int) {
	for i, c := range p.cards {
		if c == card {
			p.cards = append(p.cards[:i], p.cards[i+1:]...)
			break
		}
	}
}

func (p *Player) draw() int {
	card := p.table.DrawCard()
	p.FeedCard([]int{card})
	result := p.analyze_gang()
	if len(result) > 0 {
		log.Debug("analyze_gang result:%v", result)
	}
	if p.CheckHu(card) {
		p.Hu(card)
		return 0
	} else {
		p.WriteMsg(&proto.DrawCardReq{
			Card: uint32(card),
		})
	}
	rsp, err := p.WaitDrawRsp()
	if err != nil {
		p.SetOnline(false)
		log.Debug("err:%v", err)
		return 0
	}

	return int(rsp.(*proto.DrawCardRsp).Card)
}

func (p *Player) HandlerDrawRsp(msg interface{}) {
	p.online = true
	log.Debug("HandlerDrawRsp msg:%v", msg)
	draw_rsp_chan <- msg
}

func (p *Player) HandlerHuRsp(msg interface{}) {
	p.online = true
	log.Debug("HandlerHuRsp msg:%v", msg)
	hu_rsp_chan <- msg
}

func (p *Player) HandlerEatRsp(msg interface{}) {
	p.online = true
	log.Debug("HandlerEatRsp msg:%v", msg)
	eat_rsp_chan <- msg
}

func (p *Player) HandlerPongRsp(msg interface{}) {
	p.online = true
	log.Debug("HandlerPongRsp msg:%v", msg)
	pong_rsp_chan <- msg
}

func (p *Player) WaitDrawRsp() (interface{}, error) {
	select {
	case msg := <-draw_rsp_chan:
		p.discard(int(msg.(*proto.DrawCardRsp).Card))
		return msg, nil
	case <-time.After(10 * time.Second):
		return nil, errors.New("time out")
	}
}

func (p *Player) WaitHuRsp() (interface{}, error) {
	select {
	case msg := <-hu_rsp_chan:
		return msg, nil
	case <-time.After(10 * time.Second):
		return nil, errors.New("time out")
	}
}

func (p *Player) WaitEatRsp() (interface{}, error) {
	select {
	case msg := <-eat_rsp_chan:
		return msg, nil
	case <-time.After(10 * time.Second):
		return nil, errors.New("time out")
	}
}

func (p *Player) WaitPongRsp() (interface{}, error) {
	select {
	case msg := <-pong_rsp_chan:
		return msg, nil
	case <-time.After(10 * time.Second):
		return nil, errors.New("time out")
	}
}

func (p *Player) Hu(card int) {

}

func (p *Player) Pong(card int, count int) {
	var cards []int32
	for i := 0; i < count-1; i++ {
		cards = append(cards, int32(card))
	}
	p.DelCards(cards)
	cards = append(cards, int32(card))
	p.AddPongCards(cards)
}

func (p *Player) Eat(eat *proto.Eat) {
	p.AddPongCards(eat.WaveCard)
	p.DelCards(eat.HandCard)
}

func (p *Player) CheckHu(card int) bool {
	hu := false
	if card == p.table.hun_card && len(p.prewin_cards) > 0 {
		hu = true
	}
	if _, ok := p.prewin_cards[card]; ok {
		p.Hu(card)
		hu = true
	}
	if hu {
		p.WriteMsg(&proto.DrawCardReq{
			Card: uint32(card),
		})
		rsp, err := p.WaitHuRsp()
		if err != nil {
			log.Debug("WaitHuRsp err:%v", err)
			return false
		}
		return rsp.(*proto.HuRsp).Ok
	} else {
		return hu
	}
}

func (p *Player) CheckEat(card int) (*proto.Eat, int, bool) {
	m := card / 100
	if m == 4 || card == p.table.hun_card {
		return nil, 0, false
	}
	c_1 := p.count(card - 1)
	c_2 := p.count(card - 2)
	c1 := p.count(card + 1)
	c2 := p.count(card + 2)

	var req proto.EatReq
	var eat proto.Eat

	if c_1 > 0 && c_2 > 0 && (p.table.hun_card < card-2 || p.table.hun_card > card) {
		eat.HandCard = []int32{int32(card - 2), int32(card - 1)}
		eat.WaveCard = []int32{int32(card - 2), int32(card - 1), int32(card)}
		req.Eat = append(req.Eat, &eat)
	}
	if c_1 > 0 && c1 > 0 && (p.table.hun_card < card-1 || p.table.hun_card > card+1) {
		eat.HandCard = []int32{int32(card - 1), int32(card + 1)}
		eat.WaveCard = []int32{int32(card - 1), int32(card), int32(card + 1)}
		req.Eat = append(req.Eat, &eat)
	}
	if c1 > 0 && c2 > 0 && (p.table.hun_card < card || p.table.hun_card > card+2) {
		eat.HandCard = []int32{int32(card + 1), int32(card + 2)}
		eat.WaveCard = []int32{int32(card), int32(card + 1), int32(card + 2)}
		req.Eat = append(req.Eat, &eat)
	}
	if len(req.Eat) > 0 {
		log.Debug("eat req:%v", req)
		p.WriteMsg(&req)
		rsp, err := p.WaitEatRsp()
		if err != nil {
			log.Debug("WaitEatRsp err:%v", err)
			return nil, 0, false
		}
		eat := rsp.(*proto.EatRsp).Eat
		dis_card := rsp.(*proto.EatRsp).DisCard
		return eat, int(dis_card), true
	} else {
		return nil, 0, false
	}
}

func (p *Player) CheckPong(card int) (int, int) {
	count := p.count(card)
	if count >= 2 {
		p.WriteMsg(&proto.PongReq{
			Card:  int32(card),
			Count: int32(count),
		})
		rsp, err := p.WaitPongRsp()
		if err != nil {
			log.Debug("WaitPongRsp err:%v", err)
			return 0, 0
		}
		rsp_count := rsp.(*proto.PongRsp).Count
		dis_card := rsp.(*proto.PongRsp).DisCard
		if rsp_count == 0 {
			return 0, 0
		} else if rsp_count == 2 {
			p.Pong(card, 2)
			return int(dis_card), 2
		} else if rsp_count == 3 && count == 3 {
			p.Pong(card, 3)
			return p.draw(), 3
		}
	}
	return 0, 0
}

func (p *Player) SetMaster(master bool) {
	p.master = master
}

func (p *Player) SetTable(t *Table) {
	p.table = t
}

func (p *Player) SetOnline(online bool) {
	p.online = online
}

func (p *Player) GetOnline() bool {
	return p.online
}

func (p *Player) WriteMsg(msg interface{}) {
	p.agent.WriteMsg(msg)
}

func (p *Player) LocalAddr() net.Addr {
	return p.agent.LocalAddr()
}

func (p *Player) RemoteAddr() net.Addr {
	return p.agent.RemoteAddr()
}

func (p *Player) Close() {
	p.agent.Close()
}

func (p *Player) Destroy() {
	p.agent.Destroy()
}

func (p *Player) UserData() interface{} {
	return p.agent.UserData()
}

func (p *Player) SetUserData(data interface{}) {

}
