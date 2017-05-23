package internal

import (
	"errors"
	"fmt"
	"github.com/name5566/leaf/gate"
	"github.com/name5566/leaf/log"
	"net"
	"server/mahjong"
	"server/proto"
	"sort"
	"time"
)

type Player struct {
	agent           gate.Agent
	uid             uint64
	Name            string
	cards           []int32
	separate_result [5][]int32
	pong_cards      []int32
	prewin_cards    map[int32]interface{}
	win_card        int32
	master          bool
	online          bool
	table           *Table
	robot           robot
	isRobot         bool
	timeout         time.Duration
}

var (
	rsp_chan = make(chan interface{})
)

func NewPlayer(agent gate.Agent, uid uint64) *Player {
	p := new(Player)
	p.agent = agent
	p.uid = uid
	p.win_card = 0
	p.robot = NewRobot(p)
	p.timeout = 10
	if MinRobotId <= uid && uid <= MaxRobotId {
		p.isRobot = true
	}
	return p
}

func (p *Player) SetAgent(agent gate.Agent) {
	p.agent = agent
	p.SetOnline(true)
}

func (p *Player) String() string {
	sort.Slice(p.cards, func(i, j int) bool {
		if p.cards[i] == p.table.hun_card {
			return true
		} else if p.cards[j] == p.table.hun_card {
			return false
		} else {
			return p.cards[i] < p.cards[j]
		}
	})
	str := fmt.Sprintf("uid:%v, %v/[%v]", p.uid, mahjong.CardsStr(p.cards), mahjong.CardStr(p.table.hun_card))
	if len(p.prewin_cards) > 0 {
		keys := []int32{}
		for key := range p.prewin_cards {
			if key != 0 {
				keys = append(keys, key)
			}
		}
		mahjong.SortCards(keys, 0)
		str = str + fmt.Sprintf("/听%v", mahjong.CardsStr(keys))
	}
	if p.win_card != 0 {
		str = str + "->" + mahjong.CardStr(p.win_card) + " 胡牌!"
	}
	return str
}

func (p *Player) Clear() {
	p.cards = append(p.cards[:0], p.cards[:0]...)
	p.pong_cards = append(p.pong_cards[:0], p.pong_cards[:0]...)
	p.prewin_cards = make(map[int32]interface{})
	p.win_card = 0
	p.separate_result = [5][]int32{}
}

func (p *Player) FeedCard(cards []int32) {
	p.cards = append(p.cards, cards...)
	mahjong.SortCards(p.cards, p.table.hun_card)
	p.separate_result = mahjong.SeparateCards(p.cards, p.table.hun_card)
}

func (p *Player) AddPongCards(cards []int32) {
	for _, card := range cards {
		p.pong_cards = append(p.pong_cards, card)
	}
}

func (p *Player) GetCardIndex(card int32) (int, error) {
	for i, c := range p.cards {
		if c == card {
			return i, nil
		}
	}
	return -1, errors.New("not found")
}

func (p *Player) DelNumCards(card int32, count int) {
	for i := 0; i < count; i++ {
		if index, err := p.GetCardIndex(card); err == nil {
			p.cards = append(p.cards[:index], p.cards[index+1:]...)
		}
	}
}

func (p *Player) DelCards(cards []int32) {
	for _, card := range cards {
		if index, err := p.GetCardIndex(card); err == nil {
			p.cards = append(p.cards[:index], p.cards[index+1:]...)
		}
	}
	p.separate_result = mahjong.SeparateCards(p.cards, p.table.hun_card)
}

func (p *Player) DelCard(card int32) {
	for i, c := range p.cards {
		if c == card {
			p.cards = append(p.cards[:i], p.cards[i+1:]...)
			break
		}
	}
	p.separate_result = mahjong.SeparateCards(p.cards, p.table.hun_card)
	p.prewin_cards = p.table.GetTingCards(p)
	log.Release("%v", p)
}

