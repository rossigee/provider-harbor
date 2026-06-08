/*
Copyright 2024 Crossplane Harbor Provider.
*/

package clients

// UserGroupSpec defines the desired state of a UserGroup
type UserGroupSpec struct {
	GroupName   string  `json:"groupName"`
	GroupType   int64   `json:"groupType"`
	LdapGroupDn *string `json:"ldapGroupDn,omitempty"`
}

// UserGroupStatus represents the observed state of a UserGroup
type UserGroupStatus struct {
	ID          int64
	GroupName   string
	GroupType   int64
	LdapGroupDn string
}
