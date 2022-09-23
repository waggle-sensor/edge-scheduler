package nodescheduler

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/waggle-sensor/edge-scheduler/pkg/datatype"
	"github.com/waggle-sensor/edge-scheduler/pkg/interfacing"
	"github.com/waggle-sensor/edge-scheduler/pkg/logger"
)

type KnowledgeBase struct {
	nodeID         string
	rules          map[string][]string
	measures       map[string]interface{}
	ruleCheckerURI string
}

func NewKnowledgeBase(nodeID string, ruleCheckerURI string) *KnowledgeBase {
	return &KnowledgeBase{
		nodeID:         nodeID,
		rules:          make(map[string][]string),
		measures:       map[string]interface{}{},
		ruleCheckerURI: ruleCheckerURI,
	}
}

func (kb *KnowledgeBase) add(obj interface{}, k string, v interface{}) {
	currentKB := obj.(map[string]interface{})
	keys := strings.SplitN(k, ".", 2)
	if len(keys) == 1 {
		currentKB[keys[0]] = v
	} else {
		if nextKB, exist := currentKB[keys[0]]; !exist {
			currentKB[keys[0]] = map[string]interface{}{}
			kb.add(currentKB[keys[0]], keys[1], v)
		} else {
			kb.add(nextKB, keys[1], v)
		}
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
	logger.Debug.Printf("Added raw measure %q:%s", k, v)
	// kb.add(kb.measures, k, v)
	r := interfacing.NewHTTPRequest(kb.ruleCheckerURI)
	data, _ := json.Marshal(map[string]interface{}{
		"key":   k,
		"value": v,
	})
	resp, err := r.RequestPost("store", data, nil)

	body, err := r.ParseJSONHTTPResponse(resp)
	if err != nil {

	}
	logger.Debug.Printf("%v", body)
	// logger.Debug.Printf("Added raw measure %q:%s", k, v.(string))
	// v, err := strconv.ParseFloat(v.(string), 64)
	// if err != nil {
	// 	kb.measures[k] = v.(string)
	// } else {
	// 	kb.measures[k] = v
	// }
}

func (kb *KnowledgeBase) AddMeasure(v *datatype.WaggleMessage) {

}

func (kb *KnowledgeBase) EvaluateRule(rule string) (string, error) {
	condition, result, err := parseRule(rule)
	if err != nil {
		return "", fmt.Errorf("Failed to parse rule")
	}
	r := interfacing.NewHTTPRequest(kb.ruleCheckerURI)
	data, _ := json.Marshal(map[string]interface{}{
		"rule": condition,
	})
	resp, err := r.RequestPost("evaluate", data, nil)
	if err != nil {
		return "", fmt.Errorf("Failed to get data from checker: %s", err.Error())
	}
	body, err := r.ParseJSONHTTPResponse(resp)
	if err != nil {
		return "", fmt.Errorf("Failed to parse response: %s", err.Error())
	}
	if r, exists := body["response"]; exists {
		if r.(string) == "failed" {
			return "", fmt.Errorf("Failed to evaluate rule: %s", body["error"])
		}
	}
	if v, exists := body["result"]; exists {
		if v.(bool) == true {
			return result, nil
		} else {
			return "", nil
		}
	} else {
		return "", fmt.Errorf("Response does not contain result: %v", body)
	}
	// if parser.Evaluate(condition, kb.measures) {
	// 	return result, nil
	// } else {
	// 	return "", nil
	// }
}

func (kb *KnowledgeBase) EvaluateGoal(goalID string) (results []string, err error) {
	if rules, exist := kb.rules[goalID]; exist {
		for _, rule := range rules {
			if result, err := kb.EvaluateRule(rule); err != nil {
				logger.Error.Printf("Failed to evaluate rule %q: %s", rule, err.Error())
			} else if result != "" {
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