func (p *Player) Deal() {
	req := proto.NewOperatReq()
	req.Type = proto.OperatType_DealOperat
	req.DealReq.Uid = p.uid
	req.DealReq.Cards = p.cards
	req.DealReq.FanCard = int32(p.table.fan_card)
	req.DealReq.HunCard = int32(p.table.hun_card)
	rsp, err := p.Notify(req)
	if err != nil {
		log.Error("uid:%v, Deal err:%v", p.uid, err)
		return
	}
	result, err := p.ValidRsp(req, rsp.(*proto.OperatRsp))
	if err != nil {
		_ = result
		log.Error("deal rsp invalid err:%v, rsp:%v", err, rsp.(*proto.OperatRsp).Info())
		return
	}
	return
}

// card为零的情况是吃或者碰之后出错，要随机出一张牌
func (p *Player) Drop(card int32) int32 {
	discard := card
	if discard == 0 {
		discard = p.cards[len(p.cards)-1]
	}
	req := proto.NewOperatReq()
	req.Type = proto.OperatType_DropOperat
	p.CanAnGang(req)
	p.CanHu(card, req)
	rsp, err := p.Notify(req)
	if err != nil {
		log.Error("uid:%v Drop err:%v", p.uid, err)
		return card
	}
	log.Release("uid:%v, %v, %v", p.uid, req.Info(), rsp.(*proto.OperatRsp).Info())
	if err != nil {
		log.Error("uid:%v Draw err:%v", p.uid, err)
		return card
	}
	result, err := p.ValidRsp(req, rsp.(*proto.OperatRsp))
	if err != nil {
		return card
	}
	p.BoardCastMsg(rsp.(*proto.OperatRsp))
	switch rsp.(*proto.OperatRsp).Type {
	case proto.OperatType_HuOperat:
		p.Hu(result.(*proto.HuRsp))
		return 0
	case proto.OperatType_GangOperat:
		p.Gang(result.(*proto.GangRsp).Gang.Card, result.(*proto.GangRsp).Gang.Type)
		return p.Draw(true)
	case proto.OperatType_DropOperat:
		discard = result.(*proto.DropRsp).DisCard
		p.DelCard(discard)
		return discard
	}
	return discard
}

func (p *Player) Draw(gang_flag bool) int32 {
	card := p.table.DrawCard()
	p.FeedCard([]int32{card})

	req := proto.NewOperatReq()
	req.Type = proto.OperatType_DrawOperat
	req.DrawReq.Card = card

	rsp, err := p.Notify(req)
	if err != nil {
		log.Error("uid:%v Draw err:%v", p.uid, err)
		return card
	}
	log.Release("uid:%v, %v, %v", p.uid, req.Info(), rsp.(*proto.OperatRsp).Info())
	log.Release("%v", p)
	result, err := p.ValidRsp(req, rsp.(*proto.OperatRsp))
	_ = result
	if err != nil {
		return card
	}
	switch rsp.(*proto.OperatRsp).Type {
	case proto.OperatType_DrawOperat:
		return p.Drop(card)
	}
	return card
}

func (p *Player) HandlerOperatRsp(msg interface{}) {
	p.online = true
	rsp_chan <- msg
}

func (p *Player) SendRcv(msg interface{}) (interface{}, error) {
	p.WriteMsg(msg)
	select {
	case msg := <-rsp_chan:
		p.SetOnline(true)
		return msg, nil
	case <-time.After(p.timeout * time.Second):
		p.SetOnline(false)
		return nil, errors.New("time out")
	}
}

