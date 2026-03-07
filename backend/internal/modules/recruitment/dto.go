package recruitment

type CreateSubmissionRequest struct {
	Name      string `json:"name"`
	School    string `json:"school"`
	Grade     string `json:"grade"`
	Direction string `json:"direction"`
	Contact   string `json:"contact"`
	Bio       string `json:"bio"`
}

type SubmissionResponse struct {
	ID        string `json:"id"`
	UserID    string `json:"userId"`
	Name      string `json:"name"`
	School    string `json:"school"`
	Grade     string `json:"grade"`
	Direction string `json:"direction"`
	Contact   string `json:"contact"`
	Bio       string `json:"bio"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

type SubmissionListResponse struct {
	Items  []SubmissionResponse `json:"items"`
	Limit  int                  `json:"limit"`
	Offset int                  `json:"offset"`
}
