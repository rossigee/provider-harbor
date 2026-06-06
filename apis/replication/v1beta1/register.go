/*
Copyright 2024 Crossplane Harbor Provider.
*/

package v1beta1

import (
	"reflect"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Replication type metadata.
var (
	ReplicationKind             = reflect.TypeOf(Replication{}).Name()
	ReplicationGroupKind        = schema.GroupKind{Group: Group, Kind: ReplicationKind}
	ReplicationKindAPIVersion   = ReplicationKind + "." + SchemeGroupVersion.String()
	ReplicationGroupVersionKind = SchemeGroupVersion.WithKind(ReplicationKind)
)

func init() {
	SchemeBuilder.Register(&Replication{}, &ReplicationList{})
}