func (p *Player) BoardCastMsg(rsp *proto.OperatRsp) {
	msg := proto.NewOperatMsg()
	msg.Uid = p.uid
	msg.Type = rsp.Type
	switch rsp.Type {
	case proto.OperatType_DealOperat:
		msg.Deal = rsp.DealRsp
	case proto.OperatType_DrawOperat:
		msg.Draw = rsp.DrawRsp
	case proto.OperatType_HuOperat:
		msg.Hu = rsp.HuRsp
	case proto.OperatType_PongOperat:
		msg.Pong = rsp.PongRsp
	case proto.OperatType_EatOperat:
		msg.Eat = rsp.EatRsp
	case proto.OperatType_GangOperat:
		msg.Gang = rsp.GangRsp
	case proto.OperatType_DropOperat:
		msg.Drop = rsp.DropRsp
	}
	p.table.BroadcastExceptMe(msg, p.uid)
}

func (p *Player) Notify(req *proto.OperatReq) (interface{}, error) {
	if p.isRobot {
		return p.robot.HandlerOperatMsg(req)
	}
	if p.online {
		rsp, err := p.SendRcv(req)
		if err != nil {
			log.Error("uid:%v, Notify err:%v", p.uid, err)
			return nil, err
		}
		return rsp, nil
	} else {
		p.WriteMsg(req)
		return p.robot.HandlerOperatMsg(req)
	}
	return nil, errors.New("online error")
}

func (p *Player) Hu(huRsp *proto.HuRsp) {
	p.win_card = huRsp.Card
	log.Release("%v", p)
	log.Release("uid:%v, %v", p.uid, huRsp.Info())
	return
}

func (p *Player) Gang(card int32, gangType proto.GangType) {
	var cards []int32
	var delCards []int32
	switch gangType {
	case proto.GangType_MingGang:
		cards = []int32{card, card, card}
		delCards = []int32{card, card, card}
	case proto.GangType_BuGang:
		cards = []int32{card}
		delCards = []int32{card}
	case proto.GangType_AnGang:
		cards = []int32{card, card, card, card}
		delCards = []int32{card, card, card, card}
	}
	p.DelCards(delCards)
	p.AddPongCards(cards)
}

func (p *Player) Pong(card int32) {
	cards := []int32{card, card}
	p.DelCards(cards)
	p.AddPongCards(cards)
}

func (p *Player) Eat(eat *proto.Eat) {
	p.AddPongCards(eat.WaveCard)
	p.DelCards(eat.HandCard)
}

func (p *Player) ValidRsp(req *proto.OperatReq, rsp *proto.OperatRsp) (interface{}, error) {
	if rsp.ErrCode != 0 {
		return nil, errors.New(rsp.ErrMsg)
	}
	switch rsp.Type {
	case proto.OperatType_DealOperat:
		if req.Type&proto.OperatType_DealOperat != 0 {
			return nil, nil
		}
	case proto.OperatType_DrawOperat:
		if req.Type&proto.OperatType_DrawOperat != 0 {
			return nil, nil
		}
	case proto.OperatType_DropOperat:
		if req.Type&proto.OperatType_DropOperat != 0 {
			if p.ValidDrop(rsp.DropRsp.DisCard) {
				return rsp.DropRsp, nil
			}
		}
	case proto.OperatType_HuOperat:
		if req.Type&proto.OperatType_HuOperat != 0 {
			if p.ValidHu(req.HuReq, rsp.HuRsp) {
				return rsp.HuRsp, nil
			}
		}
	case proto.OperatType_PongOperat:
		if req.Type&proto.OperatType_PongOperat != 0 {
			if p.ValidPong(req.PongReq, rsp.PongRsp) {
				return rsp.PongRsp, nil
			}
		}
	case proto.OperatType_GangOperat:
		if req.Type&proto.OperatType_GangOperat != 0 {
			if p.ValidGang(req.GangReq, rsp.GangRsp) {
				return rsp.GangRsp, nil
			}
		}
	case proto.OperatType_EatOperat:
		if req.Type&proto.OperatType_EatOperat != 0 {
			if p.ValidEat(req.EatReq, rsp.EatRsp) {
				return rsp.EatRsp, nil
			}
		}
	default:
		log.Error("uid:%v rsp type:%v err", p.uid, rsp.Type)
	}
	log.Error("uid:%v rsp err:%v", p.uid, rsp.Info())
	return nil, errors.New("rsp error")
}

