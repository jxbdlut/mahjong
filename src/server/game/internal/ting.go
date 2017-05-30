package internal

import (
	"reflect"
	"fmt"
)

type Ting struct {
	card int32
	pengpeng_hu bool
	shunzi_count int
	kezi_count int
	pair_7 int
	qingyise bool
	dandian  bool
	kazhang bool
	duidao bool
	shunzi bool
	quanqiuren bool
	dragon_num int
	need_hun_num int
	need_hun_list []NeedHun
}

type NeedHun struct {
	num int
	card int32
	eye bool
}

func NewTing(num int) *Ting {
	t := new(Ting)
	t.need_hun_num = num
	return t
}

func MaxTing() *Ting {
	t := new(Ting)
	t.need_hun_num = 5
	return t
}

func Minting() *Ting {
	t := new(Ting)
	t.need_hun_num = 0
	return t
}

func (m *Ting) String() string {
	return fmt.Sprintf("%v", m.need_hun_num)
}

func (m *Ting) Copy() *Ting {
	ting := new(Ting)
	ting.need_hun_num = m.need_hun_num
	return ting
}

func GetMax(t1 interface{}, t2 interface{}) interface{} {
	num1, num2 := 0, 0
	if reflect.TypeOf(t1) == reflect.TypeOf(int(0)) {
		num1 = t1.(int)
	} else {
		num1 = t1.(*Ting).need_hun_num
	}
	if reflect.TypeOf(t2) == reflect.TypeOf(int(0)) {
		num2 = t2.(int)
	} else {
		num2 = t2.(*Ting).need_hun_num
	}
	if num1 > num2 {
		return t1
	} else {
		return t2
	}
}

func GetMin(t1 *Ting, t2 *Ting) *Ting {
	if t1.need_hun_num < t2.need_hun_num {
		return t1
	}
	return t2
}

func (m *Ting) AddHun(num int, card int32, eye bool) {
	m.need_hun_num = m.need_hun_num + num
	m.need_hun_list = append(m.need_hun_list, NeedHun{num:num, card:card, eye:eye})
}

func (m *Ting) AddNum(num int) *Ting {
	m.need_hun_num = m.need_hun_num + num
	return m
}

func (m *Ting) SubNum(num int) *Ting {
	m.need_hun_num = m.need_hun_num - num
	return m
}

func (m *Ting) AddTing(t *Ting) *Ting {
	m.need_hun_num = m.need_hun_num + t.need_hun_num
	return m
}

func (m *Ting) SubTing(t *Ting) *Ting {
	m.need_hun_num = m.need_hun_num - t.need_hun_num
	return m
}

func (m *Ting) Equal(t *Ting) bool {
	return m.need_hun_num == t.need_hun_num
}

func (m *Ting) EqualNum(t int) bool {
	return m.need_hun_num == t
}

func (m *Ting) Bigger(t *Ting) bool {
	return m.need_hun_num > t.need_hun_num
}

func (m *Ting) BiggerNum(t int) bool {
	return m.need_hun_num > t
}

func (m *Ting) BiggerOrEqual(t *Ting) bool {
	return m.need_hun_num >= t.need_hun_num
}

func (m *Ting) BiggerOrEqualNum(t int) bool {
	return m.need_hun_num >= t
}

func (m *Ting) Smaller(t *Ting) bool {
	return m.need_hun_num < t.need_hun_num
}

func (m *Ting) SmallerOrEqual(t *Ting) bool {
	return m.need_hun_num <= t.need_hun_num
}