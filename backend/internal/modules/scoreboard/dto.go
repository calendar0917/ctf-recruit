package scoreboard

type ScoreboardItem struct {
	Rank        int    `json:"rank"`
	UserID      string `json:"userId"`
	DisplayName string `json:"displayName"`
	TotalPoints int    `json:"totalPoints"`
	SolvedCount int    `json:"solvedCount"`
}

type ScoreboardResponse struct {
	Items  []ScoreboardItem `json:"items"`
	Limit  int              `json:"limit"`
	Offset int              `json:"offset"`
}
