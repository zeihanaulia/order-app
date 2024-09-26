package main

import (
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

var log zerolog.Logger

// Struct to store customer details
type CustomerInfo struct {
	ID    string
	Name  string
	Email string
	Phone string
}

// Struct to represent Order
type Order struct {
	ID          string
	Status      string
	Amount      float64
	Items       []OrderItem
	Fulfilled   bool
	PaymentDone bool
	Customer    BillingAddress
	ProcessedBy string
	Refunded    bool
	Cancelled   bool
}

// Struct to store billing address details
type BillingAddress struct {
	CustomerID string `json:"customer_id"`
	Name       string `json:"name"`
	Email      string `json:"email"`
	Phone      string `json:"phone"`
	Address    string `json:"address"`
	City       string `json:"city"`
	PostalCode string `json:"postal_code"`
	Country    string `json:"country"`
}

// Struct to represent an item in the order
type OrderItem struct {
	ItemID   string  `json:"item_id"`
	Name     string  `json:"name"`
	Quantity int     `json:"quantity"`
	Price    float64 `json:"price"`
}

// Struct to represent payment request
type PaymentRequest struct {
	OrderID        string         `json:"order_id"`
	Amount         float64        `json:"amount"`
	BillingAddress BillingAddress `json:"billing_address"`
}

// Struct to represent create order request
type CreateOrderRequest struct {
	OrderID        string         `json:"order_id"`
	BillingAddress BillingAddress `json:"billing_address"`
	Items          []OrderItem    `json:"items"`
}

type Item struct {
	ItemID   string  `json:"item_id"`
	Name     string  `json:"name"`
	Quantity int     `json:"quantity"`
	Price    float64 `json:"price"`
}

type Cart struct {
	CartID     string `json:"cart_id"`
	CustomerID string `json:"customer_id"`
	Items      []Item `json:"items"`
}

// Mock in-memory storage for carts
var carts = make(map[string]*Cart)

// Request struct for creating a cart
type CartRequest struct {
	CustomerID string `json:"customer_id"`
	Items      []Item `json:"items"`
}

// In-memory order storage (for demo purposes)
var orders = make(map[string]*Order)

func CreateCartHandler(c *fiber.Ctx) error {
	var cartReq CartRequest

	// Parse the JSON input for cart creation
	if err := c.BodyParser(&cartReq); err != nil {
		log.Warn().Msg("Invalid JSON input for /create-cart")
		return c.Status(400).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "InvalidRequest",
				"message": "Invalid JSON payload",
			},
		})
	}

	// Validate cart input
	if cartReq.CustomerID == "" || len(cartReq.Items) == 0 {
		log.Warn().Msg("Customer ID or items missing in /create-cart request")
		return c.Status(400).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "InvalidRequest",
				"message": "Customer ID and items are required",
			},
		})
	}

	// Create or update the cart for the customer
	carts[cartReq.CustomerID] = &Cart{
		CartID:     uuid.New().String(),
		CustomerID: cartReq.CustomerID,
		Items:      cartReq.Items,
	}

	log.Info().Str("event.action", "create_cart").
		Str("customer.id", cartReq.CustomerID).
		Msg("Cart created successfully")

	return c.JSON(fiber.Map{
		"message": "Cart created successfully",
		"cart":    carts[cartReq.CustomerID],
	})
}

