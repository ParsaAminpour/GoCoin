package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	_ "github.com/ParsaAminpour/GoCoin/user_auth/config"
	"github.com/ParsaAminpour/GoCoin/user_auth/helper"
	_ "github.com/ParsaAminpour/GoCoin/user_auth/helper"
	"github.com/ParsaAminpour/GoCoin/user_auth/models"
	"github.com/fatih/color"
	"github.com/golang-jwt/jwt"
	echojwt "github.com/labstack/echo-jwt"
	_ "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	_ "github.com/libp2p/go-libp2p"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	db   *gorm.DB
	once sync.Once
	mu   = sync.Mutex{}
)

func getDB() *gorm.DB {
	conf := models.Config{}
	once.Do(func() {
		conf, err := conf.ExtractDbConfig()
		if err != nil {
			log.Fatalf("Failed to extract DB config: %v", err)
		}
		_, err = init_db_connection(&conf)
		if err != nil {
			log.Fatalf("Failed to initialize DB: %v", err)
		}
	})
	return db
}

func init_db_connection(conf *models.Config) (*gorm.DB, error) {
	fmt.Println("I'm here...")
	if db != nil {
		return nil, fmt.Errorf("Db already initialized")
	}
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		conf.Host, conf.Port, conf.User, conf.Password, conf.DBName, conf.SSLMode,
	)
	var err error
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{}) // Use the global 'db' variable here
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}
	fmt.Println("Connected to DB")

	// AutoMigrate the User model
	if err := db.AutoMigrate(&models.User{}); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %v", err)
	}
	fmt.Println("I'm here too...")

	return db, nil
}

func getAllUsers(c echo.Context) error {
	mu.Lock()
	defer mu.Unlock()
	var users []models.User
	db.Find(&users)

	return c.JSON(http.StatusOK, users)
}

func fetchUser(c echo.Context) error {
	mu.Lock()
	defer mu.Unlock()

	req_username := c.Param("username")
	fmt.Println(req_username)
	database := &models.Database{DB: db}

	var user models.User
	res := database.GetUser(&user, req_username, nil)
	c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

	if errors.Is(res, fmt.Errorf("User not found")) {
		c.Response().WriteHeader(http.StatusNoContent)
		return c.String(http.StatusNoContent, res.Error())
	} else {
		c.Response().WriteHeader(http.StatusOK)
		return json.NewEncoder(c.Response()).Encode(&user)
	}
}

func createUser(c echo.Context) error {
	mu.Lock()
	defer mu.Unlock()
	u := new(models.User)

	if err := c.Bind(u); err != nil {
		return c.String(http.StatusBadRequest, "Invalid parameters provided")
	}

	color.Green("Created: Username: %s, Email: %s\n", u.Username, u.Email)
	encrypted_password, _ := u.HashUserPassword(u.Password)
	u.Password = encrypted_password
	database := &models.Database{DB: db}

	err := database.CreateUser(u)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusCreated, u)
}

func deleteUser(c echo.Context) error {
	mu.Lock()
	defer mu.Unlock()
	id := c.Param("id")

	var user models.User
	if err := db.First(&user, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "User not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Could not retrieve user"})
	}

	if err := db.Delete(&user).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Could not delete user"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "User deleted successfully"})
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

func withHandlerFunc(_handlerFunc func(c echo.Context, db *gorm.DB) error) echo.HandlerFunc {
	return func(c echo.Context) error {
		return _handlerFunc(c, db)
	}
}
func main() {
	my_db := getDB()
	fmt.Println("DB initialized:", my_db)
	fmt.Println("err: ", db.Error)

	e := echo.New()

	e.Group("users")
	e.GET("/users/get/:username", fetchUser)
	e.GET("/users/all", getAllUsers)
	e.POST("/users/create", createUser)
	e.DELETE("/users/:id", deleteUser)

	e.Group("auth")
	e.POST("/auth/signup", withHandlerFunc(helper.Signup))
	e.POST("/auth/login", withHandlerFunc(helper.Login))
	e.POST("/auth/logout", func(c echo.Context) error { return nil })
	e.POST("/auth/resert-password", withHandlerFunc(helper.ResetPassword))

	e.Use(echojwt.JWT([]byte("secret")))
	e.Use(middleware.Recover())
	e.Use(middleware.Logger())
	e.Use(middleware.TimeoutWithConfig(middleware.TimeoutConfig{
		Skipper:      middleware.DefaultSkipper,
		ErrorMessage: "custom timeout error message returns to client",
		OnTimeoutRouteErrorHandler: func(err error, c echo.Context) {
			log.Println(c.Path())
		},
		Timeout: 30 * time.Second,
	}))
	e.Logger.Fatal(e.Start(":8082"))
}
