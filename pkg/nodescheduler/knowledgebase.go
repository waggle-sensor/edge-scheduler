package nodescheduler

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/sagecontinuum/ses/pkg/datatype"
	"github.com/sagecontinuum/ses/pkg/logger"

	"github.com/nikunjy/rules/parser"
)

type KnowledgeBase struct {
	nodeID   string
	rules    map[string][]string
	measures map[string]interface{}
}

func NewKnowledgeBase(nodeID string) *KnowledgeBase {
	return &KnowledgeBase{
		nodeID:   nodeID,
		rules:    make(map[string][]string),
		measures: map[string]interface{}{},
	}
}

func (kb *KnowledgeBase) AddRulesFromScienceGoal(s *datatype.ScienceGoal) {
	mySubGoal := s.GetMySubGoal(kb.nodeID)
	kb.rules[s.ID] = mySubGoal.ScienceRules
}

func (kb *KnowledgeBase) DropRules(goalID string) {
	if _, exist := kb.rules[goalID]; exist {
		delete(kb.rules, goalID)
	}
}

func (kb *KnowledgeBase) AddRawMeasure(k string, v interface{}) {
	logger.Debug.Printf("Added raw measure %q:%s", k, v.(string))
	v, err := strconv.ParseFloat(v.(string), 64)
	if err != nil {
		kb.measures[k] = v.(string)
	} else {
		kb.measures[k] = v
	}

}

func (kb *KnowledgeBase) AddMeasure(v *datatype.WaggleMessage) {

}

func (kb *KnowledgeBase) Evaluate(goalID string) (results []string, err error) {
	if rules, exist := kb.rules[goalID]; exist {
		for _, rule := range rules {
			condition, result, err := parseRule(rule)
			if err != nil {
				logger.Error.Printf("Failed to parse rule %q: %s", rule, err.Error())
			}
			if parser.Evaluate(condition, kb.measures) {
				results = append(results, result)
			}
		}
	} else {
		err = fmt.Errorf("Failed to find rules: rules for goal ID %q does not exist", goalID)
	}
	return
}

func (kb *KnowledgeBase) Run() {
	time.AfterFunc(duration(), func() {
		t := time.Now()
		kb.AddRawMeasure("minutevariable", t.Minute())
	})

}

func duration() time.Duration {
	t := time.Now()
	n := t.Add(time.Minute)
	n = n.Truncate(time.Minute)
	d := n.Sub(t)
	return d
}

func parseRule(r string) (condition string, result string, err error) {
	sp := strings.Split(r, ":")
	if len(sp) == 2 {
		return strings.TrimSpace(sp[1]), strings.TrimSpace(sp[0]), nil
	}
	return "", "", fmt.Errorf("Failed to split %q using \":\"", r)
}
