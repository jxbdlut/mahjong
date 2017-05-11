package internal

import (
	"net"

	"errors"
	"github.com/name5566/leaf/gate"
	"github.com/name5566/leaf/log"
	"server/proto"
	"time"
)

type Player struct {
	Agent       gate.Agent
	Uid         uint64
	Name        string
	Cards       []uint16
	PongCards   []uint16
	PrewinCards []uint16
	master      bool
	online      bool
	table       *Table
}

var (
	draw_rsp_chan = make(chan interface{})
)

func NewPlayer(agent gate.Agent, uid uint64) *Player {
	p := new(Player)
	p.Agent = agent
	p.Uid = uid

	return p
}

func (p *Player) draw() {
	card := p.table.draw_card()
	p.Cards = append(p.Cards, card)
	p.WriteMsg(&proto.DrawCardReq{
		Card: uint32(card),
	})
}

func (p *Player) HandlerDrawRsp(msg interface{}) {
	p.online = true
	log.Debug("HandlerDrawRsp msg:%v", msg)
	draw_rsp_chan <- msg
}

func (p *Player) WaitDrawRsp() (interface{}, error) {
	select {
	case <-draw_rsp_chan:
		return draw_rsp_chan, nil
	case <-time.After(10 * time.Second):
		return nil, errors.New("time out")
	}
}

func (p *Player) CheckPong() {

}

func (p *Player) SetMaster(master bool) {
	p.master = master
}

func (p *Player) SetTable(t *Table) {
	p.table = t
}

func (p *Player) SetOnline(online bool) {
	p.online = online
}

func (p *Player) GetOnline() bool {
	return p.online
}

func (p *Player) WriteMsg(msg interface{}) {
	p.Agent.WriteMsg(msg)
}

func (p *Player) LocalAddr() net.Addr {
	return p.Agent.LocalAddr()
}

func (p *Player) RemoteAddr() net.Addr {
	return p.Agent.RemoteAddr()
}

func (p *Player) Close() {
	p.Agent.Close()
}

func (p *Player) Destroy() {
	p.Agent.Destroy()
}

func (p *Player) UserData() interface{} {
	return p.Agent.UserData()
}

func (p *Player) SetUserData(data interface{}) {

}
