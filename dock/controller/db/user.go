package db

import (
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	ID       primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Name     string             `json:"name" binding:"required" bson:"name"`
	Created  time.Time          `json:"created" bson:"created"`
	Password string             `json:"password,omitempty" binding:"required" bson:"password"`
}

func (u *User) HashPassword() error {
	hashed, err := hash(u.Password)
	if err != nil {
		return errors.New("failed to hash password")
	}
	u.Password = string(hashed)
	return nil
}

func (u *User) SetCreated() {
	u.Created = time.Now()
}

func (u *User) RemovePassword() {
	u.Password = ""
}
