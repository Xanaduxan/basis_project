package dto

type CreateTeamRequest struct {
	Name string `json:"name"`
}

type TeamResponse struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	CreatedBy int64  `json:"created_by"`
	Role      string `json:"role"`
}

type InviteTeamMemberRequest struct {
	Email string `json:"email"`
}

type InviteTeamMemberResponse struct {
	TeamID int64  `json:"team_id"`
	UserID int64  `json:"user_id"`
	Role   string `json:"role"`
}
