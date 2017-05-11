package internal

import (
	"errors"
	"github.com/name5566/leaf/gate"
	"github.com/name5566/leaf/log"
	"server/userdata"
	"time"
	"go/src/cmd/go/testdata/testinternal3"
)

type Table struct {
	tid    uint32
	agents []gate.Agent
}

func NewTable(tid uint32) *Table {
	t := new(Table)
	t.tid = tid
	return t
}

func (t *Table) getAgentIndex(uid uint64) (int, error) {
	for index, agent := range t.agents {
		if agent.UserData().(*userdata.UserData).Uid == uid {
			return index, nil
		}
	}
	return -1, errors.New("not in table")
}

func (t *Table) addAgent(agent gate.Agent) error {
	if len(t.agents) < 4 {
		t.agents = append(t.agents, agent)
		return nil
	} else {
		return errors.New("this table is full!")
	}
}

func (t *Table) removeAgent(agent gate.Agent) error {
	uid := agent.UserData().(*userdata.UserData).Uid
	if index, err := t.getAgentIndex(uid); err == nil {
		t.agents = append(t.agents[:index], t.agents[index+1:]...)
		return nil
	}
	return errors.New("agent not in table")
}

func (t *Table) broadcast(msg interface{}) {
	for _, agent := range t.agents {
		agent.WriteMsg(msg)
	}
}

func (t *Table) run() {
	for {
		if len(t.agents) == 0 {
			log.Debug("table:%v is over", t.tid)
			break
		} else if len(t.agents) < 4 {
			log.Debug("waiting agent join, table id:%v agent num:%v", t.tid, len(t.agents))
		} else {
			log.Debug("Table tid:%v runing", t.tid)
		}
		time.Sleep(5 * time.Second)
	}
}
