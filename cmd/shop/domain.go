package main

import "fmt"

// Product là một mặt hàng trong catalog.
type Product struct {
	ID    string
	Name  string
	Price float64
}

// CartItem là một dòng trong giỏ hàng: sản phẩm + số lượng.
type CartItem struct {
	Product  Product
	Quantity int
}

func (ci CartItem) Subtotal() float64 {
	return ci.Product.Price * float64(ci.Quantity)
}

// Cart giữ danh sách CartItem. Zero value (Cart{}) đã dùng được ngay
// vì slice nil vẫn append được — không cần constructor bắt buộc.
type Cart struct {
	Items []CartItem
}

// AddItem thêm sản phẩm vào giỏ. Nếu sản phẩm đã có trong giỏ, cộng dồn
// số lượng thay vì tạo dòng mới.
func (c *Cart) AddItem(p Product, qty int) {
	for i := range c.Items {
		if c.Items[i].Product.ID == p.ID {
			c.Items[i].Quantity += qty
			return
		}
	}
	c.Items = append(c.Items, CartItem{Product: p, Quantity: qty})
}

// Total tính tổng tiền giỏ hàng trước discount.
func (c Cart) Total() float64 {
	var total float64
	for _, item := range c.Items {
		total += item.Subtotal()
	}
	return total
}

// Discounter được implement ngầm định (implicit satisfaction) — không cần
// khai báo "implements Discounter" như C# interface.
type Discounter interface {
	Apply(total float64) float64
}

// PercentDiscount giảm giá theo phần trăm, ví dụ Percent: 10 = giảm 10%.
type PercentDiscount struct {
	Percent float64
}

func (d PercentDiscount) Apply(total float64) float64 {
	return total - (total * d.Percent / 100)
}

// FixedDiscount giảm giá một số tiền cố định, không âm.
type FixedDiscount struct {
	Amount float64
}

func (d FixedDiscount) Apply(total float64) float64 {
	result := total - d.Amount
	if result < 0 {
		return 0
	}
	return result
}

// CheckoutSummary in ra tổng kết giỏ hàng, áp dụng discount nếu có.
// d có thể là nil (interface value), khi đó không áp dụng giảm giá.
func CheckoutSummary(c Cart, d Discounter) string {
	total := c.Total()
	final := total
	if d != nil {
		final = d.Apply(total)
	}
	return fmt.Sprintf("Subtotal: %.2f | Final: %.2f", total, final)
}
