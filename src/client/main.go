package main

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	lconf "github.com/jxbdlut/leaf/conf"
	"github.com/jxbdlut/leaf/log"
	"github.com/jxbdlut/leaf/network"
	"github.com/jxbdlut/leaf/util"
	"math"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"reflect"
	"server/conf"
	"server/utils"
	"server/proto"
	"strconv"
	"sync/atomic"
	"time"
)

type agent struct {
	uid             uint64
	name            string
	seq             uint32
	cbChan          *util.Map
	pos             int
	timeout         time.Duration
	others          *util.Map
	cards           []int32
	fan_card        int32
	hun_card        int32
	conn            network.Conn
	Processor       network.Processor
	userData        interface{}
	master          bool
	rand            *rand.Rand
	separate_result [5][]int32
}

const (
	PlayerNum = 4
	TableType = proto.CreateTableReq_TableNomal
)

var (
	tidChan = make(chan uint32, 3)
	c       = make(chan os.Signal, 1)
)

func (a *agent) Login() (bool, error) {
	md5Ctx := md5.New()
	md5Ctx.Write([]byte(strconv.FormatUint(a.uid, 10)))
	cipherStr := md5Ctx.Sum(nil)
	crypt_password := hex.EncodeToString(cipherStr)
	loginRsp, err := a.SendRcv(&proto.LoginReq{
		Uid:    a.uid,
		Name:   a.name,
		Passwd: crypt_password,
	})
	if err != nil {
		log.Error("uid:%v Login err:%v", a.uid, err)
		return false, err
	}

	log.Debug("uid:%v loginRsp:%v", a.uid, loginRsp)
	if loginRsp.(*proto.LoginRsp).GetErrCode() == 0 {
		a.uid = a.uid
		return loginRsp.(*proto.LoginRsp).NeedRecover, nil
	} else {
		return false, errors.New(loginRsp.(*proto.LoginRsp).GetErrMsg())
	}
}

func (a *agent) CreateTable() (uint32, error) {
	if a.uid != 0 {
		msg, err := a.SendRcv(&proto.CreateTableReq{
			Type: int32(TableType),
			Area: 0,
		})
		if err != nil {
			log.Error("uid:%v CreateTable err:%v", a.uid, err)
			return 0, err
		}
		log.Debug("uid:%v createTableRsp:%v", a.uid, msg)
		return msg.(*proto.CreateTableRsp).GetTableId(), nil
	}
	return 0, nil
}

func (a *agent) JoinTable(tid uint32) error {
	msg, err := a.SendRcv(&proto.JoinTableReq{
		TableId: tid,
	})
	if err != nil {
		log.Error("uid:%v JoinTable err:%v", a.uid, err)
		return err
	}
	a.others.Set(a.uid, int(msg.(*proto.JoinTableRsp).Pos))
	log.Debug("uid:%v join table:%v rsp:%v", a.uid, tid, msg)
	return nil
}

func HandlerJoinTableMsg(args []interface{}) {
	msg := args[0].(*proto.UserJoinTableMsg)
	a := args[1].(*agent)
	for _, seat := range msg.Seats {
		a.others.Set(seat.Uid, int(seat.Pos))
	}
	log.Debug("JoinTableMsg:%v", msg)
}

func HandlerOperatMsg(args []interface{}) {
	msg := args[0].(*proto.OperatMsg)
	a := args[1].(*agent)
	log.Release("uid:%v, pos:%v, %v", a.uid, a.others.Get(msg.Uid), msg.Info())
}

func HandlerTableOperatReq(args []interface{}) {
	msg := args[0].(*proto.TableOperatReq)
	a := args[1].(*agent)
	seq := args[2].(uint32)
	a.WriteMsg(&proto.TableOperatRsp{Ok: true, Type: msg.Type}, nil, seq)
	log.Release("uid:%v, seq:%v, req:%v", a.uid, seq, msg)
}

func HandlerTableOperatMsg(args []interface{}) {
	msg := args[0].(*proto.TableOperatMsg)
	a := args[1].(*agent)
	log.Release("uid:%v, pos:%v, %v", a.uid, a.others.Get(msg.Uid), msg)
}

func HandlerOperatReq(args []interface{}) {
	req := args[0].(*proto.OperatReq)
	a := args[1].(*agent)
	seq := args[2].(uint32)
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
	log.Release("uid:%v, 手牌:%v", a.uid, utils.CardsStr(a.cards))
	a.WriteMsg(rsp, nil, seq)
}

