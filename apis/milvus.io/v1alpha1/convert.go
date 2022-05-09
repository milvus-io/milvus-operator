package v1alpha1

import (
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

// ConvertTo converts this Milvus to the Hub version (v1beta1).
func (r *Milvus) ConvertTo(dstRaw conversion.Hub) error {
	panic("todo")
}

// ConvertFrom converts from the Hub version (v1beta1) to this version.
func (r *Milvus) ConvertFrom(srcRaw conversion.Hub) error {
	panic("todo")
}
