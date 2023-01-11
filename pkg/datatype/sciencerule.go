package datatype

import (
	"fmt"
	"regexp"
	"strings"
)

type ScienceRule struct {
	Rule             string                `json:"rule" yaml:"rule"`
	ActionType       ScienceRuleActionType `json:"-" yaml:"-"`
	ActionObject     string                `json:"-" yaml:"-"`
	ActionParameters map[string]string     `json:"-" yaml:"-"`
	Condition        string                `json:"-" yaml:"-"`
}

func NewScienceRule(rule string) (*ScienceRule, error) {
	scienceRule := ScienceRule{}
	if err := scienceRule.Parse(rule); err != nil {
		return nil, err
	} else {
		return &scienceRule, nil
	}
}

type ScienceRuleActionType string

const (
	ScienceRuleActionSchedule ScienceRuleActionType = "schedule"
	ScienceRuleActionPublish  ScienceRuleActionType = "publish"
	ScienceRuleActionSet      ScienceRuleActionType = "set"
)

func (r *ScienceRule) Parse(rule string) error {
	r.Rule = rule
	re := regexp.MustCompile(`^(\w+)\((.*?)\):(.*?)$`)
	sp := re.FindStringSubmatch(r.Rule)
	// length of sp should be 4: the rule, action, parameters, and condition
	if len(sp) != 4 {
		return fmt.Errorf("Failed to parse rule %q: rule must consist of an action, target object, and corresponding condition:", r.Rule)
	}
	switch strings.Trim(sp[1], " ") {
	case string(ScienceRuleActionSchedule):
		r.ActionType = ScienceRuleActionSchedule
	case string(ScienceRuleActionPublish):
		r.ActionType = ScienceRuleActionPublish
	case string(ScienceRuleActionSet):
		r.ActionType = ScienceRuleActionSet
	default:
		return fmt.Errorf("Failed to parse rule %q: unknown action type %q found", r.Rule, strings.Trim(sp[0], " "))
	}
	actionParams := strings.Split(sp[2], ",")
	if len(actionParams) < 1 {
		return fmt.Errorf("Failed to parse rule %q: no action object found", r.Rule)
	}
	r.ActionObject = strings.Trim(actionParams[0], " ")
	r.ActionObject = strings.Trim(r.ActionObject, `'`)
	r.ActionObject = strings.Trim(r.ActionObject, `"`)
	r.ActionParameters = make(map[string]string)
	for _, param := range actionParams[1:] {
		kv := strings.Split(param, "=")
		if len(kv) != 2 {
			return fmt.Errorf("Failed to parse rule %q: failed to parse param %s", r.Rule, param)
		} else {
			v := strings.Trim(kv[1], " ")
			v = strings.Trim(v, `"`)
			v = strings.Trim(v, `'`)
			r.ActionParameters[strings.Trim(kv[0], " ")] = v
		}
	}
	r.Condition = strings.Trim(sp[3], " ")
	return nil
}
