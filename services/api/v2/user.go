package v2service

import (
	u "dohoarding/apiHelpers"
	"dohoarding/models"
	res "dohoarding/resources/api/v2"
)

//UserService struct
type UserService struct {
	User models.User
}

//UserList function returns the list of users
func (us *UserService) UserList() map[string]interface{} {
	user := us.User

	userData := res.UserResponse{
		ID:    user.ID,
		Name:  "test",
		Email: "test@gmail.com",
	}
	response := u.Message(0, "This is from version 2 api")
	response["data"] = userData
	return response
}
