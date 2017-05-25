package main

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	lconf "github.com/name5566/leaf/conf"
	"github.com/name5566/leaf/log"
	"github.com/name5566/leaf/network"
	"math"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"reflect"
	"server/conf"
	"server/mahjong"
	"server/proto"
	"strconv"
	"time"
)

type agent struct {
	uid             uint64
	name            string
	pos             int
	others          map[uint64]int
	cards           []int32
	fan_card        int32
	hun_card        int32
	conn            network.Conn
	Processor       network.Processor
	userData        interface{}
	master          bool
	rand            *rand.Rand
	separate_result [5][]int32
	rspChan         chan interface{}
}

const (
	PlayerNum = 1
	TableType = proto.CreateTableReq_TableNomal
)

var (
	tidChan = make(chan uint32, 3)
	c       = make(chan os.Signal, 1)
	others  = make(map[uint64]int)
)

func (a *agent) Login() (bool, error) {
	md5Ctx := md5.New()
	md5Ctx.Write([]byte(strconv.FormatUint(a.uid, 10)))
	cipherStr := md5Ctx.Sum(nil)
	crypt_passwd := hex.EncodeToString(cipherStr)
	a.WriteMsg(&proto.LoginReq{
		Uid:    a.uid,
		Name:   a.name,
		Passwd: crypt_passwd,
	})
	loginRsp := <-a.rspChan
	log.Debug("uid:%v loginRsp:%v", a.uid, loginRsp)
	if loginRsp.(*proto.LoginRsp).GetErrCode() == 0 {
		a.uid = a.uid
		return loginRsp.(*proto.LoginRsp).NeedRecover, nil
	} else {
		return false, errors.New(loginRsp.(*proto.LoginRsp).GetErrMsg())
	}
}

func HandlerLoginRsp(args []interface{}) {
	msg := args[0].(*proto.LoginRsp)
	a := args[1].(*agent)
	log.Debug("uid:%v LoginRsp:%v", a.uid, msg)
	a.rspChan <- msg
}

func (a *agent) CreateTable() (uint32, error) {
	if a.uid != 0 {
		a.WriteMsg(&proto.CreateTableReq{
			Type: int32(TableType),
		})
		msg := <-a.rspChan
		log.Debug("uid:%v createTableRsp:%v", a.uid, msg)
		return msg.(*proto.CreateTableRsp).GetTableId(), nil
	}
	return 0, nil
}

func HandlerCreateTableRsp(args []interface{}) {
	msg := args[0].(*proto.CreateTableRsp)
	a := args[1].(*agent)
	a.rspChan <- msg
}

func (a *agent) JoinTable(tid uint32) error {
	a.WriteMsg(&proto.JoinTableReq{
		TableId: tid,
	})
	msg := <-a.rspChan
	log.Debug("uid:%v join table:%v rsp:%v", a.uid, msg)
	return nil
}

func HandlerJoinTableRsp(args []interface{}) {
	msg := args[0].(*proto.JoinTableRsp)
	a := args[1].(*agent)
	log.Debug("uid:%v join table:%v rsp:%v", a.uid, msg)
	a.rspChan <- msg
}

func HandlerJoinTableMsg(args []interface{}) {
	msg := args[0].(*proto.UserJoinTableMsg)
	uid := msg.Uid
	others[uid] = int(msg.Pos)
	log.Debug("JoinTableMsg:%v", msg)
}

func HandlerOperatMsg(args []interface{}) {
	msg := args[0].(*proto.OperatMsg)
	a := args[1].(*agent)
	log.Release("uid:%v, pos:%v, %v", a.uid, others[msg.Uid], msg.Info())
}

func HandlerTableOperatReq(args []interface{}) {
	msg := args[0].(*proto.TableOperatReq)
	a := args[1].(*agent)
	a.WriteMsg(&proto.TableOperatRsp{Ok: true, Type: msg.Type})
}

func HandlerTableOperatMsg(args []interface{}) {
	msg := args[0].(*proto.TableOperatMsg)
	a := args[1].(*agent)
	log.Release("uid:%v, pos:%v, %v", a.uid, others[msg.Uid], msg)
}