func ProcessPaymentHandler(c *fiber.Ctx) error {
	var paymentReq PaymentRequest

	// Parse JSON input
	if err := c.BodyParser(&paymentReq); err != nil {
		log.Warn().Msg("Invalid JSON input for /process-payment")
		return c.Status(400).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "InvalidRequest",
				"message": "Invalid JSON payload",
			},
		})
	}

	// Validate payment input
	if paymentReq.Amount <= 0 ||
		paymentReq.BillingAddress.CustomerID == "" ||
		paymentReq.BillingAddress.Name == "" ||
		paymentReq.BillingAddress.Email == "" ||
		paymentReq.BillingAddress.Phone == "" {
		log.Warn().Msg("Invalid parameters for /process-payment")
		return c.Status(400).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "InvalidRequest",
				"message": "Billing details and valid amount are required",
			},
		})
	}

	// Retrieve cart associated with the billing address
	cart, exists := carts[paymentReq.BillingAddress.CustomerID]
	if !exists {
		log.Warn().Msgf("Cart for customer ID %s not found", paymentReq.BillingAddress.CustomerID)
		return c.Status(404).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "CartNotFound",
				"message": "Cart for the given customer ID not found",
			},
		})
	}

	// Calculate total cart amount
	var totalAmount float64
	for _, item := range cart.Items {
		totalAmount += float64(item.Quantity) * item.Price
	}

	// Check if the total amount matches the payment amount
	if totalAmount != paymentReq.Amount {
		log.Warn().Msgf("Payment amount mismatch: expected %.2f, received %.2f", totalAmount, paymentReq.Amount)
		return c.Status(400).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "AmountMismatch",
				"message": "The payment amount does not match the total cart amount",
			},
		})
	}

	// If amounts match, process payment (In real-world scenario, integrate with payment gateway)
	log.Info().Str("event.action", "process_payment").
		Str("customer.id", paymentReq.BillingAddress.CustomerID).
		Float64("amount", paymentReq.Amount).
		Msg("Payment processed successfully")

	// Create the order after successful payment
	orderID := paymentReq.OrderID // Example, should be unique

	// Convert cart.Items (of type []Item) to []OrderItem
	orderItems := make([]OrderItem, len(cart.Items))
	for i, item := range cart.Items {
		orderItems[i] = OrderItem{
			ItemID:   item.ItemID,
			Name:     item.Name,
			Quantity: item.Quantity,
			Price:    item.Price,
		}
	}

	orders[orderID] = &Order{
		ID:          orderID,
		Status:      "Payment Processed",
		Amount:      totalAmount,
		Items:       orderItems, // Use the converted orderItems
		PaymentDone: true,
		Customer:    paymentReq.BillingAddress,
		ProcessedBy: "System",
	}

	return c.JSON(fiber.Map{
		"message": "Payment processed successfully and order created",
		"order":   orders[orderID],
	})
}

func WaitGracePeriodHandler(c *fiber.Ctx) error {
	// Get the order ID from the query parameter
	orderID := c.Query("order_id")
	if orderID == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "InvalidRequest",
				"message": "Order ID is required",
			},
		})
	}

	// Check if order exists
	order, exists := orders[orderID]
	if !exists {
		return c.Status(404).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "OrderNotFound",
				"message": "Order not found",
			},
		})
	}

	// Simulate a grace period (e.g., 5 seconds)
	log.Info().Str("order.id", orderID).Msg("Starting grace period for order")
	time.Sleep(5 * time.Second) // You can change this to a configurable time

	// Update the order status after grace period
	order.Status = "Grace Period Completed"

	log.Info().Str("order.id", orderID).Msg("Grace period completed for order")

	// Return the updated order
	return c.JSON(fiber.Map{
		"message": "Grace period completed, order ready for routing",
		"order":   order,
	})
}

func RouteOrderHandler(c *fiber.Ctx) error {
	// Parse JSON input
	var payload map[string]string
	if err := c.BodyParser(&payload); err != nil {
		log.Warn().Msg("Invalid JSON input")
		return c.Status(400).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "InvalidRequest",
				"message": "Invalid JSON payload",
			},
		})
	}

	orderID := payload["order_id"]
	if orderID == "" {
		log.Warn().Msg("Order ID is missing in the route order request")
		return c.Status(400).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "MissingOrderID",
				"message": "Order ID is required to route the order.",
			},
		})
	}

	// Check if order exists
	order, exists := orders[orderID]
	if !exists {
		log.Warn().Msgf("Order ID %s not found", orderID)
		return c.Status(404).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "OrderNotFound",
				"message": "The order ID provided does not exist.",
				"target":  "order_id",
				"details": fiber.Map{
					"order_id": orderID,
				},
			},
		})
	}

	// Simulate routing success or failure
	success := true // This would be replaced by real routing logic
	if success {
		order.Status = "Order Routed"
		log.Info().Msgf("Order ID %s successfully routed", orderID)
		return c.JSON(fiber.Map{
			"message": "Order routed",
			"order":   order,
		})
	} else {
		order.Status = "Routing Failed"
		log.Warn().Msgf("Routing failed for Order ID %s", orderID)
		return c.JSON(fiber.Map{
			"message": "Routing failed, items on hold",
			"order":   order,
		})
	}
}

