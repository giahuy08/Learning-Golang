package main

import "fmt"

// Giai đoạn 1 (docs/22-mini-ecommerce-project.md): domain models thuần Go,
// không HTTP, không DB. Chạy: go run ./cmd/shop
func main() {
	catalog := []Product{
		{ID: "p1", Name: "Keyboard", Price: 49.99},
		{ID: "p2", Name: "Mouse", Price: 19.99},
		{ID: "p3", Name: "Monitor", Price: 199.00},
	}

	var cart Cart
	cart.AddItem(catalog[0], 1)
	cart.AddItem(catalog[1], 2)
	cart.AddItem(catalog[2], 1)
	cart.AddItem(catalog[0], 1) // mua thêm 1 keyboard -> cộng dồn số lượng

	fmt.Println("No discount:")
	fmt.Println(CheckoutSummary(cart, nil))

	fmt.Println("\n10% off:")
	fmt.Println(CheckoutSummary(cart, PercentDiscount{Percent: 10}))

	fmt.Println("\n$20 off:")
	fmt.Println(CheckoutSummary(cart, FixedDiscount{Amount: 20}))
}
