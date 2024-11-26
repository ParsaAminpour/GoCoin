package server

import (
	"context"
	_ "errors"
	"fmt"
	_ "fmt"
	"log"
	"net"
	_ "net/http"
	"sync"
	_ "sync"
	_ "time"

	"github.com/ParsaAminpour/GoCoin/orderbook/models"
	_ "github.com/ParsaAminpour/GoCoin/orderbook/models"
	"github.com/ParsaAminpour/GoCoin/orderbook/pb"
	_ "github.com/ParsaAminpour/GoCoin/orderbook/pb"
	_ "github.com/fatih/color"
	_ "github.com/golang-jwt/jwt"
	_ "github.com/labstack/echo/v4"
	"google.golang.org/grpc"
	_ "google.golang.org/grpc"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	_ "gorm.io/gorm"
)

var (
	mu   = sync.Mutex{}
	once sync.Once
	db   *gorm.DB
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

	// AutoMigrate the Order model
	if err := db.AutoMigrate(&models.Order{}, &models.Orderbook{}); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %v", err)
	}
	fmt.Println("I'm here too...")

	return db, nil
}

type server struct {
	pb.OrderInfoServiceServer
}

func (s *server) GetOrderInfo(ctx context.Context, req *pb.OrderInfoRequest) (*pb.OrderInfoReply, error) {
	// var order models.Order
	return &pb.OrderInfoReply{
		Order: &pb.Order{},
	}, nil
}

func main() {
	my_db := getDB()
	fmt.Println("DB initialized:", my_db)

	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		panic(err)
	}

	s := grpc.NewServer()
	pb.RegisterOrderInfoServiceServer(s, &server{})
	if err := s.Serve(listener); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
