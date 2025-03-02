package userapi

type RegisterRequest struct {
	Username string
	Password string
	Email    string
	Phone    string
}

type RegisterResponse struct {
	Success bool
	UserId  int32
}

type LoginRequest struct {
	Username string
	Password string
}

type LoginResponse struct {
	Success       bool
	UserId        int32
	Token         string
	ErrorMessage  string
}

type DeleteUserRequest struct {
	UserId   int32
	Password string
}

type DeleteUserResponse struct {
	Success       bool
	ErrorMessage  string
}
