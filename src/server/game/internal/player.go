package internal

import (
	"errors"
	"fmt"
	"github.com/jxbdlut/leaf/gate"
	"github.com/jxbdlut/leaf/log"
	"net"
	"reflect"
	"server/mahjong"
	"server/proto"
	"sort"
	"strings"
	"time"
)

type Player struct {
	agent           gate.Agent
	uid             uint64
	name            string
	cards           []int32
	separate_result [5][]int32
	cancel_hu       bool
	waves           []*proto.Wave
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
	p.cancel_hu = false
	p.robot = NewRobot(p)
	p.timeout = 10 * time.Second
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
	str := fmt.Sprintf("uid:%v, %v/%v/[%v]", p.uid, mahjong.CardsStr(p.cards), proto.WavesStr(p.waves), mahjong.CardStr(p.table.hun_card))
	if p.win_card != 0 || len(p.prewin_cards) > 0 && mahjong.IsTingCardNum(len(p.cards)) {
		keys := []int32{}
		for key := range p.prewin_cards {
			if key != 0 {
				keys = append(keys, key)
			}
		}
		mahjong.SortCards(keys, 0)
		str = str + "听["
		tmp := []string{}
		for _, key := range keys {
			tmp = append(tmp, fmt.Sprintf("%v", p.prewin_cards[key].(*Ting).Info()))
		}
		str = str + strings.Join(tmp, ",")
		str = str + "]"
	}
	if p.win_card != 0 {
		str = str + "->" + mahjong.CardStr(p.win_card) + " 胡牌!"
	}
	return str
}

func (p *Player) GetInfo() *proto.Player {
	player := &proto.Player{}
	player.Cards = append(player.Cards, p.cards...)
	player.Uid = p.uid
	for i, p := range p.table.players {
		player.Pos = append(player.Pos, &proto.PosMsg{Uid: p.uid, Pos: int32(i)})
	}
	player.HunCard = p.table.hun_card
	player.CancelHu = p.cancel_hu
	player.DropCards = append(player.DropCards, p.table.drop_record[p.uid]...)
	player.Waves = append(player.Waves, p.waves...)
	player.PrewinCards = make(map[int32]*proto.PreWinCard)
	if len(p.prewin_cards) > 0 {
		prewinCard := &proto.PreWinCard{}
		for key := range p.prewin_cards {
			if key != 0 {
				prewinCard.Card = key
				player.PrewinCards[key] = &proto.PreWinCard{Card: key}
			}
		}
	}
	return player
}

func (p *Player) Clear() {
	p.cards = append(p.cards[:0])
	p.waves = append(p.waves[:0])
	p.prewin_cards = make(map[int32]interface{})
	p.win_card = 0
	p.cancel_hu = false
	p.separate_result = [5][]int32{}
}

func (p *Player) FeedCard(cards []int32) {
	p.Operat()
	p.cards = append(p.cards, cards...)
	mahjong.SortCards(p.cards, p.table.hun_card)
	p.separate_result = mahjong.SeparateCards(p.cards, p.table.hun_card)
}

func (p *Player) isNotPengPengHu(need_hun_arr []*Ting) bool {
	for _, wave := range p.waves {
		if wave.WaveType == proto.Wave_EatWave {
			return true
		}
	}

	shunzi_count := 0
	for _, ting := range need_hun_arr {
		if ting.shunzi_count > 0 {
			shunzi_count = shunzi_count + 1
		}
	}
	if shunzi_count > 1 {
		return true
	}
	return false
}

func (p *Player) isQingYiSe() bool {
	return true
}

func (p *Player) AddGangWave(cards []int32, t proto.GangType) {
	if t == proto.GangType_BuGang {
		for _, wave := range p.waves {
			if wave.Cards[0] == cards[0] {
				wave.Cards = append(wave.Cards, cards[0])
				wave.WaveType = proto.Wave_GangWave
				wave.GangType = proto.GangType_BuGang
				return
			}
		}
	} else {
		p.waves = append(p.waves, &proto.Wave{Cards: cards, WaveType: proto.Wave_GangWave, GangType: t})
	}
}

func (p *Player) AddPongWave(card int32) {
	p.waves = append(p.waves, &proto.Wave{Cards: []int32{card, card, card}, WaveType: proto.Wave_PongWave})
}

