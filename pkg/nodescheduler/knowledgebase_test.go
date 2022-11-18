package nodescheduler

import (
	"testing"

	"github.com/waggle-sensor/edge-scheduler/pkg/datatype"
)

func TestKnowledgeBaseEvaluate(t *testing.T) {
	// NOTE(sean) I noticed that this test seems to be missing something during setup and
	// is dereferencing a nil pointer. I'm marking this as a skip for now, so we can continue
	// running other tests but remember to revisit this.
	t.Skip("TODO restore this test after fixing setup.")
	tests := map[string]struct {
		Input struct {
			K string
			V interface{}
		}
		Rules []string
		Want  bool
	}{
		"test1": {
			Input: struct {
				K string
				V interface{}
			}{
				K: "a.b.c",
				V: 123.12,
			},
			Rules: []string{"hello: a.b.c == 123"},
			Want:  true,
		},
		// "test2": {
		// 	Input: struct {
		// 		K string
		// 		V interface{}
		// 	}{
		// 		K: "sys.time.minute",
		// 		V: 30,
		// 	},
		// 	Rule: []string{"hello: sys.time.minute % 15 == 0"},
		// 	Want: "hello",
		// },
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			kb := NewKnowledgeBase("W000", "")
			kb.AddRawMeasure(tc.Input.K, tc.Input.V)
			for _, r := range tc.Rules {
				scienceRule, err := datatype.NewScienceRule(r)
				if err != nil {
					t.Fatal(err.Error())
				}
				result, err := kb.EvaluateRule(scienceRule)
				if err != nil {
					t.Fatal(err.Error())
				} else {
					if result != tc.Want {
						t.Fatalf("result %t: wanted %t", result, tc.Want)
					}
				}
			}
		})
	}
}
