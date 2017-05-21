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
	others          []*agent
	cards           []int32
	fan_card        int32
	hun_card        int32
	conn            network.Conn
	Processor       network.Processor
	userData        interface{}
	master          bool
	rand            *rand.Rand
	separate_result [5][]int32
	turn            int
}

var (
	tid_chan = make(chan uint32)
	c        = make(chan os.Signal, 1)
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

func HandlerOperatMsg(args []interface{}) {
	req := args[0].(*proto.OperatReq)
	rsp := proto.NewOperatRsp()
	a := args[1].(*agent)
	if req.Type&proto.OperatType_DealOperat != 0 {
		a.turn++
		rsp.Type = proto.OperatType_DealOperat
		a.Deal(req.DealReq, rsp.DealRsp)
	} else if req.Type&proto.OperatType_HuOperat != 0 {
		rsp.Type = proto.OperatType_HuOperat
		a.Hu(req.HuReq, rsp.HuRsp)
	} else if req.Type&proto.OperatType_DrawOperat != 0 && req.Type&proto.OperatType_PongOperat != 0 {
		a.AnGang(req, rsp)
	} else if req.Type&proto.OperatType_DrawOperat != 0 {
		rsp.Type = proto.OperatType_DrawOperat
		a.Draw(req.DrawReq, rsp.DrawRsp)
	} else if req.Type&proto.OperatType_PongOperat != 0 {
		rsp.Type = proto.OperatType_PongOperat
		a.Pong(req.PongReq, rsp.PongRsp)
	} else if req.Type&proto.OperatType_EatOperat != 0 {
		rsp.Type = proto.OperatType_EatOperat
		a.Eat(req.EatReq, rsp.EatRsp)
	}
	log.Release("uid:%v, %v, %v", a.uid, req.Info(), rsp.Info())
	log.Release("uid:%v, 手牌:%v", a.uid, mahjong.CardsStr(a.cards))
	a.WriteMsg(rsp)
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
		if a.turn > 100 {
			break
		}
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
	}
	if a.master {
		c <- os.Signal(os.Interrupt)
	}
}

func (a *agent) Hu(req *proto.HuReq, rsp *proto.HuRsp) bool {
	rsp.Ok = true
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

func (a *agent) DropSingle() int32 {
	wind_cards := a.separate_result[4]
	if len(wind_cards) == 1 {
		return wind_cards[0]
	} else {
		for _, card := range wind_cards {
			if mahjong.Count(wind_cards, card) == 1 {
				return card
			}
		}
	}

	for i := 1; i < 4; i++ {
		min_card, max_card := int32(i*100+1), int32(i*100+9)
		if mahjong.Count(a.separate_result[i], min_card) == 1 && mahjong.Count(a.separate_result[i], min_card+1) == 0 && mahjong.Count(a.separate_result[i], min_card+2) == 0 {
			return min_card
		}
		if mahjong.Count(a.separate_result[i], max_card) == 1 && mahjong.Count(a.separate_result[i], max_card-1) == 0 && mahjong.Count(a.separate_result[i], max_card-2) == 0 {
			return max_card
		}
	}

	for i := 1; i < 4; i++ {
		for _, card := range a.separate_result[i] {
			if mahjong.Count(a.separate_result[i], card) > 1 {
				continue
			} else if mahjong.Count(a.separate_result[i], card+1) > 0 || mahjong.Count(a.separate_result[i], card-1) > 0 {
				continue
			} else {
				return card
			}
		}
	}

	return 0
}

func (a *agent) DropRand() int32 {
	for {
		index := a.rand.Intn(len(a.cards))
		if a.hun_card != a.cards[index] {
			return a.cards[index]
		}
	}
}

func (a *agent) DelGang(card int32) {
	for i := 0; i < 4; i++ {
		index := mahjong.Index(a.cards, card)
		if index != -1 {
			a.cards = append(a.cards[:index], a.cards[index+1:]...)
		}
	}
}

func (a *agent) DelCard(card1 int32, card2 int32, card3 int32) []int32 {
	index := mahjong.Index(a.cards, card1)
	if index != -1 {
		a.cards = append(a.cards[:index], a.cards[index+1:]...)
	}
	index = mahjong.Index(a.cards, card2)
	if index != -1 {
		a.cards = append(a.cards[:index], a.cards[index+1:]...)
	}
	index = mahjong.Index(a.cards, card3)
	if index != -1 {
		a.cards = append(a.cards[:index], a.cards[index+1:]...)
	}
	a.separate_result = mahjong.SeparateCards(a.cards, a.hun_card)
	return a.cards
}

func (a *agent) Draw(req *proto.DrawReq, rsp *proto.DrawRsp) {
	a.cards = append(a.cards, req.Card)
	a.separate_result = mahjong.SeparateCards(a.cards, a.hun_card)
	discard := a.DropSingle()
	if discard == 0 {
		discard = a.DropRand()
	}
	a.cards = a.DelCard(discard, 0, 0)
	rsp.Card = discard
}

func (a *agent) Eat(req *proto.EatReq, rsp *proto.EatRsp) bool {
	eat := req.Eat[0]
	a.cards = a.DelCard(eat.HandCard[0], eat.HandCard[1], 0)
	discard := a.DropSingle()
	if discard == 0 {
		discard = a.DropRand()
	}
	a.cards = a.DelCard(discard, 0, 0)
	rsp.Eat, rsp.DisCard = eat, discard
	return true
}

func (a *agent) AnGang(req *proto.OperatReq, rsp *proto.OperatRsp) bool {
	a.cards = append(a.cards, req.DrawReq.Card)
	a.DelGang(req.PongReq.Card)
	rsp.Type = proto.OperatType_PongOperat
	rsp.PongRsp.Card = req.PongReq.Card
	rsp.PongRsp.Count = 4
	rsp.PongRsp.DisCard = 0
	return true
}

func (a *agent) Pong(req *proto.PongReq, rsp *proto.PongRsp) bool {
	count, card := req.Count, req.Card
	if count == 2 {
		a.cards = a.DelCard(card, card, 0)
	} else if count == 3 {
		a.cards = a.DelCard(card, card, card)
	}
	if count == 3 {
		rsp.Count, rsp.DisCard, rsp.Card = count, 0, card
		return true
	}
	discard := a.DropSingle()
	if discard == 0 {
		discard = a.DropRand()
	}
	a.cards = a.DelCard(discard, 0, 0)
	rsp.Card, rsp.DisCard, rsp.Count = card, discard, count
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
			proto.Processor.SetHandler(&proto.OperatReq{}, HandlerOperatMsg)
			r := rand.New(rand.NewSource(time.Now().UnixNano()))
			a := &agent{uid: uid, conn: conn, Processor: proto.Processor, master: is_master, rand: r}
			a.turn = 0
			return a
		}

		client.Start()
	}

	// close
	signal.Notify(c, os.Interrupt, os.Kill)
	sig := <-c
	log.Release("Leaf closing down (signal: %v)", sig)
}
