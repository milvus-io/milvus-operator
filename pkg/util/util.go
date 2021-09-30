package util

import (
	"bytes"
	"net"
	neturl "net/url"
	"strconv"
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

func SetValue(values map[string]interface{}, v interface{}, fields ...string) {
	unstructured.SetNestedField(values, v, fields...)
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

func GetHostPortFromURL(url string) (string, int32) {
	u, err := neturl.ParseRequestURI(url)
	if err != nil {
		return url, 80
	}

	portInt, err := strconv.Atoi(u.Port())
	if err != nil {
		return u.Hostname(), 80
	}

	return u.Hostname(), int32(portInt)
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
