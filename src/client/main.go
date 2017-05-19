package main

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	lconf "github.com/name5566/leaf/conf"
	"github.com/name5566/leaf/gate"
	"github.com/name5566/leaf/log"
	"github.com/name5566/leaf/network"
	"math"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"reflect"
	"server/conf"
	"server/proto"
	"strconv"
	"time"
	"sort"
)

type agent struct {
	uid             uint64
	name            string
	pos             int
	others          []*agent
	cards           []int
	fan_card        int
	hun_card        int
	conn            network.Conn
	Processor       network.Processor
	userData        interface{}
	master          bool
	rand            *rand.Rand
	separate_result [5][]int
}

var (
	tid_chan = make(chan uint32)
)

func (a *agent) Login() error {
	md5Ctx := md5.New()
	md5Ctx.Write([]byte(strconv.FormatUint(a.uid, 10)))
	cipherStr := md5Ctx.Sum(nil)
	crypt_passwd := hex.EncodeToString(cipherStr)
	a.WriteMsg(&proto.LoginReq{
		Uid:    a.uid,
		Name:   a.name,
		Passwd: crypt_passwd,
	})
	loginRsp, err := a.ReadMsg()
	if err != nil {
		log.Debug("read message: %v", err)
		return errors.New("read message err")
	}

	log.Debug("loginRsp:%v", loginRsp)
	if loginRsp.(*proto.LoginRsp).GetErrCode() == 0 {
		a.uid = a.uid
		return nil
	} else {
		return errors.New(loginRsp.(*proto.LoginRsp).GetErrMsg())
	}
}

func (a *agent) CreateTable() (uint32, error) {
	if a.uid != 0 {
		a.WriteMsg(&proto.CreateTableReq{
			Uid: a.uid,
		})
		createTableRsp, err := a.ReadMsg()
		if err != nil {
			log.Error("create table readmsg: ", err)
			return 0, err
		}
		log.Debug("createTableRsp:%v", createTableRsp)
		return createTableRsp.(*proto.CreateTableRsp).GetTableId(), nil
	}
	return 0, nil
}

func (a *agent) JoinTable(tid uint32) error {
	a.WriteMsg(&proto.JoinTableReq{
		Uid:     a.uid,
		TableId: tid,
	})
	joinTableRsp, err := a.ReadMsg()
	if err != nil {
		log.Error("join table readmsg: ", err)
		return err
	}
	log.Debug("join table:%v rsp:%v", tid, joinTableRsp)
	return nil
}

func HandlerBroadCastMsg(args []interface{}) {
	msg := args[0]
	log.Debug("broadcast:%v", msg)
}

func HandlerDealCardMsg(args []interface{}) {
	msg := args[0].(*proto.DealCardReq)
	a := args[1].(gate.Agent)

	a.(*agent).Deal(msg.Cards, int(msg.FanCard), int(msg.HunCard))
	a.WriteMsg(&proto.DealCardRsp{
		ErrCode: 0,
		ErrMsg:  "",
	})
	log.Debug("HandlerDealCardMsg:%v", msg)
}

func HandlerDrawCardMsg(args []interface{}) {
	msg := args[0]
	a := args[1].(gate.Agent)
	discard := a.(*agent).Draw(int(msg.(*proto.DrawCardReq).Card))
	a.WriteMsg(&proto.DrawCardRsp{
		Card: uint32(discard),
	})
	log.Debug("HandlerDrawCardMsg:%v", msg)
}

func HandlerEatMsg(args []interface{}) {
	msg := args[0]
	a := args[1].(gate.Agent)
	a.WriteMsg(&proto.EatRsp{
		Eat:     msg.(*proto.EatReq).GetEat()[0],
		DisCard: 0,
	})
	log.Debug("HandlerEatMsg:%v", msg)
}

func HandlerPongMsg(args []interface{}) {
	msg := args[0]
	a := args[1].(gate.Agent)
	a.WriteMsg(&proto.PongRsp{
		Count:   msg.(*proto.PongReq).Count,
		DisCard: 0,
	})
}

