package factories

// HTTP request
type AbstractFactory interface {
	CreateProduct() AbstractProduct
}

// Query parameters and other info that is included in a query to the service
type AbstractProduct interface {
	PerformAction(data map[string]string) (map[string]interface{}, error)
}
