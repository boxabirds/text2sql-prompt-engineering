| Query      | SQL | Result |
| ---| --- | --- |
| How many customers are there? | SELECT COUNT(*) FROM "Customers"; | [{"COUNT(*)":10}] |
| How many customers have no orders? | SELECT COUNT(*) FROM "Customers" WHERE "id" NOT IN (SELECT "customer_id" FROM "Orders"); | [{"COUNT(*)":1}] |
| What's the most expensive product? | SELECT * FROM "Products" ORDER BY "price" DESC LIMIT 1; | [{"name":"Product 10","price":10000}] |
| What's the most profitable product? | SELECT p."name", SUM(op."quantity" * p."price") AS "profit" FROM "Order_Products" op JOIN "Products" p ON op."product_id" = p."id" GROUP BY p."name" ORDER BY "profit" DESC LIMIT 1; | [{"profit":8100,"name":"Product 7"}] |
| Who is the most profitable customer? | SELECT c."name", SUM(op."quantity" * p."price") AS "profit" FROM "Order_Products" op JOIN "Orders" o ON op."order_id" = o."id" JOIN "Customers" c ON o."customer_id" = c."id" JOIN "Products" p ON op."product_id" = p."id" GROUP BY c."name" ORDER BY "profit" DESC LIMIT 1; | [{"name":"Customer 9","total_spend":28500}] |
| How many orders have been shipped? | SELECT COUNT(*) FROM "Orders" WHERE "shipping_status" = 'shipped';| [{"COUNT(*)":1}] |
| What is the total value of orders we have? | SELECT SUM(op."quantity" * p."price") AS "total_value" FROM "Order_Products" op JOIN "Orders" o ON op."order_id" = o."id" JOIN "Products" p ON op "product_id" = p."id"; | [{"total_value":82500}] |
| How many copies of "Product 7" have been sold? | SELECT SUM("quantity") AS "total_sold" FROM "Order_Products" WHERE "product_id" = (SELECT "id" FROM "Products" WHERE "name" = 'Product 7'); | [{"total_sold":21}] |