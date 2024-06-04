BEGIN TRANSACTION;
CREATE TABLE IF NOT EXISTS "Customers" (
	"id"	INTEGER,
	"name"	TEXT NOT NULL,
	"email"	TEXT NOT NULL UNIQUE,
	PRIMARY KEY("id" AUTOINCREMENT)
);
CREATE TABLE IF NOT EXISTS "Orders" (
	"id"	INTEGER,
	"customer_id"	INTEGER,
	"shipping_status"	TEXT NOT NULL CHECK("shipping_status" IN ('pending', 'shipped', 'delivered')),
	PRIMARY KEY("id" AUTOINCREMENT),
	FOREIGN KEY("customer_id") REFERENCES "Customers"("id"),
);
CREATE TABLE IF NOT EXISTS "Products" (
	"id"	INTEGER,
	"name"	TEXT NOT NULL,
	"price"	REAL NOT NULL,
	PRIMARY KEY("id" AUTOINCREMENT)
);
CREATE TABLE IF NOT EXISTS "Order_Products" (
	"order_id"	INTEGER,
	"product_id"	INTEGER,
	"quantity"	INTEGER NOT NULL,
	PRIMARY KEY("order_id","product_id"),
	FOREIGN KEY("order_id") REFERENCES "Orders"("id"),
	FOREIGN KEY("product_id") REFERENCES "Products"("id")
);
CREATE INDEX IF NOT EXISTS "idx_customers_name" ON "Customers" (
	"name"
);
CREATE INDEX IF NOT EXISTS "idx_products_name" ON "Products" (
	"name"
);
COMMIT;
