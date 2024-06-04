| Query      | SQL | Result |
| ---| --- | --- |
| How many customers are there? | SELECT COUNT(*) AS num_customers FROM Customers; | 3 |
| How many orders have been shipped? | SELECT COUNT(*) AS num_shipped_orders FROM Orders WHERE shipping_status = 'shipped';| 2 |
| What is the total value of orders we have? | SELECT SUM(op.quantity * p.price) AS total_order_value FROM Order_Products op JOIN Products p ON op.product_id = p.id; | x |
| How many copies of "Product D" have been sold? | SELECT SUM(quantity) AS num_product_d_sold FROM Order_Products WHERE product_id = (SELECT id FROM Products WHERE name = 'Product D'); | x |