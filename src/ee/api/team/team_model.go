package team

import (
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"gopkg.in/guregu/null.v3"
)

type Team struct {
	ID              types.ID `json:"id,string"`
	Name            string   `json:"name"`
	Slug            string   `json:"slug"`
	IsDefault       bool     `json:"isDefault"`
	CurrentUserRole string   `json:"currentUserRole"`
}

// ToMap is similar to json.Marshal but returns a map instead.
func (t Team) ToMap() map[string]any {
	m := map[string]any{}
	m["id"] = t.ID.String()
	m["name"] = t.Name
	m["currentUserRole"] = t.CurrentUserRole
	m["isDefault"] = t.IsDefault
	m["slug"] = t.Slug

	return m
}

type Member struct {
	ID          types.ID    `json:"id,string"`
	TeamID      types.ID    `json:"teamId,string"`
	UserID      types.ID    `json:"userId,string"`
	InvitedBy   types.ID    `json:"-"`
	FirstName   null.String `json:"firstName"`
	LastName    null.String `json:"lastName"`
	Avatar      null.String `json:"avatar"`
	DisplayName string      `json:"displayName"`
	Email       string      `json:"email"`
	Role        string      `json:"role"`
	Status      bool        `json:"status"`
}

// HasWriteAccess checks if the role is either Admin or Owner.
func HasWriteAccess(role string) bool {
	return role == ROLE_ADMIN || role == ROLE_OWNER
}

// IsValidRole validates the given string and returns true if it matches our defined roles.
func IsValidRole(role string) bool {
	if role != ROLE_ADMIN && role != ROLE_DEVELOPER && role != ROLE_OWNER {
		return false
	}

	return true
}
