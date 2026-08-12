package dto

type RegisterRequest struct {
	FullName string `json:"full_name"`
	Phone    string `json:"phone"`
	Password string `json:"password"`
	Location string `json:"location"`
}
