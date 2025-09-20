/*
Copyright 2024 Crossplane Harbor Provider.
*/

package v1alpha1

import (
	"reflect"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Package type metadata.
const (
	UserKind = "User"
)

func init() {
	SchemeBuilder.Register(&User{}, &UserList{})
}

// User type metadata.
var (
	UserKindValue        = reflect.TypeOf(User{}).Name()
	UserGroupKind        = schema.GroupKind{Group: Group, Kind: UserKind}.String()
	UserKindAPIVersion   = UserKind + "." + SchemeGroupVersion.String()
	UserGroupVersionKind = SchemeGroupVersion.WithKind(UserKind)
)