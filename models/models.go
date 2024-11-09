package models

import (
	"errors"
	"fmt"

	_ "github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type Config struct {
	Host     string
	Port     string
	Password string
	User     string
	DBName   string
	SSLMode  string
}

type Database struct {
	DB *gorm.DB
}

func (db *Database) GetUser(encode_to_user *User, username string) error {
	res := db.DB.Where("username = ?", username).First(&encode_to_user)
	if res.RowsAffected == 0 {
		return fmt.Errorf("User not found")
	}
	return nil
}

func (db *Database) CreateUser(encode_to_user *User) error {
	if err := db.DB.Create(encode_to_user).Error; err != nil {
		var pgError *pgconn.PgError
		if errors.As(err, &pgError) && pgError.Code == "23505" {
			return fmt.Errorf("Username or Email already exists")
		}
		return fmt.Errorf("Could not create user")
	}
	return nil
}

type User struct {
	gorm.Model
	Username string `json:"username" form:"username" validate:"required,alphanum,min=3,max=20"`
	Email    string `json:"email" form:"email" validate:"required,min=8"`
}

func (u *User) HashUserPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func (u *User) PasswordHashValidation(password, password_hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(password_hash), []byte(password))
	return err == nil
}
