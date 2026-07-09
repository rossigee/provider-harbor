/*
Copyright 2025 The Crossplane Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
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

// RobotPermission type metadata.
var (
	RobotPermissionKind             = reflect.TypeOf(RobotPermission{}).Name()
	RobotPermissionGroupKind        = schema.GroupKind{Group: Group, Kind: RobotPermissionKind}
	RobotPermissionKindAPIVersion   = RobotPermissionKind + "." + SchemeGroupVersion.String()
	RobotPermissionGroupVersionKind = SchemeGroupVersion.WithKind(RobotPermissionKind)
)
