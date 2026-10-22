package main

import (
	"fmt"
	"slices"
)

// seedCount is how many products are generated at startup. The large count is
// there to give the service a realistic memory footprint.
const seedCount = 100_000

// brands is the pool used for both the product name and the manufacturer.
var brands = []string{"Alpha", "Bravo", "Charlie", "Delta", "Echo", "Foxtrot", "Golf", "Hotel"}

// categoryIDs lists every known category_id in ascending order, so seeding is
// the same on every start. It is built from categoryNamesMap, because ranging
// over a map directly gives a random order.
var categoryIDs = sortedCategoryIDs()

func sortedCategoryIDs() []int32 {
	ids := make([]int32, 0, len(categoryNamesMap))
	for key := range categoryNamesMap {
		ids = append(ids, key)
	}
	slices.Sort(ids)
	return ids
}

// seedProducts preloads the store with seedCount products. Add assigns IDs
// 1..seedCount in order, so the ID in each name matches the stored product_id.
// Names follow "Product [Brand] [ID]", e.g. "Product Alpha 1", and are kept in
// the sku field. The data is deterministic: no randomness.
func seedProducts(store *ProductStore) {
	for i := 1; i <= seedCount; i++ {
		brand := brands[i%len(brands)]
		store.Add(Product{
			SKU:          fmt.Sprintf("Product %s %d", brand, i),
			Manufacturer: brand,
			CategoryID:   categoryIDs[i%len(categoryIDs)],
			Weight:       int32(100 + i%900),
			SomeOtherID:  int32(1 + i%1000),
		})
	}
}
