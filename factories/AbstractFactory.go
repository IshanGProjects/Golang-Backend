package factories

type AbstractFactory interface {
	CreateProduct() AbstractProduct
}

type AbstractProduct interface {
	PerformAction(data map[string]string) (map[string]interface{}, error)
}

// Common settings and types used across products
var defaultApiSettings = struct {
	Size int
	Page int
}{
	Size: 20,
	Page: 0,
}
