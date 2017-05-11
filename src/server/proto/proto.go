package proto

import (
	"github.com/name5566/leaf/log"
	"github.com/name5566/leaf/network/protobuf"
	"reflect"
)

var Processor = protobuf.NewProcessor()

func init() {
	Processor.Register(&LoginReq{})
	Processor.Register(&LoginRsp{})
	Processor.Register(&CreateTableReq{})
	Processor.Register(&CreateTableRsp{})
	Processor.Register(&JoinTableReq{})
	Processor.Register(&JoinTableRsp{})

	Processor.Range(printRegistedMsg)
}

func printRegistedMsg(id uint16, t reflect.Type) {
	log.Debug("id:%v, type:%v", id, t)
}
