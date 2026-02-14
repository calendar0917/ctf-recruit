package challenge

type CreateChallengeRequest struct {
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Category    string     `json:"category"`
	Difficulty  Difficulty `json:"difficulty"`
	Mode        Mode       `json:"mode"`
	Points      int        `json:"points"`
	Flag        string     `json:"flag"`
	IsPublished bool       `json:"isPublished"`
}

type UpdateChallengeRequest struct {
	Title       *string     `json:"title"`
	Description *string     `json:"description"`
	Category    *string     `json:"category"`
	Difficulty  *Difficulty `json:"difficulty"`
	Mode        *Mode       `json:"mode"`
	Points      *int        `json:"points"`
	Flag        *string     `json:"flag"`
	IsPublished *bool       `json:"isPublished"`
}

type ChallengeResponse struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Category    string     `json:"category"`
	Difficulty  Difficulty `json:"difficulty"`
	Mode        Mode       `json:"mode"`
	Points      int        `json:"points"`
	IsPublished bool       `json:"isPublished"`
	CreatedAt   string     `json:"createdAt"`
	UpdatedAt   string     `json:"updatedAt"`
}

type ChallengeListResponse struct {
	Items  []ChallengeResponse `json:"items"`
	Limit  int                 `json:"limit"`
	Offset int                 `json:"offset"`
}
