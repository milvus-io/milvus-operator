package util

import (
	"encoding/json"
	"errors"
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/yaml"
)

func Test_GetBoolValue_SetValue_DeleteValue(t *testing.T) {
	origin := map[string]interface{}{}

	_, found := GetBoolValue(origin, "l1", "l2", "l3")
	assert.False(t, found)

	SetValue(origin, true, "l1", "l2", "l3")

	val, found := GetBoolValue(origin, "l1", "l2", "l3")
	assert.True(t, found)
	assert.True(t, val)

	DeleteValue(origin, "l1", "l2", "l3")
	_, found = GetBoolValue(origin, "l1", "l2", "l3")
	assert.False(t, found)

}

func TestSetStringSlice(t *testing.T) {
	origin := map[string]interface{}{}
	slice := []string{"v1", "v2"}
	SetStringSlice(origin, slice, "l1", "l2", "l3")
	values := origin["l1"].(map[string]interface{})["l2"].(map[string]interface{})["l3"].([]interface{})
	assert.Equal(t, slice[0], values[0].(string))
	assert.Equal(t, slice[1], values[1].(string))
}

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

func TestGetHostPort(t *testing.T) {
	endPoint := "host:8080"
	host, port := GetHostPort(endPoint)
	assert.Equal(t, "host", host)
	assert.Equal(t, int32(8080), port)

	endPoint = "hostOnly"
	host, port = GetHostPort(endPoint)
	assert.Equal(t, "hostOnly", host)
	assert.Equal(t, int32(80), port)

	endPoint = "host:badPort"
	host, port = GetHostPort(endPoint)
	assert.Equal(t, "host", host)
	assert.Equal(t, int32(80), port)
}

func TestGetTemplatedValues(t *testing.T) {
	template := `
k1: v1
k2: {{ .k2 }}
`
	values := map[string]interface{}{
		"k2": "v2",
	}
	ret, err := GetTemplatedValues(template, values)
	assert.NoError(t, err)
	config := map[string]interface{}{}
	yaml.Unmarshal(ret, &config)
	log.Print(string(ret))
	assert.Equal(t, "v1", config["k1"])
	assert.Equal(t, "v2", config["k2"])
}

func TestGetTemplatedValues_FormatError(t *testing.T) {
	template := `
k1: v1
k2: {{ .k2
`
	values := map[string]interface{}{}
	_, err := GetTemplatedValues(template, values)
	assert.Error(t, err)
}

func TestJoinErrors(t *testing.T) {
	errs := []error{}
	assert.Empty(t, JoinErrors(errs))

	errs = []error{errors.New("e1")}
	assert.Equal(t, "e1", JoinErrors(errs))

	errs = []error{errors.New("e1"), errors.New("e2")}
	assert.Equal(t, "e1; e2", JoinErrors(errs))
}

func TestCheckSum(t *testing.T) {
	assert.Equal(t, "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824", CheckSum([]byte("hello")))
}