func (p *Player) CanHu(card int32, req *proto.OperatReq) bool {
	_, ok := p.prewin_cards[card]
	if (card == p.table.hun_card && len(p.prewin_cards) > 0) || ok {
		req.Type = req.Type | proto.OperatType_HuOperat
		req.HuReq.Card = card
		return true
	}
	return false
}

func (p *Player) CheckHu(card int32, t proto.HuType) bool {
	req := proto.NewOperatReq()
	if p.CanHu(card, req) {
		if t == proto.HuType_GangHua || t == proto.HuType_Mo {
			p.CanPong(card, req)
			p.CanEat(card, req)
		}
		rsp, err := p.Notify(req)
		if err != nil {
			log.Debug("uid:%v, Hu err:%v", p.uid, err)
			return false
		}
		log.Release("uid:%v, %v, %v", p.uid, req.Info(), rsp.(*proto.OperatRsp).Info())
		result, err := p.ValidRsp(req, rsp.(*proto.OperatRsp))
		if err != nil {
			return false
		}
		if result.(*proto.HuRsp).Ok {
			p.BoardCastMsg(rsp.(*proto.OperatRsp))
			p.Hu(result.(*proto.HuRsp))
			return true
		}
	}
	return false
}

func (p *Player) ValidHu(req *proto.HuReq, rsp *proto.HuRsp) bool {
	if req.Type != rsp.Type {
		return false
	}
	if req.Card != rsp.Card {
		return false
	}
	if req.Lose != rsp.Lose {
		return false
	}
	return true
}

func (p *Player) ValidDrop(card int32) bool {
	if mahjong.Count(p.cards, card) == 0 {
		return false
	}
	return true
}

func (p *Player) ValidEat(req *proto.EatReq, rsp *proto.EatRsp) bool {
	for _, can_eat := range req.Eat {
		if can_eat.Equal(rsp.Eat) {
			return true
		}
	}
	return false
}

func (p *Player) ValidPong(req *proto.PongReq, rsp *proto.PongRsp) bool {
	if rsp.Ok {
		if rsp.Card != req.Card {
			return false
		}
		return true
	}
	return true
}

func (p *Player) ValidGang(req *proto.GangReq, rsp *proto.GangRsp) bool {
	if rsp.Ok {
		for _, can_gang := range req.Gang {
			if can_gang.Equal(rsp.Gang) {
				return true
			}
		}
		return false
	}
	return true
}

