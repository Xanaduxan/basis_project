package domain

import "errors"

type TeamRole string

const (
	TeamRoleOwner  TeamRole = "owner"
	TeamRoleAdmin  TeamRole = "admin"
	TeamRoleMember TeamRole = "member"
)

var (
	ErrForbidden               = errors.New("forbidden")
	ErrTeamMemberAlreadyExists = errors.New("team member already exists")
)

type Team struct {
	ID        int64
	Name      string
	CreatedBy int64
}

type TeamWithRole struct {
	Team Team
	Role TeamRole
}
