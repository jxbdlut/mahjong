package utils

import (
	"math/rand"
	"sort"
	"strings"
	"time"
)

type DisCardType int32

const (
	DisCard_Normal   = 0
	DisCard_Mo       = 1
	DisCard_BuGang   = 2
	DisCard_HaiDi    = 3
	DisCard_SelfGang = 4
)

type DisCard struct {
	Card    int32
	DisType DisCardType
	FromUid uint64
}

var (
	CardsMap = map[int32]string{101: "一万", 102: "二万", 103: "三万", 104: "四万", 105: "五万", 106: "六万", 107: "七万", 108: "八万",
		109: "九万",
		201: "一饼", 202: "二饼", 203: "三饼", 204: "四饼", 205: "五饼", 206: "六饼", 207: "七饼", 208: "八饼",
		209: "九饼",
		301: "一条", 302: "二条", 303: "三条", 304: "四条", 305: "五条", 306: "六条", 307: "七条", 308: "八条",
		309: "九条",
		401: "東", 402: "南", 403: "西", 404: "北", 405: "中", 406: "發", 407: "白",
		1: "腾空", 2: "飘将", 3: "飘门", 4: "风一色"}
)

func CardsStr(cards []int32) string {
	var str_cards []string
	for _, card := range cards {
		str_cards = append(str_cards, CardsMap[card])
	}
	return "[" + strings.Join(str_cards, ",") + "]"
}

func CardStr(card int32) string {
	return CardsMap[card]
}

func Count(cards []int32, card int32) int {
	count := 0
	for _, c := range cards {
		if c == card {
			count++
		}
	}
	return count
}

func Contain(elems []int32, elem int32) bool {
	for _, e := range elems {
		if e == elem {
			return true
		}
	}
	return false
}

func Index(cards []int32, card int32) int32 {
	for i, c := range cards {
		if c == card {
			return int32(i)
		}
	}
	return -1
}

func Copy(cards []int32) []int32 {
	cards_copy := make([]int32, len(cards))
	copy(cards_copy, cards)
	return cards_copy
}

func SearchRange(cards []int32) (int32, int32) {
	m := cards[0] / 100
	begin := int32(100*m + 10)
	end := int32(100 * m)

	if len(cards) == 2 && cards[0] == cards[1] {
		return cards[0], cards[1]
	}

	for _, card := range cards {
		if card < begin {
			begin = card
		}
		if card > end {
			end = card
		}
	}
	if m == 4 {
		return begin, end
	}
	if begin-2 < 100*m+1 {
		begin = 100*m + 1
	} else {
		begin = begin - 2
	}

	if end+2 > 100*m+9 {
		end = 100*m + 9
	} else {
		end = end + 2
	}

	return begin, end
}

func DelCountCard(cards []int32, card int32, count int) []int32 {
	for i := 0; i < count; i++ {
		index := Index(cards, card)
		if index != -1 {
			cards = append(cards[:index], cards[index+1:]...)
		}
	}
	return cards
}

func DelCard(cards []int32, card1 int32, card2 int32, card3 int32) []int32 {
	index := Index(cards, card1)
	if index != -1 {
		cards = append(cards[:index], cards[index+1:]...)
	}
	index = Index(cards, card2)
	if index != -1 {
		cards = append(cards[:index], cards[index+1:]...)
	}
	index = Index(cards, card3)
	if index != -1 {
		cards = append(cards[:index], cards[index+1:]...)
	}
	return cards
}

func SeparateCards(cards []int32, hun_card int32) [5][]int32 {
	var result = [5][]int32{}
	for _, card := range cards {
		m := int(0)
		if card != hun_card {
			m = int(card) / 100
		} else {
			m = 0
		}
		result[m] = append(result[m], card)
	}
	for _, cards := range result {
		SortCards(cards, hun_card)
	}
	return result
}

func Min(a int32, b int32) int32 {
	if a < b {
		return a
	} else {
		return b
	}
}

func IsTingCardNum(num int) bool {
	for i := 0; i <= 4; i++ {
		if num == i*3+1 {
			return true
		}
	}
	return false
}

func SortCards(cards []int32, hun_card int32) {
	sort.Slice(cards, func(i, j int) bool {
		if cards[i] == hun_card {
			return true
		} else if cards[j] == hun_card {
			return false
		} else {
			return cards[i] < cards[j]
		}
	})
}

func DropSingle(separate_result [5][]int32) int32 {
	wind_cards := separate_result[4]
	if len(wind_cards) == 1 {
		return wind_cards[0]
	} else {
		for _, card := range wind_cards {
			if Count(wind_cards, card) == 1 {
				return card
			}
		}
	}

	for i := 1; i < 4; i++ {
		min_card, max_card := int32(i*100+1), int32(i*100+9)
		if Count(separate_result[i], min_card) == 1 && Count(separate_result[i], min_card+1) == 0 && Count(separate_result[i], min_card+2) == 0 {
			return min_card
		}
		if Count(separate_result[i], max_card) == 1 && Count(separate_result[i], max_card-1) == 0 && Count(separate_result[i], max_card-2) == 0 {
			return max_card
		}
	}

	for i := 1; i < 4; i++ {
		for _, card := range separate_result[i] {
			if Count(separate_result[i], card) > 1 {
				continue
			} else if Count(separate_result[i], card+1) > 0 || Count(separate_result[i], card-1) > 0 {
				continue
			} else {
				return card
			}
		}
	}

	return 0
}

func DropNot258(cards []int32, hun_card int32) int32 {
	for _, card := range cards {
		if card == hun_card {
			continue
		}
		t, v := card/100, card%10
		if t == 4 || v != 2 && v != 5 && v != 8 {
			return card
		}
	}
	return 0
}

func DropNotWind(cards []int32, hun_card int32) int32 {
	for _, card := range cards {
		if card == hun_card {
			continue
		}
		t := card / 100
		if t != 4 {
			return card
		}
	}
	return 0
}

func AllCardsIsHun(cards []int32, hun_card int32) bool {
	for _, card := range cards {
		if card != hun_card {
			return false
		}
	}
	return true
}

func DropRand(cards []int32, hun_card int32) int32 {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	all_hun := AllCardsIsHun(cards, hun_card)
	for {
		index := r.Intn(len(cards))
		if all_hun {
			return cards[index]
		} else if hun_card != cards[index] {
			return cards[index]
		}
	}
}
