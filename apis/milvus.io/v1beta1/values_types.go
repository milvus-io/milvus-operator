package v1beta1

import (
	"encoding/json"
	"fmt"
	"reflect"
)

type Values struct {
	// Work around for https://github.com/kubernetes-sigs/kubebuilder/issues/528
	Data map[string]interface{} `json:"-"`
}

func (v Values) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.Data)
}

func (v *Values) UnmarshalJSON(data []byte) error {
	var out map[string]interface{}

	err := json.Unmarshal(data, &out)
	if err != nil {
		return err
	}

	v.Data = out

	return nil
}

// DeepCopyInto is an deepcopy function, copying the receiver, writing
// into out. In must be non-nil. Declaring this here prevents it from
// being generated in `zz_generated.deepcopy.go`.
//
// This exists here to work around https://github.com/kubernetes/code-generator/issues/50,
// and partially around https://github.com/kubernetes-sigs/controller-tools/pull/126
// and https://github.com/kubernetes-sigs/controller-tools/issues/294.
func (v *Values) DeepCopyInto(out *Values) {
	b, err := json.Marshal(v.Data)
	if err != nil {
		// The marshal should have been performed cleanly as otherwise
		// the resource would not have been created by the API server.
		panic(err)
	}

	var c map[string]interface{}

	err = json.Unmarshal(b, &c)
	if err != nil {
		panic(err)
	}

	out.Data = c
}

func (v Values) MustAsObj(obj interface{}) {
	err := v.AsObject(obj)
	if err != nil {
		panic(err)
	}
	return
}

func (v Values) AsObject(obj interface{}) error {
	marshaled, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("failed to marshal values: %v", err)
	}
	if err := json.Unmarshal(marshaled, obj); err != nil {
		return fmt.Errorf("failed to unmarshal values as obj[%s]: %v", reflect.TypeOf(obj), err)
	}
	return nil
}
