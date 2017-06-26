package default_rule

import (
	"fmt"
	"server/game/area"
	"server/utils"
)

type Ting struct {
	rule         area.Rule
	card         int32
	pengpeng_hu  bool
	pair_7       bool
	qingyise     bool
	jiangyise    bool
	windyise     bool
	dandian      bool
	kazhang      bool
	duidao       bool
	shunzi       bool
	quanqiuren   bool
	is_ying      bool
	dragon_num   int
	need_hun_num int
}

func NewTing(card int32, qingyise bool, jiangyise bool, windyise bool) *Ting {
	ting := new(Ting)
	ting.card = card
	ting.qingyise = qingyise
	ting.jiangyise = jiangyise
	ting.windyise = windyise
	return ting
}

func (m *Ting) String() string {
	return fmt.Sprintf("%v", m.need_hun_num)
}

func (m *Ting) Info() string {
	if m.pengpeng_hu {
		return "碰碰胡:" + fmt.Sprintf("%v", utils.CardStr(m.card))
	}
	if m.qingyise {
		return "清一色:" + fmt.Sprintf("%v", utils.CardStr(m.card))
	}
	if m.jiangyise {
		return "将一色:" + fmt.Sprintf("%v", utils.CardStr(m.card))
	}
	if m.windyise {
		return "风一色:" + fmt.Sprintf("%v", utils.CardStr(m.card))
	}
	if m.pair_7 {
		return "七对:" + fmt.Sprintf("%v", utils.CardStr(m.card))
	}
	return fmt.Sprintf("%v", utils.CardStr(m.card))
}

func (m *Ting) SetPengPengHu() area.Ting {
	m.pengpeng_hu = true
	return m
}

func (m *Ting) SetJiangYiSe() area.Ting {
	m.jiangyise = true
	return m
}

func (m *Ting) SetWindYiSe() area.Ting {
	m.windyise = true
	return m
}

func (m *Ting) SetPair7() area.Ting {
	m.pair_7 = true
	return m
}

func (m *Ting) Copy() *Ting {
	ting := new(Ting)
	ting.card = m.card
	ting.pengpeng_hu = m.pengpeng_hu
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
