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
	rules          map[string][]datatype.ScienceRule
	measures       map[string]interface{}
	ruleCheckerURI string
}

func NewKnowledgeBase(nodeID string, ruleCheckerURI string) *KnowledgeBase {
	return &KnowledgeBase{
		nodeID:         nodeID,
		rules:          make(map[string][]datatype.ScienceRule),
		measures:       map[string]interface{}{},
		ruleCheckerURI: ruleCheckerURI,
	}
}

// Archived
// func (kb *KnowledgeBase) add(obj interface{}, k string, v interface{}) {
// 	currentKB := obj.(map[string]interface{})
// 	keys := strings.SplitN(k, ".", 2)
// 	if len(keys) == 1 {
// 		currentKB[keys[0]] = v
// 	} else {
// 		if nextKB, exist := currentKB[keys[0]]; !exist {
// 			currentKB[keys[0]] = map[string]interface{}{}
// 			kb.add(currentKB[keys[0]], keys[1], v)
// 		} else {
// 			kb.add(nextKB, keys[1], v)
// 		}
// 	}
// }

func (kb *KnowledgeBase) AddRulesFromScienceGoal(s *datatype.ScienceGoal) error {
	if mySubGoal := s.GetMySubGoal(kb.nodeID); mySubGoal != nil {
		// This is to make sure the rules are parsed before evaluated
		parsedScienceRules := []datatype.ScienceRule{}
		for _, r := range mySubGoal.ScienceRules {
			if err := r.Parse(r.Rule); err != nil {
				logger.Error.Printf("Failed to parse ScienceRule %q: %s", r.Rule, err.Error())
			}
			parsedScienceRules = append(parsedScienceRules, r)
		}
		kb.rules[s.ID] = parsedScienceRules
		return nil
	} else {
		return fmt.Errorf("failed to find my sub goal from science goal %q", s.ID)
	}
}

func (kb *KnowledgeBase) DropRules(goalID string) {
	delete(kb.rules, goalID)
}

// Archived
func (kb *KnowledgeBase) AddRawMeasure(k string, v interface{}) {
	logger.Debug.Printf("Added raw measure %q:%s", k, v)
	// kb.add(kb.measures, k, v)
	r := interfacing.NewHTTPRequest(kb.ruleCheckerURI)
	data, _ := json.Marshal(map[string]interface{}{
		"key":   k,
		"value": v,
	})
	resp, err := r.RequestPost("store", data, nil)

	decoder, err := r.ParseJSONHTTPResponse(resp)
	if err != nil {

	}
	var body map[string]interface{}
	decoder.Decode(&body)
	logger.Debug.Printf("%v", body)
	// logger.Debug.Printf("Added raw measure %q:%s", k, v.(string))
	// v, err := strconv.ParseFloat(v.(string), 64)
	// if err != nil {
	// 	kb.measures[k] = v.(string)
	// } else {
	// 	kb.measures[k] = v
	// }
}

func (kb *KnowledgeBase) EvaluateRule(rule *datatype.ScienceRule) (bool, error) {
	r := interfacing.NewHTTPRequest(kb.ruleCheckerURI)
	data, _ := json.Marshal(map[string]interface{}{
		"rule": rule.Condition,
	})
	resp, err := r.RequestPost("evaluate", data, nil)
	if err != nil {
		return false, fmt.Errorf("failed to get data from checker: %s", err.Error())
	}
	decoder, err := r.ParseJSONHTTPResponse(resp)
	if err != nil {
		return false, fmt.Errorf("failed to parse response: %s", err.Error())
	}
	var body map[string]interface{}
	decoder.Decode(&body)
	if r, exists := body["response"]; exists {
		if r.(string) == "failed" {
			return false, fmt.Errorf("failed to evaluate rule: %s", body["error"])
		}
	}
	if v, exists := body["result"]; exists {
		return v.(bool), nil
	} else {
		return false, fmt.Errorf("response does not contain result: %v", body)
	}
}

func (kb *KnowledgeBase) EvaluateGoal(goalID string) (results []datatype.ScienceRule, err error) {
	if rules, exist := kb.rules[goalID]; exist {
		for _, rule := range rules {
			if valid, err := kb.EvaluateRule(&rule); err != nil {
				logger.Error.Printf("Failed to evaluate rule %q: %s", rule, err.Error())
			} else if valid {
				results = append(results, rule)
			}
		}
	} else {
		err = fmt.Errorf("failed to find rules: rules for goal ID %q does not exist", goalID)
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
	return "", "", fmt.Errorf("failed to split %q using \":\"", r)
}
