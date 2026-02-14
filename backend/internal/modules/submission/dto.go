package submission

type CreateSubmissionRequest struct {
	ChallengeID      string `json:"challengeId"`
	ChallengeIDSnake string `json:"challenge_id"`
	Flag             string `json:"flag"`
}

type SubmissionResponse struct {
	ID            string  `json:"id"`
	ChallengeID   string  `json:"challengeId"`
	Status        Status  `json:"status"`
	AwardedPoints int     `json:"awardedPoints"`
	JudgeJobID    *string `json:"judgeJobId,omitempty"`
	CreatedAt     string  `json:"createdAt"`
}