func (p *Player) CanEat(card int32, req *proto.OperatReq) bool {
	var eats []*proto.Eat
	m := card / 100
	if m == 4 || card == p.table.hun_card {
		return false
	}
	if p.table.players[p.table.play_turn].uid != p.uid {
		return false
	}
	c_1 := mahjong.Count(p.cards, card-1)
	c_2 := mahjong.Count(p.cards, card-2)
	c1 := mahjong.Count(p.cards, card+1)
	c2 := mahjong.Count(p.cards, card+2)

	if c_1 > 0 && c_2 > 0 && (p.table.hun_card < card-2 || p.table.hun_card > card) {
		var eat proto.Eat
		eat.HandCard = []int32{card - 2, card - 1}
		eat.WaveCard = []int32{card - 2, card - 1, card}
		eats = append(eats, &eat)
	}
	if c_1 > 0 && c1 > 0 && (p.table.hun_card < card-1 || p.table.hun_card > card+1) {
		var eat proto.Eat
		eat.HandCard = []int32{card - 1, card + 1}
		eat.WaveCard = []int32{card - 1, card, card + 1}
		eats = append(eats, &eat)
	}
	if c1 > 0 && c2 > 0 && (p.table.hun_card < card || p.table.hun_card > card+2) {
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

func (p *Player) CheckEat(card int32) (int32, bool) {
	req := proto.NewOperatReq()
	if p.CanEat(card, req) {
		rsp, err := p.Notify(req)
		if err != nil {
			log.Debug("uid:%v, Eat err:%v", p.uid, err)
			return 0, false
		}
		result, err := p.ValidRsp(req, rsp.(*proto.OperatRsp))
		if err != nil {
			return 0, false
		}
		log.Release("uid:%v, %v, %v", p.uid, req.Info(), rsp.(*proto.OperatRsp).Info())
		p.Eat(result.(*proto.EatRsp).Eat)
		p.BoardCastMsg(rsp.(*proto.OperatRsp))
		log.Release("%v", p)
		return p.Drop(0), true
	} else {
		return 0, false
	}
}

func (p *Player) CanAnGang(req *proto.OperatReq) bool {
	ret := false
	for _, m := range p.separate_result {
		if len(m) < 4 {
			continue
		}
		record := []int32{}
		for _, card := range m {
			count := mahjong.Count(p.cards, card)
			if count < 4 {
				continue
			}
			if mahjong.Contain(record, card) {
				continue
			}
			req.Type = req.Type | proto.OperatType_GangOperat
			gang := &proto.Gang{
				Card: card,
				Type: proto.GangType_AnGang,
			}
			req.GangReq.Gang = append(req.GangReq.Gang, gang)
			record = append(record, card)
			ret = true
		}
	}
	return ret
}

func (p *Player) CanBuGang(req *proto.OperatReq) bool {
	return false
}

func (p *Player) CanMingGang(card int32, req *proto.OperatReq) bool {
	count := mahjong.Count(p.cards, card)
	if count > 2 {
		req.Type = req.Type | proto.OperatType_GangOperat
		gang := &proto.Gang{
			Card: card,
			Type: proto.GangType_MingGang,
		}
		req.GangReq.Gang = append(req.GangReq.Gang, gang)
		return true
	}
	return false
}

func (p *Player) CanGang(card int32, req *proto.OperatReq) bool {
	if card == 0 {
		p.CanAnGang(req)
		p.CanBuGang(req)
	} else {
		p.CanMingGang(card, req)
	}
	return false
}

func (p *Player) CanPong(card int32, req *proto.OperatReq) bool {
	count := mahjong.Count(p.cards, card)
	if count > 1 {
		req.Type = req.Type | proto.OperatType_PongOperat
		req.PongReq.Card = card
		return true
	}
	return false
}

func (p *Player) CanGangOrPong(card int32, req *proto.OperatReq) bool {
	ret := p.CanGang(card, req)
	ret = p.CanPong(card, req)
	return ret
}

func (p *Player) CheckGangOrPong(card int32) (int32, bool) {
	req := proto.NewOperatReq()
	if p.CanGangOrPong(card, req) {
		p.CanEat(card, req)
		rsp, err := p.Notify(req)
		if err != nil {
			log.Debug("uid:%v, Pong err:%v", p.uid, err)
			return 0, false
		}
		log.Debug("uid:%v, %v, %v", p.uid, req.Info(), rsp.(*proto.OperatRsp).Info())
		result, err := p.ValidRsp(req, rsp.(*proto.OperatRsp))
		if err != nil {
			log.Error("uid:%v, rsp err:%v", p.uid, err)
			return 0, false
		}
		p.BoardCastMsg(rsp.(*proto.OperatRsp))

		switch rsp.(*proto.OperatRsp).Type {
		case proto.OperatType_PongOperat:
			p.Pong(result.(*proto.PongRsp).Card)
			return p.Drop(0), true
		case proto.OperatType_GangOperat:
			p.Gang(result.(*proto.GangRsp).Gang.Card, result.(*proto.GangRsp).Gang.Type)
			return p.Draw(true), true
		}
	}
	return 0, false
}

func (p *Player) SetMaster(master bool) {
	p.master = master
}

func (p *Player) SetTable(t *Table) {
	p.table = t
}

func (p *Player) SetOnline(online bool) {
	p.online = online
	if online {
		p.timeout = 10
	} else {
		p.timeout = 0
	}
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
