package models

import (
	"errors"
	"fmt"
	"os"

	_ "github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/joho/godotenv"
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

func (conf *Config) ExtractDbConfig() (Config, error) {
	err := godotenv.Load()
	if err != nil {
		return Config{}, fmt.Errorf("error occurred in opening .env")
	}
	db_conf := Config{
		Host:     os.Getenv("DB_HOST"),
		Port:     os.Getenv("DB_PORT"),
		Password: os.Getenv("DB_PASSWORD"),
		User:     os.Getenv("DB_USER"),
		DBName:   os.Getenv("DB_NAME"),
		SSLMode:  os.Getenv("DB_SSLMODE"),
	}

	return db_conf, nil
}

type Database struct {
	DB *gorm.DB
}

func (db *Database) GetUser(encode_to_user *User, username, email interface{}) error {
	var res *gorm.DB
	if email == nil {
		res = db.DB.Where("username = ?", username).First(&encode_to_user)
	} else if username == nil {
		res = db.DB.Where("email = ?", email).First(&encode_to_user)
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("User not found")
	}
	return nil
}

// NOTE: Assume that encode_to_user.Password has been encoded.
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

func (db *Database) DeleteUser(user_ref *User, id int) error {
	if err := db.DB.First(user_ref, "id = ?", id).Error; err != nil {
		return err
	}
	if err := db.DB.Delete(user_ref).Error; err != nil {
		return err
	}
	return nil
}

type User struct {
	gorm.Model
	Username string `json:"username" form:"username" validate:"required,alphanum,min=3,max=20"`
	Email    string `json:"email" form:"email" validate:"required,min=8"`
	Password string `json:"password" form:"password" validate:"required,min=8"`
}

// NOTE: these hashing password use bcrypt which handle the constant time compare behind the scene to avoid side-channel Timing Attack.
func (u *User) HashUserPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// NOTE: these hashing password use bcrypt which handle the constant time compare behind the scene to avoid side-channel Timing Attack.
func (u *User) PasswordHashValidation(password, password_hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(password_hash), []byte(password))
	return err == nil
}
