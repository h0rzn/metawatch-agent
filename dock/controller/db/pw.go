package db

import (
	"golang.org/x/crypto/bcrypt"
)

// https://astaxie.gitbooks.io/build-web-application-with-golang/content/en/09.5.html

func hash(input string) ([]byte, error) {
	return bcrypt.GenerateFromPassword([]byte(input), bcrypt.DefaultCost)

}

func compare(input string, hash []byte) bool {
	if err := bcrypt.CompareHashAndPassword(hash, []byte(input)); err != nil {
		return false
	}
	return true
}
