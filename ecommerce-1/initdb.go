package main

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

// Create tables
const CREATE_CUSTOMERS_TABLE = `
	CREATE TABLE IF NOT EXISTS Customers (
	    id INTEGER PRIMARY KEY AUTOINCREMENT,
	    name TEXT NOT NULL,
	    email TEXT UNIQUE NOT NULL
	);
	CREATE INDEX idx_customers_name ON Customers(name);`
const CREATE_ORDERS_TABLE = `
	CREATE TABLE IF NOT EXISTS Orders (
	    id INTEGER PRIMARY KEY AUTOINCREMENT,
	    customer_id INTEGER REFERENCES Customers(id),
	    shipping_status TEXT CHECK (shipping_status IN ('pending', 'shipped', 'delivered')) NOT NULL,
	    FOREIGN KEY(customer_id) REFERENCES Customers(id)
	);`
const CREATE_PRODUCTS_TABLE = `
	CREATE TABLE IF NOT EXISTS Products (
	    id INTEGER PRIMARY KEY AUTOINCREMENT,
	    name TEXT NOT NULL,
	    price REAL NOT NULL
	);
	CREATE INDEX idx_products_name ON Products(name);`

const CREATE_ORDER_PRODUCTS_TABLE = `
	CREATE TABLE IF NOT EXISTS Order_Products (
	    order_id INTEGER REFERENCES Orders(id),
	    product_id INTEGER REFERENCES Products(id),
	    quantity INTEGER NOT NULL,
	    PRIMARY KEY(order_id, product_id)
	);`

var TABLES = []string{CREATE_CUSTOMERS_TABLE, CREATE_ORDERS_TABLE, CREATE_PRODUCTS_TABLE, CREATE_ORDER_PRODUCTS_TABLE}

type ProductQuantity struct {
	Name     string
	Quantity int
}

func getCustomerIdByName(db *sql.DB, customerName string) (int, error) {
	var customerId int
	err := db.QueryRow("SELECT id FROM Customers WHERE name=?", customerName).Scan(&customerId)
	if err != nil && err != sql.ErrNoRows {
		return 0, fmt.Errorf("failed to get customer ID: %v", err)
	}
	return customerId, nil
}

// Function to get product ID by name
func getProductIdByName(db *sql.DB, productName string) (int, error) {
	var productId int
	err := db.QueryRow("SELECT id FROM Products WHERE name=?", productName).Scan(&productId)
	if err != nil && err != sql.ErrNoRows {
		return 0, fmt.Errorf("failed to get product ID: %v", err)
	}
	return productId, nil
}

func createNewOrder(db *sql.DB, customerName string, products []ProductQuantity, shippingStatus string) error {
	customerId, err := getCustomerIdByName(db, customerName)
	if err != nil {
		return fmt.Errorf("failed to get customer ID: %v", err)
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %v", err)
	}

	// Insert into Orders table
	result, err := tx.Exec("INSERT INTO Orders (customer_id, shipping_status) VALUES (?,?)", customerId, shippingStatus)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to insert order: %v", err)
	}

	lastOrderId, err := result.LastInsertId()
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to get last insert ID: %v", err)
	}

	// Iterate over products and insert into Order_Products table
	for _, pq := range products {
		productId, err := getProductIdByName(db, pq.Name)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to get product ID: %v", err)
		}

		_, err = tx.Exec("INSERT INTO Order_Products (order_id, product_id, quantity) VALUES (?,?,?)", lastOrderId, productId, pq.Quantity)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to insert into Order_Products: %v", err)
		}
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	return nil
}

func initialiseDb(dbName string) (*sql.DB, error) {

	// Check if the database file exists and simply return it if it does
	if _, err := os.Stat(dbName); !os.IsNotExist(err) {
		// Database exists, open it
		db, err := sql.Open("sqlite3", dbName)
		if err != nil {
			return nil, fmt.Errorf("error opening existing database: %v", err)
		}
		fmt.Println("Opened existing database:", dbName)
		return db, nil
	}

	// Open the database
	db, err := sql.Open("sqlite3", dbName)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	defer db.Close()

	// Ensure the database is properly closed when the program exits
	defer func() {
		err := db.Close()
		if err != nil {
			fmt.Println(err)
		}
	}()

	fmt.Println("Creating tables...")
	for _, query := range TABLES {
		_, err := db.Exec(query)
		if err != nil {
			fmt.Println(err)
		}
	}

	// Insert sample data
	fmt.Println("Inserting sample data...")
	insertSampleData(db)
	return db, nil
}

