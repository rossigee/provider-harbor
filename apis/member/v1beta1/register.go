/*
Copyright 2024 Crossplane Harbor Provider.
*/

package v1beta1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"reflect"
)

var (
	MemberKind             = reflect.TypeOf(Member{}).Name()
	MemberGroupKind        = schema.GroupKind{Group: Group, Kind: MemberKind}
	MemberKindAPIVersion   = MemberKind + "." + SchemeGroupVersion.String()
	MemberGroupVersionKind = SchemeGroupVersion.WithKind(MemberKind)
)

func init() {
	SchemeBuilder.Register(&Member{}, &MemberList{})
}
