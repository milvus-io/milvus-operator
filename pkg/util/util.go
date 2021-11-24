package util

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"net"
	"strconv"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func GetBoolValue(values map[string]interface{}, fields ...string) (bool, bool) {
	val, found, err := unstructured.NestedBool(values, fields...)
	if err != nil || !found {
		return false, false
	}

	return val, true
}

func DeleteValue(values map[string]interface{}, fields ...string) {
	unstructured.RemoveNestedField(values, fields...)
}

//only contains types bool, int64, float64, string, []interface{}, map[string]interface{}, json.Number and nil
func SetValue(values map[string]interface{}, v interface{}, fields ...string) {
	unstructured.SetNestedField(values, v, fields...)
}

func SetStringSlice(values map[string]interface{}, v []string, fields ...string) {
	unstructured.SetNestedStringSlice(values, v, fields...)
}

func MergeValues(origin, patch map[string]interface{}) {
	for patchK, patchV := range patch {
		if _, exist := origin[patchK]; !exist {
			origin[patchK] = patchV
			continue
		}

		originValues, ok := origin[patchK].(map[string]interface{})
		if !ok {
			origin[patchK] = patchV
			continue
		}

		patchValues, ok := patchV.(map[string]interface{})
		if !ok {
			origin[patchK] = patchV
			continue
		}

		MergeValues(originValues, patchValues)
	}
}

func GetHostPort(endpoint string) (string, int32) {
	host, port, err := net.SplitHostPort(endpoint)
	if err != nil {
		return endpoint, 80
	}

	portInt, err := strconv.Atoi(port)
	if err != nil {
		return host, 80
	}

	return host, int32(portInt)
}

func GetTemplatedValues(templateConfig string, values interface{}) ([]byte, error) {
	t, err := template.New("template").
		Funcs(sprig.TxtFuncMap()).Parse(templateConfig)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	err = t.Execute(&buf, values)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func JoinErrors(errs []error) string {
	es := make([]string, 0, len(errs))
	for _, e := range errs {
		es = append(es, e.Error())
	}
	return strings.Join(es, "; ")
}

func CheckSum(s []byte) string {
	h := sha256.New()
	h.Write(s)
	return fmt.Sprintf("%x", h.Sum(nil))
}
