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

// Registry type metadata.
var (
	RegistryKind             = reflect.TypeOf(Registry{}).Name()
	RegistryGroupKind        = schema.GroupKind{Group: Group, Kind: RegistryKind}
	RegistryKindAPIVersion   = RegistryKind + "." + SchemeGroupVersion.String()
	RegistryGroupVersionKind = SchemeGroupVersion.WithKind(RegistryKind)
)

// RegistryCredential type metadata.
var (
	RegistryCredentialKind             = reflect.TypeOf(RegistryCredential{}).Name()
	RegistryCredentialGroupKind        = schema.GroupKind{Group: Group, Kind: RegistryCredentialKind}
	RegistryCredentialKindAPIVersion   = RegistryCredentialKind + "." + SchemeGroupVersion.String()
	RegistryCredentialGroupVersionKind = SchemeGroupVersion.WithKind(RegistryCredentialKind)
)