func (a *agent) Run() {
	if a.Login() != nil {
		return
	}
	if a.master {
		table_id, err := a.CreateTable()
		if err != nil {
			return
		}
		tid_chan <- table_id
		tid_chan <- table_id
		tid_chan <- table_id
	} else {
		table_id := <-tid_chan
		err := a.JoinTable(table_id)
		if err != nil {
			return
		}
	}
	for {
		msg, err := a.ReadMsg()
		if err != nil {
			log.Error("read message:", err)
			break
		}

		err = a.Processor.Route(msg, a)
		if err != nil {
			log.Debug("route message error: %v", err)
			break
		}
		log.Debug("msg type:%v, value:%v", reflect.TypeOf(msg), msg)
	}
}

func (a *agent) SeparateCards(cards []int) [5][]int {
	var result = [5][]int{}
	for _, card := range cards {
		m := int(0)
		if int(card) != a.hun_card {
			m = card / 100
		} else {
			m = 0
		}
		result[m] = append(result[m], int(card))
	}
	for _, cards := range result {
		sort.Ints(cards)
	}
	return result
}

func (a *agent) Deal(cards []int32, fan_card int, hun_card int) {
	for _, card := range cards {
		a.cards = append(a.cards, int(card))
	}
	a.fan_card = fan_card
	a.hun_card = hun_card
	a.separate_result = a.SeparateCards(a.cards)
}


func Count(cards []int,  card int) int {
	count := 0
	for _, c := range cards {
		if int(c) == card {
			count++
		}
	}
	return count
}


func Index(cards []int,  card int) int {
	for i, c := range cards {
		if int(c) == card {
			return i
		}
	}
	return -1
}

func (a *agent) DropSingle() int {
	wind_cards := a.separate_result[4]
	if len(wind_cards) == 1 {
		return wind_cards[0]
	} else {
		for _, card := range wind_cards {
			if Count(wind_cards, card) == 1 {
				return card
			}
		}
	}

	for i:= 1; i < 4; i++ {
		min_card, max_card := i * 100 + 1, i*100+9
		if Count(a.separate_result[i], min_card) > 0 && Count(a.separate_result[i], min_card+1) == 0 && Count(a.separate_result[i], min_card+2) == 0 {
			return min_card
		}
		if Count(a.separate_result[i], max_card) > 0 && Count(a.separate_result[i], max_card-1) == 0 && Count(a.separate_result[i], max_card-2) == 0 {
			return max_card
		}
		for _, card := range a.separate_result[i] {
			if Count(a.separate_result[i], card) > 1 {
				continue
			} else if Count(a.separate_result[i], card + 1) > 0 || Count(a.separate_result[i], card - 1) > 0{
				continue
			} else {
				return card
			}
		}
	}

	return 0
}

func (a *agent) Draw(card int) int {
	a.cards = append(a.cards, card)
	a.separate_result = a.SeparateCards(a.cards)
	discard := a.DropSingle()
	if discard == 0 {
		index := a.rand.Intn(len(a.cards))
		discard = a.cards[index]
	}
	index := Index(a.cards, discard)
	a.cards = append(a.cards[:index], a.cards[index+1:]...)
	log.Debug("draw card:%v discard:%v, index:%v", card, discard, index)
	return discard
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
		//if err = a.Processor.Route(msg, a); err != nil {
		//	log.Error("Route msg:%v err:%v", reflect.TypeOf(msg), err)
		//}
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
	for i := 0; i < 4; i++ {
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
			proto.Processor.SetHandler(&proto.UserJoinTableMsg{}, HandlerBroadCastMsg)
			proto.Processor.SetHandler(&proto.DealCardReq{}, HandlerDealCardMsg)
			proto.Processor.SetHandler(&proto.DrawCardReq{}, HandlerDrawCardMsg)
			proto.Processor.SetHandler(&proto.EatReq{}, HandlerEatMsg)
			proto.Processor.SetHandler(&proto.PongReq{}, HandlerPongMsg)
			r := rand.New(rand.NewSource(time.Now().UnixNano()))
			a := &agent{uid: uid, conn: conn, Processor: proto.Processor, master: is_master, rand: r}
			return a
		}

		client.Start()
	}

	// close
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)
	sig := <-c
	log.Release("Leaf closing down (signal: %v)", sig)
}
