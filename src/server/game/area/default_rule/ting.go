package default_rule

import (
	"fmt"
	"server/utils"
	"server/game/area"
)

type Ting struct {
	rule          area.Rule
	card          int32
	shunzi_count  int
	kezi_count    int
	pengpeng_hu   bool
	pair_7        bool
	qingyise      bool
	jiangyise     bool
	dandian       bool
	kazhang       bool
	duidao        bool
	shunzi        bool
	quanqiuren    bool
	piaomen       bool
	dragon_num    int
	need_hun_num  int
}

func NewTing(card int32) area.Ting {
	ting := new(Ting)
	ting.card = card
	return ting
}

func (m *Ting) String() string {
	return fmt.Sprintf("%v", m.need_hun_num)
}

func (m *Ting) Info() string {
	return fmt.Sprintf("%v", utils.CardStr(m.card))
}

func (m *Ting) Copy() *Ting {
	ting := new(Ting)
	//ting.table = m.table
	ting.card = m.card
	ting.pengpeng_hu = m.pengpeng_hu
	ting.shunzi_count = m.shunzi_count
	ting.kezi_count = m.kezi_count
	ting.pair_7 = m.pair_7
	ting.qingyise = m.qingyise
	ting.dandian = m.dandian
	ting.kazhang = m.kazhang
	ting.duidao = m.duidao
	ting.shunzi = m.shunzi
	ting.quanqiuren = m.quanqiuren
	ting.dragon_num = m.dragon_num
	ting.need_hun_num = m.need_hun_num
	return ting
}


