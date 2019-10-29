package json_connector

import (
	"fmt"
	"io/ioutil"
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
	dataOrders, err := ioutil.ReadFile("./testdata/orders.json")
	if err != nil {
		panic(err)
	}
	dataClients, err := ioutil.ReadFile("./testdata/clients.json")
	if err != nil {
		panic(err)
	}
	dataProducts, err := ioutil.ReadFile("./testdata/products.json")
	if err != nil {
		panic(err)
	}

	var order *Order
	if err := NewJsonConnector(&order, dataOrders).
		Where("ID", "=", 1).
		AddDependency("Client", dataClients).
		AddDependency("Product", dataProducts).
		Unmarshal(); err != nil {
		panic(err)
	}
	fmt.Println("order result:")
	fmt.Printf("%+v\n", order)
	fmt.Printf("%+v\n%+v\n", order.Product, order.Client)

	fmt.Println("--------")

	var client *Client
	if err := NewJsonConnector(&client, dataClients).
		Where("client_id", "=", 2).
		AddDependency("Orders", dataOrders).
		AddDependency("Orders.Product", dataProducts).
		Unmarshal(); err != nil {
		panic(err)
	}
	fmt.Println("client result:")
	fmt.Printf("%+v\n", client)
	for _, v := range client.Orders {
		fmt.Printf("%+v\n", v)
		fmt.Printf("\t%+v\n", v.Product)
	}
}
