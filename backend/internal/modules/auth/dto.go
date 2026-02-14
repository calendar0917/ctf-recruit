package auth

type RegisterRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"displayName"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type UserResponse struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"displayName"`
	Role        Role   `json:"role"`
}

type LoginResponse struct {
	AccessToken string       `json:"accessToken"`
	TokenType   string       `json:"tokenType"`
	User        UserResponse `json:"user"`
}
