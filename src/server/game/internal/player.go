package internal

import (
	"errors"
	"github.com/name5566/leaf/gate"
	"github.com/name5566/leaf/log"
	"net"
	"server/proto"
	"sort"
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
	eat             []*proto.Eat
}

var (
	deal_rsp_chan = make(chan interface{})
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
	p.prewin_cards = make(map[int]interface{})
	p.separate_result = [5][]int{}
}

func (p *Player) FeedCard(cards []int) {
	p.cards = append(p.cards, cards...)
	sort.Ints(p.cards)
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

func (p *Player) drop(card int) {
	for i, c := range p.cards {
		if c == card {
			p.cards = append(p.cards[:i], p.cards[i+1:]...)
			break
		}
	}
	p.separate_result = p.table.SeparateCards(p.cards)
	p.prewin_cards = p.table.GetTingCards(p)
	if len(p.prewin_cards) > 0 {
		log.Debug("uid:%v, prewin_cards:%v, call_time:%v", p.uid, p.prewin_cards, p.table.call_time)
	}
}

func (p *Player) Deal() {
	var req proto.DealCardReq
	req.Uid = p.uid
	for _, card := range p.cards {
		req.Cards = append(req.Cards, int32(card))
	}
	req.FanCard = int32(p.table.fan_card)
	req.HunCard = int32(p.table.hun_card)
	p.WriteMsg(&req)
	rsp, err := p.WaitDealRsp()
	if err != nil {
		p.SetOnline(false)
		log.Debug("WaitDrawRsp err:%v", err)
		return
	}
	if rsp.(*proto.DealCardRsp).ErrCode == 0 {
		return
	} else {
		log.Debug("WaitDrawRsp err, rsp:%v", rsp)
		return
	}
	return
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
		req := &proto.DrawCardReq{
			Card: uint32(card),
		}
		log.Debug("uid:%v, draw req:%v", p.uid, req)
		p.WriteMsg(req)
	}
	rsp, err := p.WaitDrawRsp()
	if err != nil {
		p.SetOnline(false)
		log.Debug("WaitDrawRsp err:%v", err)
		return 0
	}
	log.Debug("uid:%v, draw rsp:%v", p.uid, rsp)
	dis_card := int(rsp.(*proto.DrawCardRsp).Card)
	log.Debug("uid:%v, cards:%v, cards len:%v", p.uid, p.cards, len(p.cards))
	if !p.ValidedDisCard(int(dis_card)) {
		log.Error("uid:%v, invalid drop:%v", p.uid, dis_card)
		dis_card = card
	}
	p.drop(dis_card)
	return dis_card
}

func (p *Player) HandlerDealRsp(msg interface{}) {
	p.online = true
	deal_rsp_chan <- msg
}

func (p *Player) HandlerDrawRsp(msg interface{}) {
	p.online = true
	draw_rsp_chan <- msg
}

func (p *Player) HandlerHuRsp(msg interface{}) {
	p.online = true
	hu_rsp_chan <- msg
}

func (p *Player) HandlerEatRsp(msg interface{}) {
	p.online = true
	eat_rsp_chan <- msg
}

func (p *Player) HandlerPongRsp(msg interface{}) {
	p.online = true
	pong_rsp_chan <- msg
}

func (p *Player) WaitDealRsp() (interface{}, error) {
	select {
	case msg := <-deal_rsp_chan:
		return msg, nil
	case <-time.After(10 * time.Second):
		return nil, errors.New("time out")
	}
}

