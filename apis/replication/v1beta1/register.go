/*
Copyright 2024 Crossplane Harbor Provider.
*/

package v1beta1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"reflect"
)


// Replication type metadata.
var (
	ReplicationKind             = reflect.TypeOf(Replication{}).Name()
	ReplicationGroupKind        = schema.GroupKind{Group: Group, Kind: ReplicationKind}
	ReplicationKindAPIVersion   = ReplicationKind + "." + SchemeGroupVersion.String()
	ReplicationGroupVersionKind = SchemeGroupVersion.WithKind(ReplicationKind)
)
}