func FullfillOrderHandler(c *fiber.Ctx) error {
	// Parse JSON input
	var payload map[string]string
	if err := c.BodyParser(&payload); err != nil {
		log.Warn().Msg("Invalid JSON input for fulfillment")
		return c.Status(400).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "InvalidRequest",
				"message": "Invalid JSON payload",
			},
		})
	}

	orderID := payload["order_id"]
	if orderID == "" {
		log.Warn().Msg("Order ID is missing for fulfillment")
		return c.Status(400).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "MissingOrderID",
				"message": "Order ID is required to process fulfillment.",
			},
		})
	}

	// Check if order exists and has been routed
	order, exists := orders[orderID]
	if !exists {
		log.Warn().Msgf("Order ID %s not found for fulfillment", orderID)
		return c.Status(404).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "OrderNotFound",
				"message": "The order ID provided does not exist.",
				"target":  "order_id",
			},
		})
	}

	// Ensure the order has been routed before fulfillment
	if order.Status != "Order Routed" {
		log.Warn().Msgf("Order ID %s has not been routed", orderID)
		return c.Status(400).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "OrderNotRouted",
				"message": "The order must be routed before fulfillment.",
				"target":  "order_id",
			},
		})
	}

	// Simulate fulfillment
	order.Status = "Fulfillment Completed"
	order.Fulfilled = true

	// Log successful fulfillment
	log.Info().
		Str("event.action", "fulfill_order").
		Str("order.id", orderID).
		Msg("Order fulfilled successfully")

	return c.JSON(fiber.Map{
		"message": "Order fulfilled",
		"order":   order,
	})
}

func CapturePaymentHandler(c *fiber.Ctx) error {
	// Parse JSON input
	var payload map[string]string
	if err := c.BodyParser(&payload); err != nil {
		log.Warn().Msg("Invalid JSON input for payment capture")
		return c.Status(400).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "InvalidRequest",
				"message": "Invalid JSON payload",
			},
		})
	}

	orderID := payload["order_id"]
	if orderID == "" {
		log.Warn().Msg("Order ID is missing for payment capture")
		return c.Status(400).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "MissingOrderID",
				"message": "Order ID is required to capture the payment.",
			},
		})
	}

	// Check if order exists
	order, exists := orders[orderID]
	if !exists {
		log.Warn().Msgf("Order ID %s not found for payment capture", orderID)
		return c.Status(404).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "OrderNotFound",
				"message": "The order ID provided does not exist.",
				"target":  "order_id",
			},
		})
	}

	// Ensure the order is fulfilled before capturing payment
	if !order.Fulfilled {
		log.Warn().Msgf("Order ID %s has not been fulfilled yet", orderID)
		return c.Status(400).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "OrderNotFulfilled",
				"message": "The order must be fulfilled before capturing payment.",
				"target":  "order_id",
			},
		})
	}

	// Capture the payment
	order.Status = "Payment Captured"
	order.PaymentDone = true

	// Log successful payment capture
	log.Info().
		Str("event.action", "capture_payment").
		Str("order.id", orderID).
		Msg("Payment captured successfully")

	return c.JSON(fiber.Map{
		"message": "Payment captured",
		"order":   order,
	})
}

func RefundPaymentHandler(c *fiber.Ctx) error {
	// Parse JSON input
	var payload map[string]string
	if err := c.BodyParser(&payload); err != nil {
		log.Warn().Msg("Invalid JSON input for refund payment")
		return c.Status(400).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "InvalidRequest",
				"message": "Invalid JSON payload",
			},
		})
	}

	orderID := payload["order_id"]
	if orderID == "" {
		log.Warn().Msg("Order ID is missing for refund payment")
		return c.Status(400).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "MissingOrderID",
				"message": "Order ID is required to process the refund.",
			},
		})
	}

	// Check if order exists
	order, exists := orders[orderID]
	if !exists {
		log.Warn().Msgf("Order ID %s not found for refund", orderID)
		return c.Status(404).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "OrderNotFound",
				"message": "The order ID provided does not exist.",
				"target":  "order_id",
			},
		})
	}

	// Check if payment has been made and the refund has not already been processed
	if !order.PaymentDone {
		log.Warn().Msgf("Payment was not processed for Order ID %s", orderID)
		return c.Status(400).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "PaymentNotProcessed",
				"message": "Payment has not been processed for this order.",
				"target":  "order_id",
			},
		})
	}

	if order.Refunded {
		log.Warn().Msgf("Payment has already been refunded for Order ID %s", orderID)
		return c.Status(400).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "PaymentAlreadyRefunded",
				"message": "Payment has already been refunded for this order.",
				"target":  "order_id",
			},
		})
	}

	// Refund the payment
	order.Status = "Payment Refunded"
	order.Refunded = true

	// Log successful refund
	log.Info().
		Str("event.action", "refund_payment").
		Str("order.id", orderID).
		Msg("Payment refunded successfully")

	return c.JSON(fiber.Map{
		"message": "Payment refunded",
		"order":   order,
	})
}

