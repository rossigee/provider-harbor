/*
Copyright 2024 Crossplane Harbor Provider.
*/

package v1beta1

import (
	"reflect"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Webhook type metadata.
var (
	WebhookKind             = reflect.TypeOf(Webhook{}).Name()
	WebhookGroupKind        = schema.GroupKind{Group: Group, Kind: WebhookKind}
	WebhookKindAPIVersion   = WebhookKind + "." + SchemeGroupVersion.String()
	WebhookGroupVersionKind = SchemeGroupVersion.WithKind(WebhookKind)
)
