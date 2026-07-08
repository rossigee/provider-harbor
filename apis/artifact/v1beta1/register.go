/*
Copyright 2024 Crossplane Harbor Provider.
*/

package v1beta1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"reflect"
)


// Artifact type metadata.
var (
	ArtifactKind             = reflect.TypeOf(Artifact{}).Name()
	ArtifactGroupKind        = schema.GroupKind{Group: Group, Kind: ArtifactKind}
	ArtifactKindAPIVersion   = ArtifactKind + "." + SchemeGroupVersion.String()
	ArtifactGroupVersionKind = SchemeGroupVersion.WithKind(ArtifactKind)
)
}
