package proto

import (
	"github.com/name5566/leaf/network/protobuf"
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
	Processor.Register(&DealCardReq{})
	Processor.Register(&DealCardRsp{})
	Processor.Register(&DrawCardReq{})
	Processor.Register(&DrawCardRsp{})
	Processor.Register(&HuReq{})
	Processor.Register(&HuRsp{})
	Processor.Register(&EatReq{})
	Processor.Register(&EatRsp{})
	Processor.Register(&PongReq{})
	Processor.Register(&PongRsp{})

	//Processor.Range(printRegistedMsg)
}

//func printRegistedMsg(id uint16, t reflect.Type) {
//	log.Debug("id:%v, type:%v", id, t)
//}

func (e *Eat) Equal(eat *Eat) bool {
	if len(e.HandCard) != len(eat.HandCard) || len(e.WaveCard) != len(eat.WaveCard) {
		return false
	}
	for i, card := range e.WaveCard {
		if card != eat.WaveCard[i] {
			return false
		}
	}
	for i, card := range e.HandCard {
		if card != eat.HandCard[i] {
			return false
		}
	}
	return true
}
