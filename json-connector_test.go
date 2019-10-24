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
	ID     int      `json:"client_id"`
	Name   string   `json:"name"`
	Email  string   `json:"email"`
	Orders []*Order `json:"orders" jc:"ID,ClientID"`
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
		Where("ID", "=", 1).
		AddDependency("Client", "./testdata/clients.json").
		AddDependency("Product", "./testdata/products.json").
		Unmarshal(); err != nil {
		panic(err)
	}
	fmt.Println("order result:")
	fmt.Printf("%+v\n", order)
	fmt.Printf("%+v\n%+v\n", order.Product, order.Client)

	fmt.Println("--------")

	var client *Client
	if err := NewJsonConnector(&client, "./testdata/clients.json").
		Where("client_id", "=", 2).
		AddDependency("Orders", "./testdata/orders.json").
		Unmarshal(); err != nil {
		panic(err)
	}
	fmt.Println("client result:")
	fmt.Printf("%+v\n", client)
	for _, v := range client.Orders {
		fmt.Printf("%+v\n", v)
	}
}
