package contracts

import "time"

type User struct { // want User:`v1:User:`
	ID       string `json:"id"`
	LegacyID string `json:"id"` // want `TAG104 duplicate json name "id"`

	Email string `json:"email,omitempty" validate:"required,email"` // want `TAG204 "omitempty" in json conflicts with validation rule "required"`

	PasswordHash string `json:"password"` // want `TAG401 sensitive field PasswordHash`

	DisplayName string `json:"display_name" yaml:"displayName"` // want `TAG301 external naming drift` `TAG301 yaml name "displayName" does not follow configured snake_case`

	CreatedAt time.Time `json:"created_at,omitempty"` // want `TAG202 json option "omitempty" cannot omit zero-valued time.Time`

	internal string `json:"internal"` // want `TAG101 tag on unexported field`
}

type IdentityA struct { // want IdentityA:`v1:IdentityA:`
	Primary string `json:"id"`
}

type IdentityB struct { // want IdentityB:`v1:IdentityB:`
	Legacy string `json:"id"` // want `TAG105 duplicate json name "id"`
}

type Combined struct { // want Combined:`v1:Combined:`
	IdentityA
	IdentityB
}