func insertSampleData(db *sql.DB) {
	// Insert customers
	customers := []struct {
		name  string
		email string
	}{
		{"Customer 1", "customer1@example.com"},
		{"Customer 2", "customer2@example.com"},
		{"Customer 3", "customer3@example.com"},
		{"Customer 4", "customer4@example.com"},
		{"Customer 5", "customer5@example.com"},
		{"Customer 6", "customer6@example.com"},
		{"Customer 7", "customer7@example.com"},
		{"Customer 8", "customer8@example.com"},
		{"Customer 9", "customer9@example.com"},
		{"Customer 10", "customer10@example.com"},
	}

	for _, c := range customers {
		_, err := db.Exec("INSERT INTO Customers (name, email) VALUES (?,?)", c.name, c.email)
		if err != nil {
			fmt.Println(err)
		}
	}

	// Insert products
	products := []struct {
		name  string
		price float64
	}{
		{"Product 1", 100.00},
		{"Product 2", 200.00},
		{"Product 3", 300.00},
		{"Product 4", 400.00},
		{"Product 5", 500.00},
		{"Product 6", 600.00},
		{"Product 7", 700.00},
		{"Product 8", 800.00},
		{"Product 9", 900.00},
		{"Product 10", 10000.00},
	}

	for _, p := range products {
		_, err := db.Exec("INSERT INTO Products (name, price) VALUES (?,?)", p.name, p.price)
		if err != nil {
			fmt.Println(err)
		}
	}

	// Insert orders
	orders := []struct {
		customerName   string
		products       []ProductQuantity
		shippingStatus string
	}{
		{
			customerName: "Customer 1",
			products: []ProductQuantity{
				{Name: "Product 1", Quantity: 1},
			},
			shippingStatus: "pending",
		},
		{
			customerName: "Customer 2",
			products: []ProductQuantity{
				{Name: "Product 1", Quantity: 1},
				{Name: "Product 2", Quantity: 2},
			},
			shippingStatus: "shipped",
		},
		{
			customerName: "Customer 3",
			products: []ProductQuantity{
				{Name: "Product 1", Quantity: 1},
				{Name: "Product 2", Quantity: 2},
				{Name: "Product 3", Quantity: 3},
			},
			shippingStatus: "delivered",
		},
		{
			customerName: "Customer 4",
			products: []ProductQuantity{
				{Name: "Product 1", Quantity: 1},
				{Name: "Product 2", Quantity: 2},
				{Name: "Product 3", Quantity: 3},
				{Name: "Product 4", Quantity: 4},
			},
			shippingStatus: "delivered",
		},
		{
			customerName: "Customer 5",
			products: []ProductQuantity{
				{Name: "Product 1", Quantity: 1},
				{Name: "Product 2", Quantity: 2},
				{Name: "Product 3", Quantity: 3},
				{Name: "Product 4", Quantity: 4},
				{Name: "Product 5", Quantity: 5},
			},
			shippingStatus: "delivered",
		},
		{
			customerName: "Customer 6",
			products: []ProductQuantity{
				{Name: "Product 1", Quantity: 1},
				{Name: "Product 2", Quantity: 2},
				{Name: "Product 3", Quantity: 3},
				{Name: "Product 4", Quantity: 4},
				{Name: "Product 5", Quantity: 5},
				{Name: "Product 6", Quantity: 6},
			},
			shippingStatus: "delivered",
		},
		{
			customerName: "Customer 7",
			products: []ProductQuantity{
				{Name: "Product 1", Quantity: 1},
				{Name: "Product 2", Quantity: 2},
				{Name: "Product 3", Quantity: 3},
				{Name: "Product 4", Quantity: 4},
				{Name: "Product 5", Quantity: 5},
				{Name: "Product 6", Quantity: 6},
				{Name: "Product 7", Quantity: 7},
			},
			shippingStatus: "delivered",
		},
		{
			customerName: "Customer 8",
			products: []ProductQuantity{
				{Name: "Product 1", Quantity: 1},
				{Name: "Product 2", Quantity: 2},
				{Name: "Product 3", Quantity: 3},
				{Name: "Product 4", Quantity: 4},
				{Name: "Product 5", Quantity: 5},
				{Name: "Product 6", Quantity: 6},
				{Name: "Product 7", Quantity: 7},
				{Name: "Product 8", Quantity: 8},
			},
			shippingStatus: "delivered",
		},
		{
			customerName: "Customer 9",
			products: []ProductQuantity{
				{Name: "Product 1", Quantity: 1},
				{Name: "Product 2", Quantity: 2},
				{Name: "Product 3", Quantity: 3},
				{Name: "Product 4", Quantity: 4},
				{Name: "Product 5", Quantity: 5},
				{Name: "Product 6", Quantity: 6},
				{Name: "Product 7", Quantity: 7},
				{Name: "Product 8", Quantity: 8},
				{Name: "Product 9", Quantity: 9},
			},
			shippingStatus: "delivered",
		},
	}

	for _, o := range orders {
		err := createNewOrder(db, o.customerName, o.products, o.shippingStatus)
		if err != nil {
			fmt.Println(err)
		}
	}
}
