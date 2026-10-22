package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

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

// SearchResponse is the body of GET /products/search.
type SearchResponse struct {
	Products   []Product `json:"products"`    // at most 20, never nil
	TotalCount int       `json:"total_count"` // matches among the 100 products checked
	SearchTime float64   `json:"search_time"` // milliseconds spent scanning
}

// categoryNamesMap maps category_id to a lowercase category name. The names are
// only used for matching, so they are lowercase to compare against a lowercased
// query. It is never written to after startup, so concurrent reads need no lock.
var categoryNamesMap = map[int32]string{
	1: "electronics",
	2: "clothing",
	3: "books",
	4: "home",
	5: "toys",
}

// ProductStore keeps products in memory, keyed by product ID (int64). Gin serves
// each request in its own goroutine, so products is a sync.Map and nextID is
// atomic: no mutex is needed. Do not copy a ProductStore; use the pointer.
type ProductStore struct {
	products sync.Map
	nextID   atomic.Int64 // last ID handed out; the first Add returns 1
}

// NewProductStore returns an empty store. The zero values of sync.Map and
// atomic.Int64 are ready to use.
func NewProductStore() *ProductStore {
	return &ProductStore{}
}

// Get returns the product with the given ID. The bool is false if it doesn't exist.
func (s *ProductStore) Get(id int64) (Product, bool) {
	v, ok := s.products.Load(id)
	if !ok {
		return Product{}, false
	}
	return v.(Product), true
}

// Add assigns the next ID to newProduct, stores it, and returns the stored copy.
// The ID comes from an atomic counter, so concurrent calls get unique IDs.
// Only seedProducts calls it today: no endpoint creates products.
func (s *ProductStore) Add(newProduct Product) Product {
	newProduct.ProductID = s.nextID.Add(1)
	s.products.Store(newProduct.ProductID, newProduct)
	return newProduct
}

// Update replaces an existing product. It returns false, changing nothing, if
// productID is not in the store.
func (s *ProductStore) Update(productID int64, newProduct Product) (Product, bool) {
	oldProduct, ok := s.products.Load(productID)
	if !ok {
		return Product{}, false
	}

	newProduct.ProductID = productID
	if !s.products.CompareAndSwap(productID, oldProduct, newProduct) {
		return Product{}, false
	}
	return newProduct, true
}

// FindAll checks exactly 100 products, the IDs offset+1..offset+100, so every
// search costs the same amount of work. A product matches if the lowercased query
// is contained in its name (stored in SKU) or in its category name (derived from
// category_id). It returns at most 20 matching products, the total number of
// matches among the 100 checked, and the scan time in milliseconds.
// It never stops early, even after 20 matches.
func (s *ProductStore) FindAll(query string, offset int64) ([]Product, int, float64) {
	var products []Product
	query = strings.ToLower(query)
	matches := 0
	start := time.Now()
	for count := int64(1); count <= 100; count++ {
		v, ok := s.products.Load(offset + count)
		if ok {
			prod := v.(Product)
			if strings.Contains(strings.ToLower(prod.SKU), query) || strings.Contains(categoryNamesMap[prod.CategoryID], query) {
				if matches < 20 {
					products = append(products, prod)
				}
				matches++
			}
		}
	}
	return products, matches, float64(time.Since(start).Nanoseconds()) / 1e6
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

// findProducts handles GET /products/search?q=...
// Responses: 200 with a SearchResponse (an empty list when nothing matches),
// 400 if q is missing, empty or longer than 100 characters.
// offset is an optional extra parameter that is not in the yaml. It is lenient
// on purpose: a missing, unparsable or negative value becomes 0 and a value
// above 99900 becomes 99900, so the 100-product window always stays inside the
// data and a bad offset never causes an error.
func (s *ProductStore) findProducts(c *gin.Context) {
	query := strings.TrimSpace(c.Query("q"))
	if query == "" {
		writeError(c, http.StatusBadRequest, errInvalidInput, "The provided input data is invalid", "The query param can't be empty")
		return
	}
	if len(query) > 100 {
		writeError(c, http.StatusBadRequest, errInvalidInput, "The provided input data is invalid", "The query param can't be more than 100 characters")
		return
	}

	offset, err := strconv.ParseInt(c.DefaultQuery("offset", "0"), 10, 64)
	if err != nil || offset < 0 {
		offset = 0
	}
	if offset > 99900 {
		offset = 99900
	}

	products, totalCount, searchTime := s.FindAll(query, offset)
	if products == nil {
		products = []Product{} // JSON [] instead of null
	}
	c.JSON(http.StatusOK, SearchResponse{Products: products, TotalCount: totalCount, SearchTime: searchTime})
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
				logger.Error("panic", "error", rec)
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

	// gin.New() has no middleware; gin.Default() would add its own logger and
	// recovery. observabilityMiddleware goes first so it also sees the 500 that
	// recoverMiddleware produces after a panic.
	router := gin.New()
	router.Use(observabilityMiddleware(), recoverMiddleware())

	router.GET("/metrics", metricsHandler())
	router.GET("/products/:productId", store.getProduct)
	router.POST("/products/:productId/details", store.postProductDetails)
	router.GET("/products/search", store.findProducts)

	log.Fatal(router.Run(":8080"))
}
