package tests

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/ParsaAminpour/GoCoin/user_auth/models"
	"github.com/fatih/color"
	"gorm.io/gorm"
)

func GenerateMockUsers(amount int) ([]models.User, error) {
	file, err := os.Open("./MOCK_DATA.json")
	if err != nil {
		return nil, fmt.Errorf("Error in openning file")
	}
	defer file.Close()

	var users []models.User
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&users); err != nil {
		return nil, fmt.Errorf(err.Error())
	}
	for _, user := range users {
		color.Green("username: %s, email: %s, pass: %s\n", user.Username, user.Email, user.Password)
	}
	return users[:amount], nil
}

func createBatchUser(users []models.User, db *gorm.DB) error {
	database := models.Database{DB: db}
	for i, user := range users {
		user.Password, _ = user.HashUserPassword(user.Password)
		if err := database.CreateUser(&user); err != nil {
			return fmt.Errorf("error in creating user No.%d", i)
		}
	}
	return nil
}
