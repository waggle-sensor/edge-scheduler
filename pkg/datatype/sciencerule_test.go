package datatype

import (
	"reflect"
	"testing"
)

type ScienceRuleTestWants struct {
	ShouldFailToParse bool
	ActionType        ScienceRuleActionType
	ActionObject      string
	ActionArguments   map[string]string
}

func TestScienceRule(t *testing.T) {
	tests := map[string]struct {
		ScienceRule string
		Wants       ScienceRuleTestWants
	}{
		"Incomplete science rule": {
			ScienceRule: "schedule(plugin-a: True",
			Wants: ScienceRuleTestWants{
				ShouldFailToParse: true,
			},
		},
		"Wrong Schedule type": {
			ScienceRule: "schedul(plugin-b): True",
			Wants: ScienceRuleTestWants{
				ShouldFailToParse: true,
			},
		},
		"Schedule type test1": {
			ScienceRule: "schedule(plugin-b): True",
			Wants: ScienceRuleTestWants{
				ShouldFailToParse: false,
				ActionType:        ScienceRuleActionSchedule,
				ActionObject:      "plugin-b",
				ActionArguments:   nil,
			},
		},
		"Schedule type test2": {
			ScienceRule: "schedule(plugin-b, duration=5m): True",
			Wants: ScienceRuleTestWants{
				ShouldFailToParse: false,
				ActionType:        ScienceRuleActionSchedule,
				ActionObject:      "plugin-b",
				ActionArguments: map[string]string{
					"duration": "5m",
				},
			},
		},
		"Publish type test1": {
			ScienceRule: "publish(env.event.cloudmotion.fast, to=cloud): True",
			Wants: ScienceRuleTestWants{
				ShouldFailToParse: false,
				ActionType:        ScienceRuleActionPublish,
				ActionObject:      "env.event.cloudmotion.fast",
				ActionArguments: map[string]string{
					"to": "cloud",
				},
			},
		},
		"Set type test1": {
			ScienceRule: "set(sys.time.sunrise, value=06:39:00): True",
			Wants: ScienceRuleTestWants{
				ShouldFailToParse: false,
				ActionType:        ScienceRuleActionSet,
				ActionObject:      "sys.time.sunrise",
				ActionArguments: map[string]string{
					"value": "06:39:00",
				},
			},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			r, err := NewScienceRule(test.ScienceRule)
			if test.Wants.ShouldFailToParse {
				if err == nil {
					t.Errorf("Wanted to fail parsing rule %q but succeded", test.ScienceRule)
				}
				return
			}
			if err != nil {
				t.Errorf("Failed to parse science rule %q", test.ScienceRule)
				return
			}
			if r.ActionType != test.Wants.ActionType {
				t.Errorf("Wanted action type %q but found %q", test.Wants.ActionType, r.ActionType)
				return
			}
			if r.ActionObject != test.Wants.ActionObject {
				t.Errorf("Wanted action object %q but found %q", test.Wants.ActionObject, r.ActionObject)
				return
			}
			if test.Wants.ActionArguments != nil {
				if !reflect.DeepEqual(r.ActionParameters, test.Wants.ActionArguments) {
					t.Errorf("Wanted action parameters %q but does not match with %q", test.Wants.ActionArguments, r.ActionParameters)
					return
				}
			}
		})
	}
}
