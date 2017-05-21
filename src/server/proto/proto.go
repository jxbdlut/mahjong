package proto

import (
	"github.com/name5566/leaf/network/protobuf"
	"server/mahjong"
	"strings"
	"fmt"
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

func NewOperatReq() *OperatReq {
	req := new(OperatReq)
	req.DealReq = new(DealReq)
	req.HuReq = new(HuReq)
	req.DrawReq = new(DrawReq)
	req.PongReq = new(PongReq)
	req.EatReq = new(EatReq)
	return req
}

func (m *OperatReq) Info() string {
	var result []string
	if m.Type & OperatType_DealOperat != 0 {
		result = append(result, "发牌:" + m.DealReq.Info())
	}
	if m.Type & OperatType_HuOperat != 0 {
		result = append(result, "胡:" + m.HuReq.Info())
	}
	if m.Type & OperatType_DrawOperat != 0 {
		result = append(result, "摸牌:" + m.DrawReq.Info())
	}
	if m.Type & OperatType_PongOperat != 0 {
		result = append(result, "碰:" + m.PongReq.Info())
	}
	if m.Type & OperatType_EatOperat != 0 {
		result = append(result, "吃:" + m.EatReq.Info())
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
	return Rsp
}

func (m *OperatRsp) Info() string {
	switch m.Type {
	case OperatType_DealOperat:
		return "发牌:" + m.DealRsp.Info()
	case OperatType_DrawOperat:
		return "出牌:" + m.DrawRsp.Info()
	case OperatType_HuOperat:
		return "胡:" + m.HuRsp.Info()
	case OperatType_PongOperat:
		return "碰:" + m.PongRsp.Info()
	case OperatType_EatOperat:
		return "吃:" + m.EatRsp.Info()
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
	return "[" + mahjong.CardStr(m.Card) + "]"
}

func (m *HuReq) Info() string {
	return "[" + mahjong.CardStr(m.Card) + "]"
}

func (m *HuRsp) Info() string {
	return "[" + fmt.Sprint(m.Ok) + "]"
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
	ret := "[" + m.Eat.Cards() + "],出牌[" + mahjong.CardStr(m.DisCard) + "]"
	return ret
}

func (m *PongReq) Info() string {
	cards := []int32{}
	for i := int32(0); i < m.Count; i++ {
		cards = append(cards, m.Card)
	}
	return mahjong.CardsStr(cards)
}

func (m *PongRsp) Info() string {
	cards := []int32{}
	for i := int32(0); i < m.Count; i++ {
		cards = append(cards, m.Card)
	}
	return mahjong.CardsStr(cards) + fmt.Sprintf(",出牌:[%v]", mahjong.CardStr(m.DisCard))
}