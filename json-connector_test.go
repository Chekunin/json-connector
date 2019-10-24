package json_connector

import (
	"fmt"
	"testing"
)

type Product struct {
	ID    int    `json:"product_id"`
	Title string `json:"title"`
	Price int    `json:"price"`
}

type Client struct {
	ID    int    `json:"client_id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type Order struct {
	ID        int      `json:"order_id"`
	ClientID  int      `json:"client_id"`
	ProductID int      `json:"product_id"`
	Client    *Client  `json:"client" jc:"ClientID,ID"`
	Product   *Product `json:"product" jc:"ProductID,ID"`
}

func TestDefault(t *testing.T) {
	var order *Order
	if err := NewJsonConnector(&order, "./testdata/orders.json").
		Where("order_id", "=", 1).
		AddDependency("Client", "./testdata/clients.json").
		AddDependency("Product", "./testdata/products.json").
		Unmarshal(); err != nil {
		panic(err)
	}
	fmt.Println("result:")
	fmt.Println(order)
	//for _, v := range order {
	//	//fmt.Printf("%+v", *v)
	//	fmt.Println("ID:", v.ID)
	//	fmt.Println("ClientID:", v.ClientID)
	//	fmt.Println("ProductID:", v.ProductID)
	//	fmt.Println("Product:", v.Product)
	//	fmt.Println("Client:", v.Client)
	//	fmt.Println("-----")
	//}
}
