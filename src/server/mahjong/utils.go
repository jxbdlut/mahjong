package mahjong

import (
	"strings"
	"sort"
)

var (
	CardsMap = map[int32]string{101: "一万", 102: "二万", 103: "三万", 104: "四万", 105: "五万", 106: "六万", 107: "七万", 108: "八万",
		109: "九万",
		201: "一饼", 202: "二饼", 203: "三饼", 204: "四饼", 205: "五饼", 206: "六饼", 207: "七饼", 208: "八饼",
		209: "九饼",
		301: "一条", 302: "二条", 303: "三条", 304: "四条", 305: "五条", 306: "六条", 307: "七条", 308: "八条",
		309: "九条",
		401: "東", 402: "西", 403: "南", 404: "北", 405: "中", 406: "發", 407: "白",
		1: "腾空", 2: "飘将"}
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

func Count(cards []int32, card int32) int32 {
	count := int32(0)
	for _, c := range cards {
		if c == card {
			count++
		}
	}
	return count
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