func (p *Player) AddEatWave(cards []int32) {
	p.waves = append(p.waves, &proto.Wave{Cards: cards, WaveType: proto.Wave_EatWave})
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
	req.DealReq.FanCard = p.table.fan_card
	req.DealReq.HunCard = p.table.hun_card
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

// card为零的情况是为了防止吃或者碰之后出错，要随机出一张牌
func (p *Player) Drop(disCard mahjong.DisCard) mahjong.DisCard {
	operatMsg := proto.NewOperatMsg()
	req := proto.NewOperatReq()
	req.Type = proto.OperatType_DropOperat
	p.CanGang(disCard, req)
	p.CanHu(disCard, req)
	if disCard.Card == 0 {
		disCard.FromUid = p.uid
		disCard.DisType = mahjong.DisCard_Normal
		disCard.Card = p.cards[len(p.cards)-1]
		operatMsg.Type = operatMsg.Type | proto.OperatType_DropOperat
		operatMsg.Drop.DisCard = disCard.Card
	}
	rsp, err := p.Notify(req)
	if err != nil {
		log.Error("uid:%v, Drop err:%v", p.uid, err)
		p.table.Broadcast(operatMsg)
		return disCard
	}
	log.Release("uid:%v, %v, %v", p.uid, req.Info(), rsp.(*proto.OperatRsp).Info())
	result, err := p.ValidRsp(req, rsp.(*proto.OperatRsp))
	if err != nil {
		log.Error("uid:%v ValidRsp err:%v", err)
		p.table.Broadcast(operatMsg)
		return disCard
	}
	p.BoardCastMsg(rsp.(*proto.OperatRsp))
	switch rsp.(*proto.OperatRsp).Type {
	case proto.OperatType_HuOperat:
		p.Hu(result.(*proto.HuRsp))
		disCard.Card = 0
	case proto.OperatType_GangOperat:
		p.Gang(result.(*proto.GangRsp).Gang.Cards, result.(*proto.GangRsp).Gang.Type)
		if result.(*proto.GangRsp).Gang.Type == proto.GangType_BuGang {
			disCard.Card = result.(*proto.GangRsp).Gang.Cards[0]
			disCard.DisType = mahjong.DisCard_BuGang
			disCard.FromUid = p.uid
			if p.table.CheckHu(disCard) {
				disCard.Card = 0
				return disCard
			}
		}
		return p.Draw(mahjong.DisCard_SelfGang)
	case proto.OperatType_DropOperat:
		disCard.Card = result.(*proto.DropRsp).DisCard
		p.DelCard(disCard.Card)
	}
	return disCard
}

func (p *Player) Draw(cardType mahjong.DisCardType) mahjong.DisCard {
	card := p.table.DrawCard()
	p.FeedCard([]int32{card})
	disCard := mahjong.DisCard{Card: card, FromUid: p.uid, DisType: cardType}
	req := proto.NewOperatReq()
	req.Type = proto.OperatType_DrawOperat
	req.DrawReq.Card = card

	rsp, err := p.Notify(req)
	if err != nil {
		log.Error("uid:%v Draw err:%v", p.uid, err)
		return disCard
	}
	log.Release("uid:%v, %v, %v", p.uid, req.Info(), rsp.(*proto.OperatRsp).Info())
	log.Release("%v", p)
	result, err := p.ValidRsp(req, rsp.(*proto.OperatRsp))
	_ = result
	if err != nil {
		return disCard
	}
	switch rsp.(*proto.OperatRsp).Type {
	case proto.OperatType_DrawOperat:
		return p.Drop(disCard)
	}
	return disCard
}

func (p *Player) HandlerOperatRsp(msg interface{}) {
	//log.Debug("uid:%v HandlerOperatRsp:%v", p.uid, msg)
	rsp_chan <- msg
}

func (p *Player) HandlerTableOperatRsp(msg interface{}) {
	//log.Debug("uid:%v HandlerTableOperatRsp:%v", p.uid, msg)
	rsp_chan <- msg
}

func (p *Player) SendRcv(msg interface{}) (interface{}, error) {
	p.Send(msg)
	select {
	case msg := <-rsp_chan:
		p.SetOnline(true)
		return msg, nil
	case <-time.After(p.timeout * time.Second):
		p.SetOnline(false)
		return nil, errors.New("time out")
	}
}

func (p *Player) BoardCastMsg(msg interface{}) {
	if reflect.TypeOf(msg) == reflect.TypeOf(&proto.OperatRsp{}) {
		rsp := msg.(*proto.OperatRsp)
		operatMsg := proto.NewOperatMsg()
		operatMsg.Uid = p.uid
		operatMsg.Type = rsp.Type
		switch rsp.Type {
		case proto.OperatType_DealOperat:
			operatMsg.Deal = rsp.DealRsp
		case proto.OperatType_DrawOperat:
			operatMsg.Draw = rsp.DrawRsp
		case proto.OperatType_HuOperat:
			operatMsg.Hu = rsp.HuRsp
		case proto.OperatType_PongOperat:
			operatMsg.Pong = rsp.PongRsp
		case proto.OperatType_EatOperat:
			operatMsg.Eat = rsp.EatRsp
		case proto.OperatType_GangOperat:
			operatMsg.Gang = rsp.GangRsp
		case proto.OperatType_DropOperat:
			operatMsg.Drop = rsp.DropRsp
		}
		p.table.Broadcast(operatMsg)
	} else if reflect.TypeOf(msg) == reflect.TypeOf(&proto.TableOperatMsg{}) {
		p.table.BroadcastExceptMe(msg, p.uid)
	}
}

func (p *Player) Notify(req interface{}) (interface{}, error) {
	//time.Sleep(2 * time.Second)
	if p.isRobot {
		return p.robot.HandlerMsg(req)
	}
	if p.online {
		rsp, err := p.agent.SendRcv(req)
		if err != nil {
			log.Error("uid:%v, Notify err:%v", p.uid, err)
			return nil, err
		}
		return rsp, nil
	} else {
		p.Send(req)
		return p.robot.HandlerMsg(req)
	}
	return nil, errors.New("online error")
}

func (p *Player) Hu(huRsp *proto.HuRsp) {
	p.win_card = huRsp.Card
	p.table.win_player = p
	log.Release("%v", p)
	log.Release("uid:%v, %v", p.uid, huRsp.Info())
	return
}

func (p *Player) Operat() {
	p.cancel_hu = false
}

func (p *Player) Gang(gangCards []int32, gangType proto.GangType) {
	var cards []int32
	var delCards []int32
	card := gangCards[0]
	p.Operat()
	switch gangType {
	case proto.GangType_MingGang:
		cards = []int32{card, card, card, card}
		delCards = []int32{card, card, card}
	case proto.GangType_BuGang:
		cards = []int32{card}
		delCards = []int32{card}
	case proto.GangType_AnGang:
		cards = []int32{card, card, card, card}
		delCards = []int32{card, card, card, card}
	case proto.GangType_SpecialGang:
		cards = []int32{card}
		delCards = []int32{card}
	}
	p.DelCards(delCards)
	p.AddGangWave(cards, gangType)
}

func (p *Player) Pong(card int32) {
	cards := []int32{card, card}
	p.Operat()
	p.DelCards(cards)
	p.AddPongWave(card)
}

func (p *Player) Eat(eat *proto.Eat) {
	p.Operat()
	p.AddEatWave(eat.WaveCard)
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
	log.Error("uid:%v, rsp err:%v", p.uid, rsp.Info())
	return nil, errors.New("rsp error")
}

func (p *Player) CanHu(disCard mahjong.DisCard, req *proto.OperatReq) bool {
	return p.table.rule.CanHu(disCard, p.GetInfo(), req)
}

func (p *Player) CheckHu(disCard mahjong.DisCard) bool {
	req := proto.NewOperatReq()
	if p.CanHu(disCard, req) {
		rsp, err := p.Notify(req)
		if err != nil {
			log.Debug("uid:%v, Hu err:%v", p.uid, err)
			return false
		}
		log.Release("uid:%v, %v, %v", p.uid, req.Info(), rsp.(*proto.OperatRsp).Info())
		result, err := p.ValidRsp(req, rsp.(*proto.OperatRsp))
		if err != nil {
			p.cancel_hu = true
			return false
		}
		if result.(*proto.HuRsp).Ok {
			p.BoardCastMsg(rsp.(*proto.OperatRsp))
			p.Hu(result.(*proto.HuRsp))
			return true
		} else {
			p.cancel_hu = true
		}
	}
	return false
}

func (p *Player) ValidHu(req *proto.HuReq, rsp *proto.HuRsp) bool {
	if rsp.Ok {
		if req.Type != rsp.Type {
			return false
		}
		if req.Card != rsp.Card {
			return false
		}
		if req.Lose != rsp.Lose {
			return false
		}
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
	if rsp.Ok {
		for _, can_eat := range req.Eat {
			if can_eat.Equal(rsp.Eat) {
				return true
			}
		}
		return false
	}
	return true
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

func (p *Player) CanEat(disCard mahjong.DisCard, req *proto.OperatReq) bool {
	return p.table.rule.CanEat(disCard, p.GetInfo(), req)
}

func (p *Player) CheckEat(disCard mahjong.DisCard) (mahjong.DisCard, bool) {
	req := proto.NewOperatReq()
	if p.CanEat(disCard, req) {
		rsp, err := p.Notify(req)
		if err != nil {
			log.Debug("uid:%v, Eat err:%v", p.uid, err)
			return disCard, false
		}
		result, err := p.ValidRsp(req, rsp.(*proto.OperatRsp))
		if err != nil {
			return disCard, false
		}
		log.Release("uid:%v, %v, %v", p.uid, req.Info(), rsp.(*proto.OperatRsp).Info())
		if result.(*proto.EatRsp).Ok {
			p.Eat(result.(*proto.EatRsp).Eat)
			p.BoardCastMsg(rsp.(*proto.OperatRsp))
			log.Release("%v", p)
			return p.Drop(mahjong.DisCard{Card: 0}), true
		} else {
			return disCard, false
		}
	} else {
		return disCard, false
	}
}

func (p *Player) CanAnGang(req *proto.OperatReq) bool {
	return p.table.rule.CanAnGang(p.GetInfo(), req)
}

func (p *Player) CanBuGang(req *proto.OperatReq) bool {
	return p.table.rule.CanBuGang(p.GetInfo(), req)
}

func (p *Player) CanMingGang(disCard mahjong.DisCard, req *proto.OperatReq) bool {
	return p.table.rule.CanMingGang(disCard, p.GetInfo(), req)
}

func (p *Player) CanGang(disCard mahjong.DisCard, req *proto.OperatReq) bool {
	if disCard.FromUid == p.uid {
		p.CanAnGang(req)
		p.CanBuGang(req)
	} else {
		p.CanMingGang(disCard, req)
	}
	return false
}

func (p *Player) CanPong(disCard mahjong.DisCard, req *proto.OperatReq) bool {
	return p.table.rule.CanPong(disCard, p.GetInfo(), req)
}

func (p *Player) CanGangOrPong(disCard mahjong.DisCard, req *proto.OperatReq) bool {
	ret := p.CanGang(disCard, req)
	ret = p.CanPong(disCard, req)
	return ret
}

func (p *Player) CheckGangOrPong(disCard mahjong.DisCard) (mahjong.DisCard, bool) {
	req := proto.NewOperatReq()
	if p.CanGangOrPong(disCard, req) {
		p.CanEat(disCard, req)
		rsp, err := p.Notify(req)
		if err != nil {
			log.Debug("uid:%v, Pong err:%v", p.uid, err)
			return disCard, false
		}
		log.Debug("uid:%v, %v, %v", p.uid, req.Info(), rsp.(*proto.OperatRsp).Info())
		result, err := p.ValidRsp(req, rsp.(*proto.OperatRsp))
		if err != nil {
			log.Error("uid:%v, rsp err:%v", p.uid, err)
			return disCard, false
		}
		p.BoardCastMsg(rsp.(*proto.OperatRsp))

		switch rsp.(*proto.OperatRsp).Type {
		case proto.OperatType_PongOperat:
			if result.(*proto.PongRsp).Ok {
				p.Pong(result.(*proto.PongRsp).Card)
				return p.Drop(mahjong.DisCard{Card: 0}), true
			}
		case proto.OperatType_GangOperat:
			if result.(*proto.GangRsp).Ok {
				p.Gang(result.(*proto.GangRsp).Gang.Cards, result.(*proto.GangRsp).Gang.Type)
				return p.Draw(mahjong.DisCard_SelfGang), true
			}
		case proto.OperatType_EatOperat:
			if result.(*proto.EatRsp).Ok {
				p.Eat(result.(*proto.EatRsp).Eat)
				return p.Drop(mahjong.DisCard{Card: 0}), true
			}
		}
	}
	return disCard, false
}

func (p *Player) ValidTableOpetat(req *proto.TableOperatReq, rsp *proto.TableOperatRsp) bool {
	if req.Type != rsp.Type {
		return false
	}
	return true
}

func (p *Player) CheckTableOperat(t proto.TableOperat) bool {
	req := proto.TableOperatReq{Type: t}
	rsp, err := p.Notify(&req)
	if err != nil {
		log.Error("uid:%v CheckTableOperat sendrcv err:%v", p.uid, err)
		return false
	}
	if !p.ValidTableOpetat(&req, rsp.(*proto.TableOperatRsp)) {
		log.Error("uid:%v CheckTableOperat rsp err, rsp:%v", rsp)
		return false
	}
	return rsp.(*proto.TableOperatRsp).Ok
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

func (p *Player) Replay(msg interface{}, seq uint32) {
	p.agent.Replay(msg, seq)
}

func (p *Player) Send(msg interface{}) {
	p.agent.Send(msg)
}

//func (p *Player) WriteMsg(msg interface{}, seq uint32) {
//	p.agent.WriteMsg(msg, nil, seq)
//}

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