func (p *Player) WaitDrawRsp() (interface{}, error) {
	select {
	case msg := <-draw_rsp_chan:
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
	log.Debug("uid:%v hu card:%v", p.uid, card)
	return
}

func (p *Player) Pong(card int, count int, discard int) {
	var cards []int32
	for i := 0; i < count; i++ {
		cards = append(cards, int32(card))
	}

	p.DelCards(cards)
	cards = append(cards, int32(card))
	p.AddPongCards(cards)
	if count == 2 {
		p.drop(discard)
	}
}

func (p *Player) Eat(eat *proto.Eat, discard int) {
	p.AddPongCards(eat.WaveCard)
	p.DelCards(eat.HandCard)
	p.drop(discard)
}

func (p *Player) CheckHu(card int) bool {
	hu := false
	if card == p.table.hun_card && len(p.prewin_cards) > 0 {
		hu = true
	}
	if _, ok := p.prewin_cards[card]; ok {
		hu = true
	}
	if hu {
		log.Debug("uid:%v hu card:%v", p.uid, card)
		p.WriteMsg(&proto.HuReq{
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

func (p *Player) ValidedDisCard(card int) bool {
	if p.count(card) == 0 {
		return false
	}
	return true
}

func (p *Player) ValidedEat(eat *proto.Eat) bool {
	for _, can_eat := range p.eat {
		if can_eat.Equal(eat) {
			return true
		}
	}
	return false
}

func (p *Player) CheckEat(card int) (int, bool) {
	m := card / 100
	if m == 4 || card == p.table.hun_card {
		return 0, false
	}
	c_1 := p.count(card - 1)
	c_2 := p.count(card - 2)
	c1 := p.count(card + 1)
	c2 := p.count(card + 2)

	var req proto.EatReq
	if c_1 > 0 && c_2 > 0 && (p.table.hun_card < card-2 || p.table.hun_card > card) {
		var eat proto.Eat
		eat.HandCard = []int32{int32(card - 2), int32(card - 1)}
		eat.WaveCard = []int32{int32(card - 2), int32(card - 1), int32(card)}
		req.Eat = append(req.Eat, &eat)
	}
	if c_1 > 0 && c1 > 0 && (p.table.hun_card < card-1 || p.table.hun_card > card+1) {
		var eat proto.Eat
		eat.HandCard = []int32{int32(card - 1), int32(card + 1)}
		eat.WaveCard = []int32{int32(card - 1), int32(card), int32(card + 1)}
		req.Eat = append(req.Eat, &eat)
	}
	if c1 > 0 && c2 > 0 && (p.table.hun_card < card || p.table.hun_card > card+2) {
		var eat proto.Eat
		eat.HandCard = []int32{int32(card + 1), int32(card + 2)}
		eat.WaveCard = []int32{int32(card), int32(card + 1), int32(card + 2)}
		req.Eat = append(req.Eat, &eat)
	}
	if len(req.Eat) > 0 {
		log.Debug("uid:%v eat req:%v", p.uid, req)
		p.eat = req.Eat
		p.WriteMsg(&req)
		rsp, err := p.WaitEatRsp()
		if err != nil {
			log.Debug("uid:%v WaitEatRsp err:%v", p.uid, err)
			return 0, false
		}
		log.Debug("uid:%v eat rsp:%v", p.uid, rsp)
		eat := rsp.(*proto.EatRsp).Eat
		dis_card := rsp.(*proto.EatRsp).DisCard
		if !p.ValidedEat(eat) {
			log.Error("uid:%v, invalid eat:$v", p.uid, eat)
			return 0, false
		}
		if !p.ValidedDisCard(int(dis_card)) {
			log.Error("uid:%v, invalid drop:%v", p.uid, dis_card)
			return 0, false
		}
		p.Eat(eat, int(dis_card))
		return int(dis_card), true
	} else {
		return 0, false
	}
}

func (p *Player) CheckPong(card int) (int, int) {
	count := p.count(card)
	if count >= 2 {
		req := &proto.PongReq{
			Card:  int32(card),
			Count: int32(count),
		}
		log.Debug("uid:%v pong req:%v", p.uid, req)
		p.WriteMsg(req)
		rsp, err := p.WaitPongRsp()
		if err != nil {
			log.Debug("WaitPongRsp err:%v", err)
			return 0, 0
		}
		log.Debug("uid:%v pong rsp:%v", p.uid, rsp)
		rsp_count := rsp.(*proto.PongRsp).Count
		dis_card := rsp.(*proto.PongRsp).DisCard
		if rsp_count == 0 {
			return 0, 0
		} else if rsp_count == 2 {
			if !p.ValidedDisCard(int(dis_card)) {
				log.Error("uid:%v, invalid drop:%v", p.uid, dis_card)
				return 0, 0
			}
			p.Pong(card, 2, int(dis_card))
			return int(dis_card), 2
		} else if rsp_count == 3 && count == 3 {
			p.Pong(card, 3, 0)
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
