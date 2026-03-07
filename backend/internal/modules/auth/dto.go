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
	IsDisabled  bool   `json:"isDisabled"`
}

type LoginResponse struct {
	AccessToken string       `json:"accessToken"`
	TokenType   string       `json:"tokenType"`
	User        UserResponse `json:"user"`
}

type AdminListUsersResponse struct {
	Items []UserResponse `json:"items"`
}

type AdminUpdateUserRequest struct {
	Role       *Role `json:"role,omitempty"`
	IsDisabled *bool `json:"isDisabled,omitempty"`
}
