package factories

type AbstractFactory interface {
	CreateProduct() AbstractProduct
}

type AbstractProduct interface {
	PerformAction(data map[string]string) (map[string]interface{}, error)
}
