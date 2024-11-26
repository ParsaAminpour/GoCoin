package models

import (
	_ "container/heap"
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

type OrderSide int

const (
	Buy OrderSide = iota
	Sell
)

func (order_side OrderSide) getString() string {
	switch order_side {
	case Buy:
		return "Buy"
	case Sell:
		return "Sell"
	default:
		return "Unknown"
	}
}

type Order struct {
	gorm.Model
	Price         uint      `json:"price" form:"price" validate:"required" gorm:"type:decimal(10,2)"`
	Quantity      uint      `json:"quantity" form:"quantity" validate:"required" gorm:"type:decimal(10,2)"`
	Side          OrderSide `json:"side" form:"side" validate:"required"`
	Timestamp     uint32    `json:"timestamp" form:"timestamp" validate:"required"`
	OwnerUsername string    `json:"owner_username" form:"owner_username" validate:"required"`
}

type OrderHeap []*Order

func (h OrderHeap) Len() int { return len(h) }

func (h OrderHeap) Less(i, j int) bool {
	if h[i].Side.getString() == "Buy" {
		return h[i].Price > h[j].Price // Max-heap for buy orders
	}
	return h[i].Price < h[j].Price // Min-heap for sell orders
}

func (h OrderHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

// NOTE: Push an Order: Add an order to the heap.
func (h *OrderHeap) Push(x interface{}) {
	*h = append(*h, x.(*Order)) // Add new order to the heap
}

// NOTE: Pop the Top Order: Remove the highest-priority order from the heap.
func (h *OrderHeap) Pop() interface{} {
	old := *h
	n := len(old)
	item := old[n-1]  // Get last item
	*h = old[0 : n-1] // Reduce the heap size
	return item       // Return the removed item
}

// NOTE: matching orders using heap O(1)
// NOTE: manipulating orders is in O(logn)
type Orderbook struct {
	gorm.Model
	bidOrders *OrderHeap
	askOrders *OrderHeap
}

/*
OrderBook
 ├── BidOrders (max heap or map) → {Price → {Order1, Order2, ...}}  # Sorted by highest price first
 └── AskOrders (min heap or map) → {Price → {Order1, Order2, ...}}  # Sorted by lowest price first

Order
 ├── ID: Unique identifier
 ├── Price: Price of the order
 ├── Quantity: Amount of the asset
 ├── Side: "buy" or "sell"
 ├── Timestamp: Time order was placed
 └── OwnerID: ID of the user who placed the order
*/

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
		Host:     os.Getenv("DB_HOST_ORDERBOOK"),
		Port:     os.Getenv("DB_PORT_ORDERBOOK"),
		Password: os.Getenv("DB_PASSWORD_ORDERBOOK"),
		User:     os.Getenv("DB_USER_ORDERBOOK"),
		DBName:   os.Getenv("DB_NAME_ORDERBOOK"),
		SSLMode:  os.Getenv("DB_SSLMODE_ORDERBOOK"),
	}

	return db_conf, nil
}

type Database struct {
	DB *gorm.DB
}

func (db *Database) getOrderById(encoded_to_order *Order, id int) error {
	if err := db.DB.Where("ID == ?", id).First(&encoded_to_order).Error; err != nil {
		return fmt.Errorf("error: %s", err.Error())
	}
	return nil
}
