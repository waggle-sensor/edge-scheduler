package main

import (
	"fmt"
	"reflect"
	"testing"
)

func TestParseSelector(t *testing.T) {
	testcases := []struct {
		Input string
		Want  map[string]string
	}{
		{"", map[string]string{}},
		{"resource.gps=true", map[string]string{"resource.gps": "true"}},
		{"resource.gps=true,resource.bme680=true", map[string]string{"resource.gps": "true", "resource.bme680": "true"}},
	}

	for _, tc := range testcases {
		t.Run(fmt.Sprintf("input %q", tc.Input), func(t *testing.T) {
			r, err := parseSelector(tc.Input)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(r, tc.Want) {
				t.Fatal(fmt.Errorf("expected empty"))
			}
		})
	}
}
