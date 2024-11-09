package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/ParsaAminpour/GoCoin/models"
	_ "github.com/ParsaAminpour/GoCoin/models"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/joho/godotenv"
	echojwt "github.com/labstack/echo-jwt"
	_ "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	db   *gorm.DB
	once sync.Once
	mu   = sync.Mutex{}
)

func getDB() *gorm.DB {
	once.Do(func() {
		conf, err := extract_db_config()
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

func extract_db_config() (models.Config, error) {
	err := godotenv.Load()
	if err != nil {
		return models.Config{}, fmt.Errorf("error occurred in opening .env")
	}
	db_conf := models.Config{
		Host:     os.Getenv("DB_HOST"),
		Port:     os.Getenv("DB_PORT"),
		Password: os.Getenv("DB_PASSWORD"),
		User:     os.Getenv("DB_USER"),
		DBName:   os.Getenv("DB_NAME"),
		SSLMode:  os.Getenv("DB_SSLMODE"),
	}

	return db_conf, nil
}

func fetchUser(c echo.Context) error {
	mu.Lock()
	defer mu.Unlock()

	req_username := c.Param("username")
	fmt.Println(req_username)
	database := &models.Database{DB: db}

	var user models.User
	res := database.GetUser(&user, req_username)
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

	// Bind JSON data directly into the User struct
	if err := c.Bind(u); err != nil {
		return c.String(http.StatusBadRequest, "Invalid parameters provided")
	}

	fmt.Printf("After Bind - Username: %s, Email: %s\n", u.Username, u.Email)

	if err := db.Create(u).Error; err != nil {
		var pgError *pgconn.PgError
		if errors.As(err, &pgError) && pgError.Code == "23505" {
			return c.JSON(http.StatusConflict, map[string]string{"error": "Username or Email already exists"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Could not create user"})
	}

	return c.JSON(http.StatusCreated, u)
}

func getAllUsers(c echo.Context) error {
	mu.Lock()
	defer mu.Unlock()
	var users []models.User
	db.Find(&users)

	return c.JSON(http.StatusOK, users)
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

func signup(c echo.Context) error {
	mu.Lock()
	defer mu.Unlock()
	return nil
}

func login(c echo.Context) error {
	mu.Lock()
	defer mu.Unlock()
	return nil
}

func main() {
	my_db := getDB()
	fmt.Println("DB initialized:", my_db)

	fmt.Println("DB initialized successfully")
	fmt.Println("err: ", db.Error)

	e := echo.New()

	e.Use(echojwt.JWT([]byte("secret")))
	e.Use(middleware.Recover())
	e.Use(middleware.Logger())

	e.Group("users")
	e.GET("/users/get/:username", fetchUser)
	e.GET("/users/all", getAllUsers)
	e.POST("/users/create", createUser)
	e.DELETE("/users/:id", deleteUser)

	e.Group("auth")
	e.POST("/auth/signup", signup)
	e.POST("/atuh/login", login)

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
