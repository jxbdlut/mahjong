package proto

import (
	"fmt"
	"github.com/name5566/leaf/network/protobuf"
	"server/mahjong"
	"strings"
)

var Processor = protobuf.NewProcessor()

func init() {
	Processor.Register(&LoginReq{})
	Processor.Register(&LoginRsp{})
	Processor.Register(&CreateTableReq{})
	Processor.Register(&CreateTableRsp{})
	Processor.Register(&JoinTableReq{})
	Processor.Register(&JoinTableRsp{})
	Processor.Register(&UserJoinTableMsg{})
	Processor.Register(&OperatReq{})
	Processor.Register(&OperatRsp{})
	Processor.Register(&OperatMsg{})

	//Processor.Range(printRegistedMsg)
}

//func printRegistedMsg(id uint16, t reflect.Type) {
//	log.Debug("id:%v, type:%v", id, t)
//}

func (m *Eat) Equal(eat *Eat) bool {
	if len(m.HandCard) != len(eat.HandCard) || len(m.WaveCard) != len(eat.WaveCard) {
		return false
	}
	for i, card := range m.WaveCard {
		if card != eat.WaveCard[i] {
			return false
		}
	}
	for i, card := range m.HandCard {
		if card != eat.HandCard[i] {
			return false
		}
	}
	return true
}

func (m *Gang) Equal(gang *Gang) bool {
	if m.Type != gang.Type {
		return false
	}
	if m.Card != gang.Card {
		return false
	}
	return true
}

func NewOperatReq() *OperatReq {
	req := new(OperatReq)
	req.DealReq = new(DealReq)
	req.HuReq = new(HuReq)
	req.DrawReq = new(DrawReq)
	req.PongReq = new(PongReq)
	req.EatReq = new(EatReq)
	req.GangReq = new(GangReq)
	req.DropReq = new(DropReq)
	return req
}

func (m *OperatReq) Info() string {
	var result []string
	if m.Type&OperatType_DealOperat != 0 {
		result = append(result, "发牌:"+m.DealReq.Info())
	}
	if m.Type&OperatType_HuOperat != 0 {
		result = append(result, "胡:"+m.HuReq.Info())
	}
	if m.Type&OperatType_DrawOperat != 0 {
		result = append(result, "摸牌:"+m.DrawReq.Info())
	}
	if m.Type&OperatType_PongOperat != 0 {
		result = append(result, "碰:"+m.PongReq.Info())
	}
	if m.Type&OperatType_EatOperat != 0 {
		result = append(result, "吃:"+m.EatReq.Info())
	}
	if m.Type&OperatType_DropOperat != 0 {
		result = append(result, "出牌:"+m.DropReq.Info())
	}
	return strings.Join(result, ",")
}

func NewOperatRsp() *OperatRsp {
	Rsp := new(OperatRsp)
	Rsp.DealRsp = new(DealRsp)
	Rsp.HuRsp = new(HuRsp)
	Rsp.DrawRsp = new(DrawRsp)
	Rsp.PongRsp = new(PongRsp)
	Rsp.EatRsp = new(EatRsp)
	Rsp.GangRsp = new(GangRsp)
	Rsp.DropRsp = new(DropRsp)
	return Rsp
}

func (m *OperatRsp) Info() string {
	switch m.Type {
	case OperatType_DealOperat:
		return "发牌:" + m.DealRsp.Info()
	case OperatType_DrawOperat:
		return "摸牌:" + m.DrawRsp.Info()
	case OperatType_HuOperat:
		return "胡:" + m.HuRsp.Info()
	case OperatType_PongOperat:
		return "碰:" + m.PongRsp.Info()
	case OperatType_EatOperat:
		return "吃:" + m.EatRsp.Info()
	case OperatType_DropOperat:
		return "出牌:" + m.DropRsp.Info()
	default:
		return "rsp type err, type:" + fmt.Sprint(m.Type)
	}
}

func (m *DealReq) Info() string {
	return mahjong.CardsStr(m.Cards)
}

func (m *DealRsp) Info() string {
	return "[]"
}

func (m *DrawReq) Info() string {
	return "[" + mahjong.CardStr(m.Card) + "]"
}

func (m *DrawRsp) Info() string {
	return "[]"
}

func (m *HuReq) Info() string {
	return "[" + mahjong.CardStr(m.Card) + "]"
	return "[" + fmt.Sprintf("card:%v, type:%v, loser:%v", mahjong.CardStr(m.Card), m.Type, m.Lose) + "]"
}

func (m *HuRsp) Info() string {
	return "[" + fmt.Sprintf("%v, card:%v, type:%v, loser:%v", m.Ok, mahjong.CardStr(m.Card), m.Type, m.Lose) + "]"
}

func (m *Eat) Cards() string {
	return mahjong.CardsStr(m.HandCard) + "/" + mahjong.CardsStr(m.WaveCard)
}

func (m *EatReq) Info() string {
	var str_eats []string
	for _, eat := range m.Eat {
		str_eats = append(str_eats, eat.Cards())
	}
	ret := "[" + strings.Join(str_eats, ",") + "]"
	return ret
}

func (m *EatRsp) Info() string {
	ret := "[" + m.Eat.Cards() + "]"
	return ret
}

func (m *PongReq) Info() string {
	cards := []int32{}
	for i := int32(0); i < 2; i++ {
		cards = append(cards, m.Card)
	}
	return mahjong.CardsStr(cards)
}

func (m *PongRsp) Info() string {
	cards := []int32{}
	for i := int32(0); i < 2; i++ {
		cards = append(cards, m.Card)
	}
	return mahjong.CardsStr(cards)
}

func (m *DropReq) Info() string {
	return "[]"
}

func (m *DropRsp) Info() string {
	return "[" + mahjong.CardStr(m.DisCard) + "]"
}

func NewOperatMsg() *OperatMsg {
	msg := new(OperatMsg)
	msg.Deal = new(DealRsp)
	msg.Hu = new(HuRsp)
	msg.Draw = new(DrawRsp)
	msg.Pong = new(PongRsp)
	msg.Eat = new(EatRsp)
	msg.Drop = new(DropRsp)
	return msg
}

func (m *OperatMsg) Info() string {
	str := fmt.Sprintf("uid:%v, ", m.Uid)
	switch m.Type {
	case OperatType_DealOperat:
		return str + "发牌:" + m.Deal.Info()
	case OperatType_DrawOperat:
		return str + "摸牌:" + m.Draw.Info()
	case OperatType_HuOperat:
		return str + "胡:" + m.Hu.Info()
	case OperatType_PongOperat:
		return str + "碰:" + m.Pong.Info()
	case OperatType_EatOperat:
		return str + "吃:" + m.Eat.Info()
	case OperatType_DropOperat:
		return str + "出牌:" + m.Drop.Info()
	default:
		return "rsp type err, type:" + fmt.Sprint(m.Type)
	}
}
