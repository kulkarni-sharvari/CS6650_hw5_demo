package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"

	"github.com/gin-gonic/gin"
)

// Error codes returned in the "error" field of APIErrResponse.
const (
	errInvalidInput = "INVALID_INPUT"
	errNotFound     = "NOT_FOUND"
	errInternal     = "INTERNAL_ERROR"
)

// Product according to Product schema in api.yaml.
type Product struct {
	ProductID    int64  `json:"product_id"`
	SKU          string `json:"sku"`
	Manufacturer string `json:"manufacturer"`
	CategoryID   int32  `json:"category_id"`
	Weight       int32  `json:"weight"`
	SomeOtherID  int32  `json:"some_other_id"`
}

// APIErrResponse is the JSON body for every error, matching the Error schema in api.yaml.
type APIErrResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Details string `json:"details"`
}

// ProductStore keeps products in memory, keyed by product ID. The mutex guards
// products and nextID, because Gin serves each request in its own goroutine.
type ProductStore struct {
	mu       sync.RWMutex
	products map[int64]Product
	nextID   int64 // next ID. starts at 1
}

// NewProductStore returns an empty store. The map must be created here otherwise  writing to a nil map panics.
func NewProductStore() *ProductStore {
	return &ProductStore{
		products: make(map[int64]Product),
		nextID:   1,
	}
}

// Get returns the product with the given ID. The bool is false if it doesn't exist.
func (s *ProductStore) Get(id int64) (Product, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	p, ok := s.products[id]

	return p, ok
}

// Add assigns the next ID to newProduct, stores it, and returns the stored copy.
// Only seedProducts calls it: no endpoint creates products.
func (s *ProductStore) Add(newProduct Product) Product {
	s.mu.Lock()
	defer s.mu.Unlock()

	newProduct.ProductID = s.nextID
	s.products[s.nextID] = newProduct
	s.nextID++

	return newProduct
}

// Update replaces an existing product. It returns false, changing nothing, if
// productID is not in the store.
func (s *ProductStore) Update(productID int64, newProduct Product) (Product, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.products[productID]; !ok {
		return Product{}, false
	}

	newProduct.ProductID = productID
	s.products[productID] = newProduct
	return newProduct, true
}

// seedProducts preloads the store. POST only updates existing products, so
// there must be some to update. To add another, add one line to the slice.
func seedProducts(store *ProductStore) {
	seed := []Product{
		{SKU: "ABC-123-XYZ", Manufacturer: "Acme Corporation", CategoryID: 1, Weight: 500, SomeOtherID: 101},
		{SKU: "DEF-456-UVW", Manufacturer: "Globex", CategoryID: 2, Weight: 1250, SomeOtherID: 102},
		{SKU: "GHI-789-RST", Manufacturer: "Initech", CategoryID: 1, Weight: 300, SomeOtherID: 103},
	}

	for _, p := range seed {
		store.Add(p)
	}
}

// ---- Handlers and helpers ----

// parseProductID reads the :productId path parameter. It returns false if the
// value is not a positive integer
func parseProductID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("productId"), 10, 64)
	if err != nil || id < 1 {
		return 0, false
	}
	return id, true
}

// getProduct handles GET /products/:productId.
// Responses: 200 with the product, 404 if the ID is invalid or not found.
// GET has no 400 in api.yaml, so a malformed ID is reported as 404.
func (s *ProductStore) getProduct(c *gin.Context) {
	id, ok := parseProductID(c)
	if !ok {
		writeError(c, http.StatusNotFound, errInvalidInput, "The provided input data is invalid", "Product ID must be a positive integer")
		return
	}

	p, ok := s.Get(id)
	if !ok {
		writeNotFound(c, id)
		return
	}
	c.JSON(http.StatusOK, p)
}

// postProductDetails handles POST /products/:productId/details.
// It replaces the details of an existing product; it never creates one.
// Responses: 204 on success, 400 for a bad ID, malformed JSON or invalid
// fields, 404 if the product doesn't exist.
func (s *ProductStore) postProductDetails(c *gin.Context) {
	id, ok := parseProductID(c)
	if !ok {
		writeError(c, http.StatusBadRequest, errInvalidInput, "The provided input data is invalid", "Product ID must be a positive integer")
		return
	}

	// ShouldBindJSON (not BindJSON): BindJSON writes its own 400 response,
	// which would bypass our error format.
	var product Product
	if err := c.ShouldBindJSON(&product); err != nil {
		writeError(c, http.StatusBadRequest, errInvalidInput, "Invalid request body", "Bad request")
		return
	}

	if msg := validateProduct(product); msg != "" {
		writeError(c, http.StatusBadRequest, errInvalidInput, "The provided input data is invalid", msg)
		return
	}

	if _, ok := s.Update(id, product); !ok {
		writeNotFound(c, id)
		return
	}
	// 204 must not have a body, so set only the status instead of calling c.JSON.
	c.Status(http.StatusNoContent)
}

// validateProduct checks the request body against the Product schema in
// api.yaml. It returns a description of the first problem, or "" if valid.
func validateProduct(p Product) string {
	switch {
	case len(p.SKU) < 1 || len(p.SKU) > 100:
		return "sku is required and must be 1-100 characters"
	case len(p.Manufacturer) < 1 || len(p.Manufacturer) > 200:
		return "manufacturer is required and must be 1-200 characters"
	case p.CategoryID < 1:
		return "category_id is required and must be a positive integer"
	case p.Weight < 0:
		return "weight must be 0 or greater"
	case p.SomeOtherID < 1:
		return "some_other_id is required and must be a positive integer"
	}
	return ""
}

// writeError sends an Error body (error, message, details) with the given status.
func writeError(c *gin.Context, status int, code string, msg string, details string) {
	c.JSON(status, APIErrResponse{
		Error:   code,
		Message: msg,
		Details: details,
	})
}

// writeNotFound sends the 404 used when a valid ID has no matching product.
func writeNotFound(c *gin.Context, id int64) {
	writeError(c, http.StatusNotFound, errNotFound, "Product not found", fmt.Sprintf("No product with ID %d", id))
}

// recoverMiddleware turns a panic in any handler into a 500 response in our
// Error format. gin.Default()'s built-in recovery returns an empty body instead.
func recoverMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				// Log the cause on the server; send the client a generic message.
				log.Printf("panic: %v", rec)
				writeError(c, http.StatusInternalServerError, errInternal,
					"Internal server error", "An unexpected error occurred")
				c.Abort()
			}
		}()
		// Test hook: lets the 500 response be demonstrated, since the in-memory
		// store can't fail on its own. The panic is caught by the defer above.
		if c.GetHeader("X-Force-Error") == "true" {
			panic("forced error for testing")
		}
		c.Next()
	}
}

func main() {
	store := NewProductStore()
	seedProducts(store)

	// gin.New() has no middleware; gin.Default() would add its own recovery.
	router := gin.New()
	router.Use(gin.Logger(), recoverMiddleware())

	router.GET("/products/:productId", store.getProduct)
	router.POST("/products/:productId/details", store.postProductDetails)

	log.Fatal(router.Run(":8080"))
}
