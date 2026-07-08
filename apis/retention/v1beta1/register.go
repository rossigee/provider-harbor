/*
Copyright 2024 Crossplane Harbor Provider.
*/

package v1beta1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"reflect"
)


// Retention type metadata.
var (
	RetentionKind             = reflect.TypeOf(Retention{}).Name()
	RetentionGroupKind        = schema.GroupKind{Group: Group, Kind: RetentionKind}
	RetentionKindAPIVersion   = RetentionKind + "." + SchemeGroupVersion.String()
	RetentionGroupVersionKind = SchemeGroupVersion.WithKind(RetentionKind)
)
}
