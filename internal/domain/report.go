package domain

type TeamStatistics struct {
	TeamID            int64
	TeamName          string
	MemberCount       int64
	DoneLastSevenDays int64
}

type TopTaskCreator struct {
	TeamID           int64
	TeamName         string
	UserID           int64
	UserName         string
	CreatedTaskCount int64
	Position         int64
}

type InvalidTaskAssignee struct {
	TaskID     int64
	TeamID     int64
	Title      string
	AssigneeID int64
}
