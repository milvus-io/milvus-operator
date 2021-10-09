package util

import (
	"encoding/json"
	"testing"
)

var originJson = `{
	"a": {
		"a1": "av1",
		"a2": {
			"aa2": "aav2"
		}
	},
	"b": [1,2,3],
	"c": 5
}`

var patchJson = `{
	"a": {
		"a2": {
			"aa2": "patch-aav2",
			"aa3": 123
		}
	}
}`

func TestMergeValues(t *testing.T) {
	origin := map[string]interface{}{}
	patch := map[string]interface{}{}
	if err := json.Unmarshal([]byte(originJson), &origin); err != nil {
		t.Error(err)
	}
	if err := json.Unmarshal([]byte(patchJson), &patch); err != nil {
		t.Error(err)
	}

	MergeValues(origin, patch)
	c, ok := origin["c"].(float64)
	if !ok || c != 5 {
		t.Errorf("c is not int or 5")
	}

}
