package helper

import (
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/ParsaAminpour/GoCoin/models"
	"github.com/fatih/color"
	"github.com/golang-jwt/jwt"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

var (
	mu = sync.Mutex{}
)

func Signup(c echo.Context, db *gorm.DB) error {
	// mu.Lock()
	// defer mu.Unlock()
	user := new(models.User)
	if err := c.Bind(user); err != nil {
		return c.String(http.StatusBadRequest, "Invalid parameters provided")
	}
	// TODO: validating data here using echo .Validate

	database := &models.Database{DB: db}
	user.Password, _ = user.HashUserPassword(user.Password)

	color.Green("Created: Username: %s, Email: %s\n", user.Username, user.Email)
	err := database.CreateUser(user)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusCreated, user)
}

func Login(c echo.Context, db *gorm.DB) error {
	// mu.Lock()
	// defer mu.Unlock()
	user := new(models.User)
	database := &models.Database{DB: db}
	if err := c.Bind(&user); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	//TODO: Adding param validation here.

	var fetched_user models.User
	if err := database.DB.Where("username = ? AND email = ?", user.Username, user.Email).First(&fetched_user).Error; err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}

	fmt.Printf("user.Password: %s | fetced_user: %s, %s, %s\n", user.Password, fetched_user.Username, fetched_user.Email, fetched_user.Password)
	password_auth := user.PasswordHashValidation(user.Password, fetched_user.Password)
	if !password_auth {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Password is wrong!"})
	}

	jwt_token, jwt_err := _generateJWT(fetched_user.Username, uint(time.Now().Add(24*time.Hour).Unix()))
	if jwt_err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": jwt_err.Error()})
	}
	return c.JSON(http.StatusOK, echo.Map{
		"message": "Login Successful",
		"token":   jwt_token,
	})
}

type ResetPasswordReqStructure struct {
	Username    string `json:"username"`
	OldPassword string `json:"old-password"`
	NewPassword string `json:"new-password"`
}

// TODO: Add concurrency to this endpoint handler.
func ResetPassword(c echo.Context, db *gorm.DB) error {
	// mu.Lock()
	// defer mu.Unlock()
	user := models.User{}
	database := models.Database{DB: db}

	bind_format := ResetPasswordReqStructure{}
	if err := c.Bind(&bind_format); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid credentials"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Invalid credentials"})
	}

	if err := database.DB.Where("username = ?", bind_format.Username).First(&user).Error; err != nil {
		return c.JSON(http.StatusConflict, map[string]string{"error": "Invalid credentials"})
	}

	if verified := user.PasswordHashValidation(bind_format.OldPassword, user.Password); !verified {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid credentials"})
	}

	user.Password, _ = user.HashUserPassword(bind_format.NewPassword)
	if err := db.Save(&user).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to update password!"})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Successful",
	})
}

func _generateJWT(username string, exp_time uint) (string, error) {
	claims := jwt.MapClaims{
		"username": username,
		"exp":      exp_time,
	}
	jwtSecret := []byte("SECRET")
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}
