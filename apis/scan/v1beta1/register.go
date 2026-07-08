/*
Copyright 2024 Crossplane Harbor Provider.
*/

package v1beta1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"reflect"
)


var (
	ScanKind             = reflect.TypeOf(Scan{}).Name()
	ScanGroupKind        = schema.GroupKind{Group: Group, Kind: ScanKind}
	ScanKindAPIVersion   = ScanKind + "." + SchemeGroupVersion.String()
	ScanGroupVersionKind = SchemeGroupVersion.WithKind(ScanKind)
)
}
