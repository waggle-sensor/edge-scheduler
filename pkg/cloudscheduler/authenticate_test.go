package cloudscheduler

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestAuthentication(t *testing.T) {
	content := `
[{"vsn":"V001","access":["schedule"]},{"vsn":"V002","access":["schedule"]}]
	`
	decoder := json.NewDecoder(bytes.NewReader([]byte(content)))
	type n struct {
		Vsn    string   `json:"vsn"`
		Access []string `json:"Access"`
	}
	var aa []n
	decoder.Decode(&aa)
	t.Log(aa)
}
