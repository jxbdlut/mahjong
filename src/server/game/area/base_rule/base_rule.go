package base_rule

type BaseRule struct {
	has_wind      bool
	has_hun       bool
	is_258		  bool
}

func NewBaseRule(has_wind bool, has_hun bool, is_258 bool) *BaseRule {
	rule := new(BaseRule)
	rule.has_wind = has_wind
	rule.has_hun = has_hun
	rule.is_258 = is_258
	return rule
}

func (m *BaseRule) IsJiang(card int32) bool {
	if m.is_258 {
		t := card / 100
		v := card % 10
		if t != 4 && (v == 2 || v == 5 || v == 8) {
			return true
		} else {
			return false
		}
	}
	return true
}

func (m *BaseRule) HasHun() bool {
	return m.has_hun
}

func (m *BaseRule) HasWind() bool {
	return m.has_wind
}

//func (m *BaseRule)HasDuiJiang(cards []int32) bool {
//	var cache []int32
//	for _, card := range cards {
//		if utils.Contain(cache, card) {
//			continue
//		}
//		cache = append(cache, card)
//		if utils.Count(cards, card) >= 2 && m.rule.IsJiang(card) {
//			return true
//		}
//	}
//	return false
//}

func (m *BaseRule)Check3Combine(card1 int32, card2 int32, card3 int32) bool {
	m1, m2, m3 := card1/100, card2/100, card3/100

	if m1 != m2 || m1 != m3 {
		return false
	}
	v1, v2, v3 := card1%10, card2%10, card3%10
	if v1 == v2 && v2 == v3 {
		return true
	}
	if m3 == 4 {
		return false
	}
	if v1+1 == v2 && v1+2 == v3 {
		return true
	}
	return false
}

func (m *BaseRule)Check2Combine(card1 int32, card2 int32) bool {
	if card1 == card2 {
		return true
	}
	return false
}

func (m *BaseRule) GetModNeedNum(len int, eye bool) int {
	var need_hun_arr []int
	if eye {
		need_hun_arr = []int{2, 1, 0}
	} else {
		need_hun_arr = []int{0, 2, 1}
	}

	if len == 0 {
		return 0
	} else {
		return need_hun_arr[len%3]
	}
}
