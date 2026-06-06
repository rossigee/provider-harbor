/*
Copyright 2024 Crossplane Harbor Provider.
*/

package v1beta1

import (
	"reflect"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Robot type metadata.
var (
	RobotKind             = reflect.TypeOf(Robot{}).Name()
	RobotGroupKind        = schema.GroupKind{Group: Group, Kind: RobotKind}
	RobotKindAPIVersion   = RobotKind + "." + SchemeGroupVersion.String()
	RobotGroupVersionKind = SchemeGroupVersion.WithKind(RobotKind)
)
