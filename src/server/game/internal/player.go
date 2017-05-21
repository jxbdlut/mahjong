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
	p.robot = NewAgent(p)
	p.timeout = 10
	return p
}

func (p *Player) SetAgent(agent gate.Agent)  {
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
		mahjong.SortCards(keys, p.table.hun_card)
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

func (p *Player) AnalyzeGang(req *proto.OperatReq) bool {
	var result []int32
	for _, m := range p.separate_result {
		if len(m) < 4 {
			continue
		}
		for _, card := range m {
			count := mahjong.Count(p.cards, card)
			if count >= 4 {
				result = append(result, card)
			}
		}
	}
	if len(result) > 0 {
		req.Type = req.Type | proto.OperatType_PongOperat
		req.PongReq.Count = 4
		req.PongReq.Type = proto.PongReq_AnGang
		req.PongReq.Card = result[0]
		return true
	}
	return false
}

func (p *Player) drop(card int32) {
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
	rsp, err := p.SendRcv(req)
	if err != nil {
		log.Debug("uid:%v, Deal rsp err:%v", p.uid, err)
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

func (p *Player) Draw(gang_flag bool) int32 {
	card := p.table.DrawCard()
	p.FeedCard([]int32{card})

	req := proto.NewOperatReq()
	req.Type = proto.OperatType_DrawOperat
	req.DrawReq.Card = card

	p.AnalyzeGang(req)
	p.CanHu(card, req)
	rsp, err := p.Notify(req)
	if err != nil {
		log.Error("uid:%v Draw err:%v", p.uid, err)
		return card
	}
	log.Debug("uid:%v, %v, %v", p.uid, req.Info(), rsp.(*proto.OperatRsp).Info())
	result, err := p.ValidRsp(req, rsp.(*proto.OperatRsp))
	if err != nil {
		return card
	}
	switch rsp.(*proto.OperatRsp).Type {
	case proto.OperatType_HuOperat:
		p.Hu(card, true)
		return 0
	case proto.OperatType_PongOperat:
		p.Pong(result.(*proto.PongRsp).Card, 4, result.(*proto.PongRsp).DisCard)
		return p.Draw(true)
	case proto.OperatType_DrawOperat:
		dis_card := result.(int32)
		p.drop(dis_card)
		return dis_card
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

func (p *Player) Notify(req *proto.OperatReq) (interface{}, error) {
	if p.online {
		rsp, err := p.SendRcv(req)
		if err != nil {
			log.Error("uid:%v, Notify err:%v", p.uid, err)
			return nil, err
		}
		return rsp, nil
	} else {
		return p.robot.HandlerOperatMsg(req)
	}
	return nil, errors.New("online error")
}

func (p *Player) Hu(card int32, mo bool) {
	str := fmt.Sprintf("uid:%v, 胡:%v", p.uid, mahjong.CardStr(card))
	if mo {
		str = str + " 自摸"
	}
	p.win_card = card
	log.Release("%v", p)
	log.Release(str)
	return
}

func (p *Player) Pong(card int32, count int, discard int32) {
	var cards []int32
	for i := 0; i < count; i++ {
		cards = append(cards, card)
	}

	p.DelCards(cards)
	if count != 4 {
		cards = append(cards, int32(card))
	}
	p.AddPongCards(cards)
	if count == 2 {
		p.drop(discard)
	}
}

func (p *Player) Eat(eat *proto.Eat, discard int32) {
	p.AddPongCards(eat.WaveCard)
	p.DelCards(eat.HandCard)
	p.drop(discard)
}

func (p *Player) ValidRsp(req *proto.OperatReq, rsp *proto.OperatRsp) (interface{}, error) {
	if rsp.ErrCode != 0 {
		return nil, errors.New(rsp.ErrMsg)
	}
	switch rsp.Type {
	case proto.OperatType_DealOperat:
		return nil, nil
	case proto.OperatType_DrawOperat:
		if p.ValidDisCard(rsp.DrawRsp.Card) {
			return rsp.DrawRsp.Card, nil
		} else {
			log.Error("uid:%v, invalid discard:%v", p.uid, rsp.DrawRsp.Card)
			return req.DrawReq.Card, nil
		}
	case proto.OperatType_HuOperat:
		return rsp.HuRsp.Ok, nil
	case proto.OperatType_PongOperat:
		count := rsp.PongRsp.Count
		card := req.PongReq.Card
		if count == 0 || (count > 1 && count <= req.PongReq.Count && card == rsp.PongRsp.Card) {
			if count == 3 && rsp.PongRsp.DisCard == 0 {
				return rsp.PongRsp, nil
			}
			if count == 4 && req.PongReq.Type == proto.PongReq_AnGang && rsp.PongRsp.DisCard == 0 {
				return rsp.PongRsp, nil
			}
			if p.ValidDisCard(rsp.PongRsp.DisCard) {
				return rsp.PongRsp, nil
			} else {
				log.Error("uid:%v, invalid discard:%v", p.uid, rsp.PongRsp.Info())
				return 0, errors.New("invalid discard")
			}
		} else {
			log.Error("uid:%v, invalid pong:%v", p.uid, rsp.PongRsp.Info())
			return 0, errors.New("invalid pong")
		}
	case proto.OperatType_EatOperat:
		if p.ValidEat(req.EatReq, rsp.EatRsp) {
			return rsp.EatRsp, nil
		} else {
			log.Error("uid:%v, invalid eat:%v", rsp.EatRsp.Info())
			return 0, errors.New("invalid eat")
		}
	default:
	}
	log.Error("uid:%v rsp type:%v err", p.uid, rsp.Type)
	return nil, errors.New("rsp type error")
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

func (p *Player) CheckHu(card int32, mo bool) bool {
	req := proto.NewOperatReq()
	if p.CanHu(card, req) {
		if !mo {
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
		return result.(bool)
	}
	return false
}

func (p *Player) ValidDisCard(card int32) bool {
	if mahjong.Count(p.cards, card) == 0 {
		return false
	}
	return true
}

func (p *Player) ValidEat(req *proto.EatReq, rsp *proto.EatRsp) bool {
	for _, can_eat := range req.Eat {
		if can_eat.Equal(rsp.Eat) {
			return p.ValidDisCard(rsp.DisCard)
		}
	}
	return false
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
		log.Debug("uid:%v, %v, %v", p.uid, req.Info(), rsp.(*proto.OperatRsp).Info())
		dis_card := result.(*proto.EatRsp).DisCard
		p.Eat(result.(*proto.EatRsp).Eat, dis_card)
		return dis_card, true
	} else {
		return 0, false
	}
}

func (p *Player) CanPong(card int32, req *proto.OperatReq) bool {
	count := mahjong.Count(p.cards, card)
	if count > 1 {
		req.Type = req.Type | proto.OperatType_PongOperat
		req.PongReq.Card = card
		req.PongReq.Count = count
		return true
	}
	return false
}

func (p *Player) CheckPong(card int32) (int32, int) {
	req := proto.NewOperatReq()
	if p.CanPong(card, req) {
		p.AnalyzeGang(req)
		p.CanEat(card, req)
		rsp, err := p.Notify(req)
		if err != nil {
			log.Debug("uid:%v, Pong err:%v", p.uid, err)
			return 0, 0
		}
		log.Debug("uid:%v, %v, %v", p.uid, req.Info(), rsp.(*proto.OperatRsp).Info())
		result, err := p.ValidRsp(req, rsp.(*proto.OperatRsp))
		if err != nil {
			log.Error("uid:%v, rsp err:%v", p.uid, err)
			return 0, 0
		}
		rsp_count := result.(*proto.PongRsp).Count
		dis_card := result.(*proto.PongRsp).DisCard
		if rsp_count == 0 {
			return 0, 0
		} else if rsp_count == 2 {
			p.Pong(card, 2, dis_card)
			return dis_card, 2
		} else if rsp_count == 3 {
			p.Pong(card, 3, dis_card)
			return p.Draw(true), 3
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
	if online {
		p.timeout = 10
	} else {
		p.timeout = 1
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
