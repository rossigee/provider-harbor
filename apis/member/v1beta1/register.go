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

	UserMemberKind             = reflect.TypeOf(UserMember{}).Name()
	UserMemberGroupKind        = schema.GroupKind{Group: Group, Kind: UserMemberKind}
	UserMemberKindAPIVersion   = UserMemberKind + "." + SchemeGroupVersion.String()
	UserMemberGroupVersionKind = SchemeGroupVersion.WithKind(UserMemberKind)

	GroupMemberKind             = reflect.TypeOf(GroupMember{}).Name()
	GroupMemberGroupKind        = schema.GroupKind{Group: Group, Kind: GroupMemberKind}
	GroupMemberKindAPIVersion   = GroupMemberKind + "." + SchemeGroupVersion.String()
	GroupMemberGroupVersionKind = SchemeGroupVersion.WithKind(GroupMemberKind)
)

func init() {
	SchemeBuilder.Register(&Member{}, &MemberList{})
	SchemeBuilder.Register(&UserMember{}, &UserMemberList{})
	SchemeBuilder.Register(&GroupMember{}, &GroupMemberList{})
}
