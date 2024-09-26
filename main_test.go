package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

// Helper function to set up the Fiber app for testing
func setupApp() *fiber.App {
	app := fiber.New()
	setupRoutes(app) // Ensure that your routes are initialized
	return app
}

// Test Create Cart
func TestCreateCart(t *testing.T) {
	app := setupApp() // Create a fresh instance of the app for each test

	// Define the payload
	payload := `{
		"customer_id": "cust_12345",
		"items": [
			{
				"item_id": "item001",
				"name": "Laptop",
				"quantity": 1,
				"price": 1000
			},
			{
				"item_id": "item002",
				"name": "Mouse",
				"quantity": 2,
				"price": 50
			}
		]
	}`

	// Create a request
	req := httptest.NewRequest(http.MethodPost, "/create-cart", bytes.NewBuffer([]byte(payload)))
	req.Header.Set("Content-Type", "application/json")

	// Perform the request
	resp, err := app.Test(req, -1) // -1 disables request timeout for the test
	assert.NoError(t, err)

	// Check the status code
	assert.Equal(t, 200, resp.StatusCode)

	// Parse the response body
	var responseBody map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&responseBody)
	assert.NoError(t, err)

	// Check if the "cart" key exists and has an ID
	cart, exists := responseBody["cart"]
	assert.True(t, exists, "Response should contain 'cart' key")

	cartMap := cart.(map[string]interface{})
	assert.NotEmpty(t, cartMap["cart_id"], "Cart should have an 'id'")
}

// Test Process Payment
func TestProcessPayment(t *testing.T) {
	// Initialize the app and setup routes
	app := setupApp()

	// First, create a cart to use in the payment process
	cartPayload := `{
		"customer_id": "cust_12345",
		"items": [
			{
				"item_id": "item001",
				"name": "Laptop",
				"quantity": 1,
				"price": 1000
			},
			{
				"item_id": "item002",
				"name": "Mouse",
				"quantity": 2,
				"price": 50
			}
		]
	}`

	reqCart := httptest.NewRequest(http.MethodPost, "/create-cart", bytes.NewBuffer([]byte(cartPayload)))
	reqCart.Header.Set("Content-Type", "application/json")

	// Perform the create cart request
	respCart, err := app.Test(reqCart, -1)
	assert.NoError(t, err)

	// Assert that the cart was created successfully
	assert.Equal(t, 200, respCart.StatusCode)

	var cartResponse map[string]interface{}
	err = json.NewDecoder(respCart.Body).Decode(&cartResponse)
	assert.NoError(t, err)

	// Extract cart ID from the response
	cartID := cartResponse["cart"].(map[string]interface{})["cart_id"].(string)

	// Now, process the payment for the created cart
	paymentPayload := map[string]interface{}{
		"cart_id": cartID,
		"billing_address": map[string]interface{}{
			"customer_id": "cust_12345",
			"name":        "John Doe",
			"email":       "john@example.com",
			"phone":       "555-5555",
			"address":     "123 Main St",
			"city":        "New York",
			"postal_code": "10001",
			"country":     "USA",
		},
		"amount": 1100, // Amount matches the total of the cart (Laptop + 2 Mice)
	}

	paymentPayloadBytes, _ := json.Marshal(paymentPayload)
	reqPayment := httptest.NewRequest(http.MethodPost, "/process-payment", bytes.NewBuffer(paymentPayloadBytes))
	reqPayment.Header.Set("Content-Type", "application/json")

	// Perform the process payment request
	respPayment, err := app.Test(reqPayment, -1)
	assert.NoError(t, err)

	// Assert that the payment was processed successfully
	assert.Equal(t, 200, respPayment.StatusCode)

	// Parse the payment response
	var paymentResponse map[string]interface{}
	err = json.NewDecoder(respPayment.Body).Decode(&paymentResponse)
	assert.NoError(t, err)

	// Check if payment was processed
	order := paymentResponse["order"].(map[string]interface{})
	assert.Equal(t, "Payment Processed", order["Status"])
	assert.Equal(t, 1100.0, order["Amount"])
	assert.Equal(t, "John Doe", order["Customer"].(map[string]interface{})["name"])
}