func HandlerOperatReq(args []interface{}) {
	req := args[0].(*proto.OperatReq)
	a := args[1].(*agent)
	rsp := proto.NewOperatRsp()
	if req.Type&proto.OperatType_DealOperat != 0 {
		rsp.Type = proto.OperatType_DealOperat
		a.Deal(req.DealReq, rsp.DealRsp)
	} else if req.Type&proto.OperatType_HuOperat != 0 {
		rsp.Type = proto.OperatType_HuOperat
		a.Hu(req.HuReq, rsp.HuRsp)
	} else if req.Type&proto.OperatType_DrawOperat != 0 {
		rsp.Type = proto.OperatType_DrawOperat
		a.Draw(req.DrawReq, rsp.DrawRsp)
	} else if req.Type&proto.OperatType_GangOperat != 0 {
		rsp.Type = proto.OperatType_GangOperat
		a.Gang(req.GangReq, rsp.GangRsp)
	} else if req.Type&proto.OperatType_PongOperat != 0 {
		rsp.Type = proto.OperatType_PongOperat
		a.Pong(req.PongReq, rsp.PongRsp)
	} else if req.Type&proto.OperatType_EatOperat != 0 {
		rsp.Type = proto.OperatType_EatOperat
		a.Eat(req.EatReq, rsp.EatRsp)
	} else if req.Type&proto.OperatType_DropOperat != 0 {
		rsp.Type = proto.OperatType_DropOperat
		a.Drop(req.DropReq, rsp.DropRsp)
	}
	log.Release("uid:%v, %v, %v", a.uid, req.Info(), rsp.Info())
	log.Release("uid:%v, 手牌:%v", a.uid, mahjong.CardsStr(a.cards))
	a.WriteMsg(rsp)
}

func (a *agent) Start() {
	need_recover, err := a.Login()
	if err != nil {
		return
	}
	if need_recover {
		log.Debug("uid:%v, need_recover", a.uid)
		return
	}
	if a.master {
		table_id, err := a.CreateTable()
		if err != nil {
			return
		}
		tidChan <- table_id
		tidChan <- table_id
		tidChan <- table_id
	} else {
		table_id := <-tidChan
		err := a.JoinTable(table_id)
		if err != nil {
			return
		}
	}
}

func (a *agent) Run() {
	go a.Start()
	for {
		_, err := a.ReadMsg()
		if err != nil {
			log.Error("read message:%v", err)
			break
		}
	}
	if a.master {
		c <- os.Signal(os.Interrupt)
	}
	log.Release("uid:%v run exit", a.uid)
}

func (a *agent) Hu(req *proto.HuReq, rsp *proto.HuRsp) bool {
	rsp.Ok = true
	rsp.Card = req.Card
	rsp.Type = req.Type
	rsp.Lose = req.Lose
	return true
}

func (a *agent) Deal(req *proto.DealReq, rsp *proto.DealRsp) bool {
	cards := req.Cards
	a.cards = append(a.cards[:0])
	for _, card := range cards {
		a.cards = append(a.cards, card)
	}
	a.fan_card = req.FanCard
	a.hun_card = req.HunCard
	a.separate_result = mahjong.SeparateCards(a.cards, a.hun_card)
	return true
}

func (a *agent) DelGang(card int32) {
	for i := 0; i < 4; i++ {
		index := mahjong.Index(a.cards, card)
		if index != -1 {
			a.cards = append(a.cards[:index], a.cards[index+1:]...)
		}
	}
}

func (a *agent) Drop(req *proto.DropReq, rsp *proto.DropRsp) bool {
	a.separate_result = mahjong.SeparateCards(a.cards, a.hun_card)
	discard := mahjong.DropSingle(a.separate_result)
	if discard == 0 {
		discard = mahjong.DropRand(a.cards, a.hun_card)
	}
	a.cards = mahjong.DelCard(a.cards, discard, 0, 0)
	rsp.DisCard = discard
	return true
}

func (a *agent) Draw(req *proto.DrawReq, rsp *proto.DrawRsp) bool {
	a.cards = append(a.cards, req.Card)
	a.separate_result = mahjong.SeparateCards(a.cards, a.hun_card)
	mahjong.SortCards(a.cards, a.hun_card)
	return true
}

func (a *agent) Eat(req *proto.EatReq, rsp *proto.EatRsp) bool {
	eat := req.Eat[0]
	a.cards = mahjong.DelCard(a.cards, eat.HandCard[0], eat.HandCard[1], 0)
	rsp.Eat = eat
	return true
}

func (a *agent) Pong(req *proto.PongReq, rsp *proto.PongRsp) bool {
	card := req.Card
	a.cards = mahjong.DelCard(a.cards, card, card, 0)
	rsp.Card, rsp.Ok = card, true
	return true
}