func CancelOrderHandler(c *fiber.Ctx) error {
	// Parse JSON input
	var payload map[string]string
	if err := c.BodyParser(&payload); err != nil {
		log.Warn().Msg("Invalid JSON input for order cancellation")
		return c.Status(400).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "InvalidRequest",
				"message": "Invalid JSON payload",
			},
		})
	}

	orderID := payload["order_id"]
	if orderID == "" {
		log.Warn().Msg("Order ID is missing for cancellation")
		return c.Status(400).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "MissingOrderID",
				"message": "Order ID is required to cancel the order.",
			},
		})
	}

	// Check if order exists
	order, exists := orders[orderID]
	if !exists {
		log.Warn().Msgf("Order ID %s not found for cancellation", orderID)
		return c.Status(404).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "OrderNotFound",
				"message": "The order ID provided does not exist.",
				"target":  "order_id",
			},
		})
	}

	// Check if order is already fulfilled or cancelled
	if order.Fulfilled {
		log.Warn().Msgf("Order ID %s has already been fulfilled and cannot be cancelled", orderID)
		return c.Status(400).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "OrderAlreadyFulfilled",
				"message": "The order has already been fulfilled and cannot be cancelled.",
				"target":  "order_id",
			},
		})
	}

	if order.Cancelled {
		log.Warn().Msgf("Order ID %s has already been cancelled", orderID)
		return c.Status(400).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "OrderAlreadyCancelled",
				"message": "The order has already been cancelled.",
				"target":  "order_id",
			},
		})
	}

	// Cancel the order
	order.Status = "Order Cancelled"
	order.Cancelled = true

	// Log successful cancellation
	log.Info().
		Str("event.action", "cancel_order").
		Str("order.id", orderID).
		Msg("Order cancelled successfully")

	return c.JSON(fiber.Map{
		"message": "Order cancelled",
		"order":   order,
	})
}

func GetOrdersHandler(c *fiber.Ctx) error {
	// If no orders are available, return an empty list
	if len(orders) == 0 {
		log.Info().Msg("No orders available")
		return c.JSON(fiber.Map{
			"message": "No orders found",
			"orders":  []Order{},
		})
	}

	// Return all the orders in a JSON response
	log.Info().Msg("Fetching all orders")
	return c.JSON(fiber.Map{
		"message": "All orders retrieved successfully",
		"orders":  orders,
	})
}

// setupRoutes sets up the necessary routes for the application
func setupRoutes(app *fiber.App) {
	app.Post("/create-cart", CreateCartHandler)
	app.Post("/process-payment", ProcessPaymentHandler)
	app.Get("/wait-grace-period", WaitGracePeriodHandler)
	app.Post("/route-order", RouteOrderHandler)
	app.Post("/fulfill-order", FullfillOrderHandler)
	app.Post("/capture-payment", CapturePaymentHandler)
	app.Post("/refund-payment", RefundPaymentHandler)
	app.Post("/cancel-order", CancelOrderHandler)

	app.Get("/orders", GetOrdersHandler)
}

func main() {
	// Initialize zerolog logger
	log = zerolog.New(os.Stdout).With().Timestamp().Logger()

	app := fiber.New()

	// Middleware to recover from panics
	app.Use(func(c *fiber.Ctx) error {
		defer func() {
			if r := recover(); r != nil {
				// Capture stack trace
				stackBuf := make([]byte, 1024)
				runtime.Stack(stackBuf, false)

				// Log panic with ECS structured fields
				log.Error().
					Str("error.type", "panic").
					Str("error.message", "Recovered from panic").
					Str("file.name", "main.go").
					Bytes("error.stack_trace", stackBuf).
					Msgf("Panic: %v", r)

				c.Status(500).SendString("Internal Server Error")
			}
		}()
		return c.Next()
	})

	setupRoutes(app)

	// Graceful shutdown on SIGTERM or SIGINT
	go func() {
		// Listen for system signals
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

		<-quit // Wait for the signal

		log.Info().Msg("Gracefully shutting down...")

		if err := app.Shutdown(); err != nil {
			log.Error().Err(err).Msg("Error shutting down the server")
		}

		log.Info().Msg("Server successfully shut down.")
	}()

	// Start the Fiber app
	if err := app.Listen(":3000"); err != nil {
		log.Fatal().Err(err).Msg("Error starting server")
	}
}
