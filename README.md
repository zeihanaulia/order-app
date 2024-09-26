## Fiber Order Workflow Project

This project simulates an **order processing system** based on the provided BPMN diagram using the **Fiber framework** in Go. It represents key workflows such as payment processing, order creation, routing, fulfillment, and handling cancellations or refunds. Each part of the workflow is exposed as an HTTP endpoint, which interacts with the order state.

### Features:
- **Payment Processing**: Simulate payment authorization and capture.
- **Order Management**: Endpoints to create, route, and fulfill orders.
- **Grace Period Handling**: Introduce delays (grace period) before moving to the next stage.
- **Routing & Fulfillment**: Route orders to fulfillment centers and handle different fulfillment strategies.
- **Refund Handling**: Simulate refunds and cancellation processes.
- **State Tracking**: Each order is tracked using an internal map to simulate state changes as it progresses through the system.

### Endpoints:
1. **`POST /process-payment?order_id={id}&amount={amount}`** - Process the payment for an order.
2. **`POST /create-order?order_id={id}`** - Create the order after payment.
3. **`GET /wait-grace-period?order_id={id}`** - Wait for a grace period before proceeding.
4. **`POST /route-order?order_id={id}`** - Route the order to fulfillment centers.
5. **`POST /fulfill-order?order_id={id}`** - Fulfill the order (store/DC).
6. **`POST /capture-payment?order_id={id}`** - Capture the payment after fulfillment.
7. **`POST /refund-payment?order_id={id}`** - Refund payment for canceled orders.
8. **`POST /cancel-order?order_id={id}`** - Cancel the order.

### Installation & Setup:
1. Clone the repository:
   ```bash
   git clone https://github.com/your-repo/fiber-order-workflow.git
   cd fiber-order-workflow
   ```
2. Install dependencies:
   ```bash
   go mod tidy
   ```
3. Run the application:
   ```bash
   go run main.go
   ```

4. Test the endpoints using cURL, Postman, or any other API testing tool:
   ```bash
   curl -X POST http://localhost:3000/process-payment?order_id=123&amount=100
   ```

### Project Structure:
```
.
├── main.go          # Entry point of the project
├── go.mod           # Go module dependencies
└── README.md        # Project documentation
```

### Warning:
**This project does NOT represent a real-world production system.** It is a **simplified simulation** designed to demonstrate how to implement a BPMN workflow using the **Fiber framework**. Critical elements such as error handling, persistence, concurrency management, and security are minimal or missing. **Do not use this in a production environment without proper modifications and enhancements.**

### To Do:
- Integrate a real database for persisting order states.
- Add proper authentication and authorization.
- Add concurrency safety to handle multiple orders concurrently.
- Improve error handling and logging.
