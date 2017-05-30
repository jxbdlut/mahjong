package proto

import (
	"errors"
	"fmt"
	"github.com/jxbdlut/leaf/network/protobuf"
	"server/mahjong"
	"strings"
)

var (
	Processor   = protobuf.NewProcessor()
	GangTypeMap = map[GangType]string{GangType_MingGang: "明杠", GangType_BuGang: "补杠", GangType_AnGang: "暗杠"}
	HuTypeMap   = map[HuType]string{HuType_Nomal: "平胡", HuType_Mo: "自摸", HuType_GangHua: "杠上花", HuType_QiangGang: "抢杠", HuType_HaiDiLao: "海底捞"}
	WaveTypeMap = map[Wave_WaveType]string{Wave_EatWave: "吃", Wave_PongWave: "碰", Wave_GangWave: "杠"}
)

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
	Processor.Register(&TableOperatReq{})
	Processor.Register(&TableOperatRsp{})
	Processor.Register(&TableOperatMsg{})

	//Processor.Range(printRegistedMsg)
}

//func printRegistedMsg(id uint16, t reflect.Type) {
//	log.Debug("id:%v, type:%v", id, t)
//}

func GangTypeStr(gangType GangType) string {
	return GangTypeMap[gangType]
}

func HuTypeStr(huType HuType) string {
	return HuTypeMap[huType]
}

func WaveStr(wave Wave_WaveType) string {
	return WaveTypeMap[wave]
}

func WavesStr(waves []*Wave) string {
	var str_cards []string
	for _, wave := range waves {
		str_cards = append(str_cards, wave.Info())
	}
	return "[" + strings.Join(str_cards, ",") + "]"
}
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
	if len(m.Cards) != len(gang.Cards) {
		return false
	}
	for i, card := range gang.Cards {
		if m.Cards[i] != card {
			return false
		}
	}

	return true
}

func (m *Wave) Equal(wave *Wave) bool {
	if m.GangType != wave.GangType {
		return false
	}
	if len(m.Cards) != len(wave.Cards) {
		return false
	}
	for i, card := range m.Cards {
		if card != wave.Cards[i] {
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
	if m.Type&OperatType_GangOperat != 0 {
		result = append(result, "杠:"+m.GangReq.Info())
	}
	if m.Type&OperatType_EatOperat != 0 {
		result = append(result, "吃:"+m.EatReq.Info())
	}
	if m.Type&OperatType_DropOperat != 0 {
		result = append(result, "出牌:"+m.DropReq.Info())
	}
	if len(result) == 0 {
		return "req type err"
	}
	return "[" + strings.Join(result, ",") + "]"
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
	var result []string
	if m.Type&OperatType_DealOperat != 0 {
		result = append(result, "发牌:"+m.DealRsp.Info())
	}
	if m.Type&OperatType_HuOperat != 0 {
		result = append(result, "胡:"+m.HuRsp.Info())
	}
	if m.Type&OperatType_DrawOperat != 0 {
		result = append(result, "摸牌:"+m.DrawRsp.Info())
	}
	if m.Type&OperatType_PongOperat != 0 {
		result = append(result, "碰:"+m.PongRsp.Info())
	}
	if m.Type&OperatType_GangOperat != 0 {
		result = append(result, "杠:"+m.GangRsp.Info())
	}
	if m.Type&OperatType_EatOperat != 0 {
		result = append(result, "吃:"+m.EatRsp.Info())
	}
	if m.Type&OperatType_DropOperat != 0 {
		result = append(result, "出牌:"+m.DropRsp.Info())
	}
	if len(result) == 0 {
		return "rsp type err"
	}
	return "[" + strings.Join(result, ",") + "]"
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
	return "[" + fmt.Sprintf("card:%v, type:%v, loser:%v", mahjong.CardStr(m.Card), HuTypeStr(m.Type), m.Lose) + "]"
}

func (m *HuRsp) Info() string {
	return "[" + fmt.Sprintf("%v, card:%v, type:%v, loser:%v", m.Ok, mahjong.CardStr(m.Card), HuTypeStr(m.Type), m.Lose) + "]"
}

func (m *Eat) Info() string {
	return mahjong.CardsStr(m.HandCard) + "/" + mahjong.CardsStr(m.WaveCard)
}

func (m *EatReq) Info() string {
	var str_eats []string
	for _, eat := range m.Eat {
		str_eats = append(str_eats, eat.Info())
	}
	ret := "[" + strings.Join(str_eats, ",") + "]"
	return ret
}

func (m *EatRsp) Info() string {
	ret := "[" + m.Eat.Info() + "]"
	return ret
}

func (m *PongReq) Info() string {
	cards := []int32{m.Card, m.Card}
	return mahjong.CardsStr(cards)
}

func (m *PongRsp) Info() string {
	if m.Ok {
		cards := []int32{m.Card, m.Card}
		return mahjong.CardsStr(cards)
	}
	return fmt.Sprintf("[%v]", m.Ok)
}

func (m *Gang) Info() string {
	return fmt.Sprintf("[%v, %v]", mahjong.CardsStr(m.Cards), GangTypeStr(m.Type))
}

func (m *GangReq) Info() string {
	var strGangs []string
	for _, gang := range m.Gang {
		strGangs = append(strGangs, gang.Info())
	}
	return "[" + strings.Join(strGangs, ",") + "]"
}

func (m *GangRsp) Info() string {
	if m.Ok {
		return m.Gang.Info()
	}
	return fmt.Sprintf("[%v]", m.Ok)
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
	msg.Gang = new(GangRsp)
	msg.Drop = new(DropRsp)
	return msg
}

func (m *OperatMsg) Info() string {
	str := fmt.Sprintf("uid:%v, ", m.Uid)
	var result []string
	if m.Type&OperatType_DealOperat != 0 {
		result = append(result, "发牌:"+m.Deal.Info())
	}
	if m.Type&OperatType_HuOperat != 0 {
		result = append(result, "胡:"+m.Hu.Info())
	}
	if m.Type&OperatType_DrawOperat != 0 {
		result = append(result, "摸牌:"+m.Draw.Info())
	}
	if m.Type&OperatType_PongOperat != 0 {
		result = append(result, "碰:"+m.Pong.Info())
	}
	if m.Type&OperatType_GangOperat != 0 {
		result = append(result, "杠"+m.Gang.Info())
	}
	if m.Type&OperatType_EatOperat != 0 {
		result = append(result, "吃:"+m.Eat.Info())
	}
	if m.Type&OperatType_DropOperat != 0 {
		result = append(result, "出牌:"+m.Drop.Info())
	}
	if len(result) == 0 {
		return "rsp type err"
	}
	return str + strings.Join(result, ",")
}

func (m *Wave) Info() string {
	return fmt.Sprintf("%v", mahjong.CardsStr(m.Cards))
}

func (m *Player) GetPlayerIndex(uid uint64) (int, error) {
	for i, pos := range m.Pos {
		if pos.Uid == uid {
			return i, nil
		}
	}
	return 0, errors.New(fmt.Sprintf("uid:%v not in table", uid))
}
