package internal

import (
	"net"

	"github.com/name5566/leaf/gate"
)

type Player struct {
	Agent       gate.Agent
	Uid         uint64
	Name        string
	Cards       []uint16
	PongCards   []uint16
	PrewinCards []uint16
	master      bool
}

func NewPlayer(agent gate.Agent, uid uint64) *Player {
	p := new(Player)
	p.Agent = agent
	p.Uid = uid

	return p
}

func (p *Player) SetMaster(master bool) {
	p.master = master
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
