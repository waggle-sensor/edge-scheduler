package nodescheduler

import "testing"

func TestKnowledgeBaseEvaluate(t *testing.T) {
	tests := map[string]struct {
		Input struct {
			K string
			V interface{}
		}
		Rule []string
		Want string
	}{
		"test1": {
			Input: struct {
				K string
				V interface{}
			}{
				K: "a.b.c",
				V: 123,
			},
			Rule: []string{"hello: a.b.c > 100"},
			Want: "hello",
		},
		"test2": {
			Input: struct {
				K string
				V interface{}
			}{
				K: "a.b.c",
				V: 10,
			},
			Rule: []string{"hello: a.b.c == 10"},
			Want: "hello",
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			kb := NewKnowledgeBase("W000")
			kb.AddRawMeasure(tc.Input.K, tc.Input.V)
			for _, r := range tc.Rule {
				result, err := kb.EvaluateRule(r)
				if err != nil {
					t.Fatal(err.Error())
				} else {
					if result != tc.Want {
						t.Fatalf("result %s: wanted %s", result, tc.Want)
					}
				}
			}

		})
	}
}
