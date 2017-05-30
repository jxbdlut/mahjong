package area

import (
	"errors"
	"fmt"
	"reflect"
	"math"
	"github.com/jxbdlut/leaf/log"
)

var (
	ruleInfo      []*RuleInfo
	ruleID        map[reflect.Type]uint16
)

type RuleInfo struct {
	AreaType reflect.Type
}
func Init() {
	ruleID = make(map[reflect.Type]uint16)
	Register(&DefaultRule{})
}

func Register(rule Rule) error {
	ruleType := reflect.TypeOf(rule)
	if _, ok := ruleID[ruleType]; ok {
		return errors.New(fmt.Sprintf("%v has areadly register", rule))
	}
	if len(ruleInfo) >= math.MaxUint16 {
		log.Fatal("too many protobuf messages (max = %v)", math.MaxUint16)
	}
	i := new(RuleInfo)
	i.AreaType = ruleType
	ruleInfo = append(ruleInfo, i)
	ruleID[ruleType] = uint16(len(ruleInfo) - 1)
	return nil
}

func GetArea(rule_id uint16) Rule {
	switch rule_id {
	case 0:
		return NewDefaultRule()
	}
	return NewDefaultRule()
}
