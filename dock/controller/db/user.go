package db

import (
	"errors"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type BasicUser struct {
	ID   primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Name string             `json:"name" binding:"required" bson:"name"`
}

type User struct {
	BasicUser
	Password string `json:"password" binding:"required" bson:"password"`
}

func (u *User) HashPassword(pw string) (string, error) {
	hashed, err := hash(pw)
	if err != nil {
		return "", errors.New("failed to hash password")
	}
	u.Password = string(hashed)
	return "", nil
}