func (a *agent) Gang(req *proto.GangReq, rsp *proto.GangRsp) bool {
	gang := req.Gang[0]
	card := gang.Card
	switch gang.Type {
	case proto.GangType_MingGang:
		a.cards = mahjong.DelCard(a.cards, card, card, card)
	case proto.GangType_BuGang:
		a.cards = mahjong.DelCard(a.cards, card, 0, 0)
	case proto.GangType_AnGang:
		a.cards = mahjong.DelCard(a.cards, card, card, card)
		a.cards = mahjong.DelCard(a.cards, card, 0, 0)
	}
	rsp.Ok = true
	rsp.Gang = req.Gang[0]
	return true
}

func (a *agent) OnClose() {
	a.conn.Close()
}

func (a *agent) WriteMsg(msg interface{}) {
	if a.Processor != nil {
		data, err := a.Processor.Marshal(msg)
		if err != nil {
			log.Error("marshal message %v error: %v", reflect.TypeOf(msg), err)
			return
		}
		err = a.conn.WriteMsg(data...)
		if err != nil {
			log.Error("write message %v error: %v", reflect.TypeOf(msg), err)
		}
	}
}

func (a *agent) ReadMsg() (interface{}, error) {
	data, err := a.conn.ReadMsg()
	if err != nil {
		log.Debug("read message: %v", err)
		return nil, err
	}
	if a.Processor != nil {
		msg, err := a.Processor.Unmarshal(data)
		if err != nil {
			log.Debug("Unmarshal data:%v", err)
			return nil, err
		}
		if err = a.Processor.Route(msg, a); err != nil {
			log.Error("Route msg:%v err:%v", reflect.TypeOf(msg), err)
		}
		return msg, nil
	}
	return nil, errors.New("processor is nil")
}

func (a *agent) LocalAddr() net.Addr {
	return a.conn.LocalAddr()
}

func (a *agent) RemoteAddr() net.Addr {
	return a.conn.RemoteAddr()
}

func (a *agent) Close() {
	a.conn.Close()
}

func (a *agent) Destroy() {
	a.conn.Destroy()
}

func (a *agent) UserData() interface{} {
	return a.userData
}

func (a *agent) SetUserData(data interface{}) {
	a.userData = data
}

func main() {
	lconf.LogLevel = conf.Server.LogLevel
	lconf.LogPath = conf.Server.LogPath
	//lconf.LogPath = "./log/"
	lconf.LogFlag = conf.LogFlag
	lconf.ConsolePort = conf.Server.ConsolePort
	lconf.ProfilePath = conf.Server.ProfilePath
	if lconf.LogLevel != "" {
		logger, err := log.New(lconf.LogLevel, lconf.LogPath, lconf.LogFlag)
		if err != nil {
			panic(err)
		}
		log.Export(logger)
		defer logger.Close()
	}

	table_start := 10000
	for i := 0; i < PlayerNum; i++ {
		uid := uint64(table_start)
		is_master := false
		if table_start == 10000 {
			is_master = true
		}
		table_start++
		client := new(network.TCPClient)
		client.Addr = "127.0.0.1:3563"
		client.ConnNum = 1
		client.ConnectInterval = 3 * time.Second
		client.PendingWriteNum = conf.PendingWriteNum
		client.LenMsgLen = 2
		client.MaxMsgLen = math.MaxUint32
		client.NewAgent = func(conn *network.TCPConn) network.Agent {
			proto.Processor.SetHandler(&proto.LoginRsp{}, HandlerLoginRsp)
			proto.Processor.SetHandler(&proto.CreateTableRsp{}, HandlerCreateTableRsp)
			proto.Processor.SetHandler(&proto.JoinTableRsp{}, HandlerJoinTableRsp)
			proto.Processor.SetHandler(&proto.UserJoinTableMsg{}, HandlerJoinTableMsg)
			proto.Processor.SetHandler(&proto.OperatReq{}, HandlerOperatReq)
			proto.Processor.SetHandler(&proto.OperatMsg{}, HandlerOperatMsg)
			proto.Processor.SetHandler(&proto.TableOperatReq{}, HandlerTableOperatReq)
			proto.Processor.SetHandler(&proto.TableOperatMsg{}, HandlerTableOperatMsg)
			r := rand.New(rand.NewSource(time.Now().UnixNano()))
			a := &agent{uid: uid, conn: conn, Processor: proto.Processor, master: is_master, rand: r}
			a.rspChan = make(chan interface{})
			return a
		}

		client.Start()
	}

	// close
	signal.Notify(c, os.Interrupt, os.Kill)
	sig := <-c
	log.Release("Leaf closing down (signal: %v)", sig)
}
