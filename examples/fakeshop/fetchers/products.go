package fetchers

import (
	"context"
	"fmt"

	"github.com/heetch/datablocks/pkg/datablocks"
	"github.com/heetch/universe/src/services/pickup-experience/core/internal/composer"
)

// Produce contains the information for a given product in the store
type Product struct {
	Sku         string
	Name        string
	ImageURL    string
	Description string
	Price       int64
}

// ProductAsynReq prepares a function that conforms to the
// `DataFetcherFn` data type by preparing a closure on the
// input params. It also prepares a "hash" or unique ID for
// this function call (and params) so the datafecher can
// maintain a local temporary cache for the result.
func ProductAsyncReq(productID string) *composer.AsyncFetchReq {
	ad := composer.AsyncFetchReq{
		Fetcher: func(c context.Context) (interface{}, error) {
			// here we should fetch the required data, now we fake it
			return Product{
				Sku:         "1",
				Name:        "a book",
				ImageURL:    "https//www.example.com/img1.jpg",
				Description: "beautiful book",
			}, nil
		},
		Hash: fmt.Sprintf("Product_%s", productID),
	}
	return &ad
}

// ToProduct is a helper function to cast the interface{}
// result from ProductAsyncReq to the typed struct.
func ToProduct(d *datablocks.AsyncFetchData) (*Product, error) {
	if d == nil {
		return nil, fmt.Errorf("cannot cast nil value")
	}
	out, ok := d.Result.(*Customer)
	if !ok {
		return nil, fmt.Errorf("cannot cast")
	}
	return out, nil
}

// ProductList returns a list of products
type ProductList []Product

// RelatedProductsAsynReq .
func RelatedProductsAsyncReq(productID string) *composer.AsyncFetchReq {
	ad := composer.AsyncFetchReq{
		Fetcher: func(c context.Context) (interface{}, error) {
			// here we should fetch the required data, now we fake it
			return ProductList{
				Product{
					Sku:         "2",
					Name:        "another book",
					ImageURL:    "https//www.example.com/img2.jpg",
					Description: "another beautiful book",
				},
				Product{
					Sku:         "3",
					Name:        "a pencil",
					ImageURL:    "https//www.example.com/img3.jpg",
					Description: "a pencil to take notes",
				},
			}, nil
		},
		Hash: fmt.Sprintf("Product_%s", productID),
	}
	return &ad
}

// ToProductList is a helper function to cast the interface{}
// result from ProductAsyncReq to the typed struct.
func ToProductList(d *datablocks.AsyncFetchData) (*Product, error) {
	if d == nil {
		return nil, fmt.Errorf("cannot cast nil value")
	}
	out, ok := d.Result.([]ProductList)
	if !ok {
		return nil, fmt.Errorf("cannot cast")
	}
	return out, nil
}
