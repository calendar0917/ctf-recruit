package instance

type StartInstanceRequest struct {
	ChallengeID string `json:"challengeId"`
}

type InstanceResponse struct {
	ID            string      `json:"id"`
	UserID        string      `json:"userId"`
	ChallengeID   string      `json:"challengeId"`
	Status        Status      `json:"status"`
	ContainerID   *string     `json:"containerId,omitempty"`
	AccessInfo    *AccessInfo `json:"accessInfo,omitempty"`
	StartedAt     *string     `json:"startedAt,omitempty"`
	ExpiresAt     *string     `json:"expiresAt,omitempty"`
	CooldownUntil *string     `json:"cooldownUntil,omitempty"`
}

type AccessInfo struct {
	Host             string `json:"host"`
	Port             int    `json:"port"`
	ConnectionString string `json:"connectionString,omitempty"`
}

type TransitionRequest struct {
	Status      Status  `json:"status"`
	ContainerID *string `json:"containerId,omitempty"`
}

type StopInstanceRequest struct {
	InstanceID *string `json:"instanceId,omitempty"`
}

type InstanceCooldownResponse struct {
	RetryAt string `json:"retryAt"`
}

type MyInstanceResponse struct {
	Instance *InstanceResponse         `json:"instance"`
	Cooldown *InstanceCooldownResponse `json:"cooldown,omitempty"`
}
