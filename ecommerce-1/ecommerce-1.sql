-- Customers table
CREATE TABLE Customers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    email TEXT UNIQUE NOT NULL
);

-- Orders table
CREATE TABLE Orders (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    customer_id INTEGER REFERENCES Customers(id),
    shipping_status TEXT CHECK (shipping_status IN ('pending', 'shipped', 'delivered')) NOT NULL,
    -- Calculated fields like total_price will be handled application-side or through triggers
    FOREIGN KEY(customer_id) REFERENCES Customers(id)
);

-- Products table
CREATE TABLE Products (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    price REAL NOT NULL
);

-- Order_Products junction table
CREATE TABLE Order_Products (
    order_id INTEGER REFERENCES Orders(id),
    product_id INTEGER REFERENCES Products(id),
    quantity INTEGER NOT NULL,
    PRIMARY KEY(order_id, product_id)
);

INSERT INTO Customers (name, email) VALUES ('John Doe', 'john.doe@example.com');
INSERT INTO Customers (name, email) VALUES ('Jane Smith', 'jane.smith@example.com');
INSERT INTO Customers (name, email) VALUES ('Alice Johnson', 'alice.johnson@example.com');
INSERT INTO Customers (name, email) VALUES ('Bob Brown', 'bob.brown@example.com');
INSERT INTO Customers (name, email) VALUES ('Charlie Davis', 'charlie.davis@example.com');
INSERT INTO Customers (name, email) VALUES ('David Wilson', 'david.wilson@example.com');
INSERT INTO Customers (name, email) VALUES ('Evelyn Miller', 'evelyn.miller@example.com');
INSERT INTO Customers (name, email) VALUES ('Frank Taylor', 'frank.taylor@example.com');
INSERT INTO Customers (name, email) VALUES ('Grace Anderson', 'grace.anderson@example.com');
INSERT INTO Customers (name, email) VALUES ('Helen Thomas', 'helen.thomas@example.com');

INSERT INTO Products (name, price) VALUES ('Product A', 99.99);
INSERT INTO Products (name, price) VALUES ('Product B', 49.99);
INSERT INTO Products (name, price) VALUES ('Product C', 79.99);
INSERT INTO Products (name, price) VALUES ('Product D', 129.99);
INSERT INTO Products (name, price) VALUES ('Product E', 89.99);
INSERT INTO Products (name, price) VALUES ('Product F', 159.99);
INSERT INTO Products (name, price) VALUES ('Product G', 69.99);
INSERT INTO Products (name, price) VALUES ('Product H', 199.99);
INSERT INTO Products (name, price) VALUES ('Product I', 39.99);
INSERT INTO Products (name, price) VALUES ('Product J', 149.99);


