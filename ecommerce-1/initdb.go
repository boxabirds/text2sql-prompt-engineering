package main

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

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

// Function to create a new order
// Function to create a new order
func createNewOrder(db *sql.DB, customerName string, productName string, qty int) error {
	// Get customer ID by name
	customerId, err := getCustomerIdByName(db, customerName)
	if err != nil {
		return fmt.Errorf("failed to get customer ID: %v", err)
	}

	// Get product ID by name
	productId, err := getProductIdByName(db, productName)
	if err != nil {
		return fmt.Errorf("failed to get product ID: %v", err)
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %v", err)
	}

	// Insert into Orders table
	result, err := tx.Exec("INSERT INTO Orders (customer_id, qty, shipping_status) VALUES (?,?,?)", customerId, qty, "pending")
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to insert order: %v", err)
	}

	// Get the last inserted order ID
	lastOrderId, err := result.LastInsertId()
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to get last insert ID: %v", err)
	}

	// Insert into Order_Products table
	_, err = tx.Exec("INSERT INTO Order_Products (order_id, product_id, quantity) VALUES (?,?,?)", lastOrderId, productId, qty)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to insert into Order_Products: %v", err)
	}

	// Commit the transaction
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	return nil
}

func initialiseDb(dbName string) {

	// Open the database
	db, err := sql.Open("sqlite3", dbName)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer db.Close()

	// Ensure the database is properly closed when the program exits
	defer func() {
		err := db.Close()
		if err != nil {
			fmt.Println(err)
		}
	}()

	// Create tables
	createCustomersTable := `
	CREATE TABLE IF NOT EXISTS Customers (
	    id INTEGER PRIMARY KEY AUTOINCREMENT,
	    name TEXT NOT NULL,
	    email TEXT UNIQUE NOT NULL
	);`
	createOrdersTable := `
	CREATE TABLE IF NOT EXISTS Orders (
	    id INTEGER PRIMARY KEY AUTOINCREMENT,
	    customer_id INTEGER REFERENCES Customers(id),
		qty INTEGER NOT NULL,
	    shipping_status TEXT CHECK (shipping_status IN ('pending', 'shipped', 'delivered')) NOT NULL,
	    FOREIGN KEY(customer_id) REFERENCES Customers(id)
	);`
	createProductsTable := `
	CREATE TABLE IF NOT EXISTS Products (
	    id INTEGER PRIMARY KEY AUTOINCREMENT,
	    name TEXT NOT NULL,
	    price REAL NOT NULL
	);`
	createOrderProductsTable := `
	CREATE TABLE IF NOT EXISTS Order_Products (
	    order_id INTEGER REFERENCES Orders(id),
	    product_id INTEGER REFERENCES Products(id),
	    quantity INTEGER NOT NULL,
	    PRIMARY KEY(order_id, product_id)
	);`

	tables := []string{createCustomersTable, createOrdersTable, createProductsTable, createOrderProductsTable}

	fmt.Println("Creating tables...")
	for _, query := range tables {
		_, err := db.Exec(query)
		if err != nil {
			fmt.Println(err)
		}
	}

	// Insert sample data
	fmt.Println("Inserting sample data...")
	insertSampleData(db)
}

func insertSampleData(db *sql.DB) {
	// Insert customers
	customers := []struct {
		name  string
		email string
	}{
		{"Customer 1", "customer1@example.com"},
		{"Customer 2", "customer2@example.com"},
		// Add more customers as needed
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
		// Add more products as needed
	}

	for _, p := range products {
		_, err := db.Exec("INSERT INTO Products (name, price) VALUES (?,?)", p.name, p.price)
		if err != nil {
			fmt.Println(err)
		}
	}

	// Insert orders
	orders := []struct {
		customerName string
		productName  string
		qty          int
	}{
		{"Customer 1", "Product 1", 1},
		{"Customer 1", "Product 2", 2},
		{"Customer 2", "Product 1", 3},
		{"Customer 2", "Product 2", 4},
		// Add more orders as needed
	}
	for _, o := range orders {
		err := createNewOrder(db, o.customerName, o.productName, o.qty)
		if err != nil {
			fmt.Println(err)
		}
	}
}
