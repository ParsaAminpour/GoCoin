package tests

// Testing database functionality in memory as tmp_db to aviod main database manipulation
import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ParsaAminpour/GoCoin/helper"
	"github.com/ParsaAminpour/GoCoin/models"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	_ "github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func CreateTestMemDatabase() (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	err = db.AutoMigrate(&models.User{})
	if err != nil {
		return nil, err
	}
	return db, nil

}
func CloseTestMemDatabase(db *gorm.DB) error {
	sqkDB, _ := db.DB()
	err := sqkDB.Close()
	if err != nil {
		return err
	}
	return nil
}

func SendRequest(req_body map[string]interface{}, method string, path string) *http.Request {
	json_req, _ := json.Marshal(req_body)

	var req *http.Request
	if method == http.MethodPost && len(req_body) != 0 {
		req = httptest.NewRequest(method, path, bytes.NewReader(json_req))
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set(echo.HeaderAuthorization, "Bearer eyJhbGciOiJIUzI1NiJ9.e30.XmNK3GpH3Ys_7wsYBfq4C3M6goz71I7dTgUkuIa5lyQ")
	return req
}

func TestSignupEndpoit(t *testing.T) {
	e := echo.New()
	req_body := map[string]interface{}{
		"username": "parsa",
		"email":    "parsa.aminpour@gmail.com",
		"password": "parsatestpassword",
	}

	req := SendRequest(req_body, http.MethodPost, "/auth/signup")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	db, err := CreateTestMemDatabase()
	assert.NoError(t, err)
	defer CloseTestMemDatabase(db)

	err = helper.Signup(c, db)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)

	var response models.User
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	fmt.Println(req_body["username"])
	assert.Equal(t, response.Username, req_body["username"])
	assert.Equal(t, response.Email, req_body["email"])
	assert.True(t, response.PasswordHashValidation(req_body["password"].(string), response.Password))
}

func TestLoginEndpointWithValidBody(t *testing.T) {
	e := echo.New()
	req_body := map[string]interface{}{
		"username": "parsa",
		"email":    "parsa.aminpour@gmail.com",
		"password": "parsatestpassword",
	}
	db, err := CreateTestMemDatabase()
	if err != nil {
		_ = fmt.Errorf("Someting worong in openning mem database")
	}
	defer CloseTestMemDatabase(db)

	signup_req := SendRequest(req_body, http.MethodPost, "/auth/signup")
	signup_rec := httptest.NewRecorder()
	signup_c := e.NewContext(signup_req, signup_rec)
	helper.Signup(signup_c, db)

	// Login Request
	login_req_body := map[string]interface{}{
		"username": "parsa",
		"email":    "parsa.aminpour@gmail.com",
		"password": "parsatestpassword",
	}
	login_req := SendRequest(login_req_body, http.MethodPost, "/auth/login")
	login_rec := httptest.NewRecorder()
	login_c := e.NewContext(login_req, login_rec)
	helper.Login(login_c, db)

	var response map[string]string
	err = json.Unmarshal(login_rec.Body.Bytes(), &response)
	assert.NoError(t, err)

	token_parts := strings.Split(response["token"], ".")
	assert.Equal(t, response["message"], "Login Successful")
	assert.Equal(t, token_parts[0], "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9")
}

func TestResetPasswordEndpoint(t *testing.T) {
	e := echo.New()
	db, err := CreateTestMemDatabase()
	assert.NoError(t, err)
	defer CloseTestMemDatabase(db)
	database := &models.Database{DB: db}

	var mock_users []models.User
	mock_users, _ = GenerateMockUsers(10)
	err = createBatchUser(mock_users, db)
	assert.NoError(t, err)

	req_body := map[string]interface{}{
		"username":     mock_users[0].Username,
		"old-password": mock_users[0].Password,
		"new-password": "newtestpasswordABC1",
	}

	req := SendRequest(req_body, http.MethodPost, "/auth/resert-password")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	helper.ResetPassword(c, db)

	var response map[string]string
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, response["message"], "Successful")

	var updated_user models.User
	database.DB.Where("username = ?", mock_users[0].Username).Find(&updated_user)
	assert.NotEqual(t, updated_user.Password, req_body["old-password"])
}