func (a *agent) Start() {
	need_recover, err := a.Login()
	if err != nil {
		return
	}
	if need_recover {
		log.Debug("uid:%v, need_recover", a.uid)
		// todo
		//return
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
	rsp.Card = req.Card
	rsp.Type = req.Type
	rsp.Lose = req.Lose
	if req.Type != proto.HuType_Nomal {
		rsp.Ok = true
	} else {
		rsp.Ok = false
	}
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
	a.separate_result = utils.SeparateCards(a.cards, a.hun_card)
	return true
}

func (a *agent) DelGang(card int32) {
	for i := 0; i < 4; i++ {
		index := utils.Index(a.cards, card)
		if index != -1 {
			a.cards = append(a.cards[:index], a.cards[index+1:]...)
		}
	}
}

func (a *agent) Drop(req *proto.DropReq, rsp *proto.DropRsp) bool {
	a.separate_result = utils.SeparateCards(a.cards, a.hun_card)
	discard := utils.DropSingle(a.separate_result)
	if discard == 0 {
		discard = utils.DropRand(a.cards, a.hun_card)
	}
	a.cards = utils.DelCard(a.cards, discard, 0, 0)
	rsp.DisCard = discard
	return true
}

func (a *agent) Draw(req *proto.DrawReq, rsp *proto.DrawRsp) bool {
	a.cards = append(a.cards, req.Card)
	a.separate_result = utils.SeparateCards(a.cards, a.hun_card)
	utils.SortCards(a.cards, a.hun_card)
	return true
}

func (a *agent) Eat(req *proto.EatReq, rsp *proto.EatRsp) bool {
	eat := req.Eat[0]
	a.cards = utils.DelCard(a.cards, eat.HandCard[0], eat.HandCard[1], 0)
	rsp.Eat = eat
	rsp.Ok = true
	return true
}

func (a *agent) Pong(req *proto.PongReq, rsp *proto.PongRsp) bool {
	card := req.Card
	a.cards = utils.DelCard(a.cards, card, card, 0)
	rsp.Card, rsp.Ok = card, true
	return true
}

func (a *agent) Gang(req *proto.GangReq, rsp *proto.GangRsp) bool {
	gang := req.Gang[0]
	card := gang.Cards[0]
	switch gang.Type {
	case proto.GangType_MingGang:
		a.cards = utils.DelCard(a.cards, card, card, card)
	case proto.GangType_BuGang:
		a.cards = utils.DelCard(a.cards, card, 0, 0)
	case proto.GangType_AnGang:
		a.cards = utils.DelCard(a.cards, card, card, card)
		a.cards = utils.DelCard(a.cards, card, 0, 0)
	case proto.GangType_SpecialGang:
		a.cards = utils.DelCard(a.cards, card, 0, 0)
	}
	rsp.Ok = true
	rsp.Gang = req.Gang[0]
	return true
}

func (a *agent) DelTimeOut(seq uint32) bool {
	if a.cbChan.Get(seq) != nil {
		a.cbChan.Del(seq)
		return true
	}
	return false
}

func (a *agent) OnClose() {
	a.conn.Close()
}

func (a *agent) SendRcv(msg interface{}) (interface{}, error) {
	cbChan := make(chan interface{})
	atomic.AddUint32(&a.seq, 1)
	a.WriteMsg(msg, cbChan, a.seq)
	atomic.AddUint32(&a.seq, 1)
	select {
	case msg := <-cbChan:
		return msg, nil
	case <-time.After(a.timeout):
		a.DelTimeOut(a.seq)
		log.Error("sendrcv err seq:%v", a.seq)
		return nil, errors.New("time out")
	}
}

func (a *agent) WriteMsg(msg interface{}, cbChan chan interface{}, seq uint32) {
	if a.Processor != nil {
		if cbChan != nil {
			a.cbChan.Set(seq, cbChan)
		}
		data, err := a.Processor.Marshal(msg, seq)
		if err != nil {
			log.Error("marshal message %v error: %v", reflect.TypeOf(msg), err)
			return
		}
		err = a.conn.WriteMsg(data...)
		if err != nil {
			log.Error("write message %v error: %v", reflect.TypeOf(msg), err)
			return
		}
		return
	}
	return
}

func (a *agent) ReadMsg() (interface{}, error) {
	data, err := a.conn.ReadMsg()
	if err != nil {
		log.Debug("read message: %v", err)
		return nil, err
	}
	if a.Processor != nil {
		msg, seq, err := a.Processor.Unmarshal(data)
		if err != nil {
			log.Debug("Unmarshal data:%v", err)
			return nil, err
		}
		// cbChan
		if cbChan := a.cbChan.Get(seq); cbChan != nil {
			cbChan.(chan interface{}) <- msg
			a.DelTimeOut(seq)
			return msg, nil
		}
		if err = a.Processor.Route(msg, seq, a); err != nil {
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
	lconf.LogPath = ""
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

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	uid_start := r.Intn(4200000000)
	for i := 0; i < PlayerNum; i++ {
		uid := uint64(uid_start)
		is_master := false
		if i == 0 {
			is_master = true
		}
		uid_start++
		client := new(network.TCPClient)
		client.Addr = "127.0.0.1:3563"
		client.ConnNum = 1
		client.ConnectInterval = 3 * time.Second
		client.PendingWriteNum = conf.PendingWriteNum
		client.LenMsgLen = 2
		client.MaxMsgLen = math.MaxUint32
		client.NewAgent = func(conn *network.TCPConn) network.Agent {
			proto.Processor.SetHandler(&proto.UserJoinTableMsg{}, HandlerJoinTableMsg)
			proto.Processor.SetHandler(&proto.OperatReq{}, HandlerOperatReq)
			proto.Processor.SetHandler(&proto.OperatMsg{}, HandlerOperatMsg)
			proto.Processor.SetHandler(&proto.TableOperatReq{}, HandlerTableOperatReq)
			proto.Processor.SetHandler(&proto.TableOperatMsg{}, HandlerTableOperatMsg)
			r := rand.New(rand.NewSource(time.Now().UnixNano()))
			a := &agent{uid: uid, conn: conn, Processor: proto.Processor, master: is_master, rand: r}
			a.cbChan = new(util.Map)
			a.others = new(util.Map)
			a.timeout = 2 * time.Second
			return a
		}

		client.Start()
	}

	// close
	signal.Notify(c, os.Interrupt, os.Kill)
	sig := <-c
	log.Release("Leaf closing down (signal: %v)", sig)
}
