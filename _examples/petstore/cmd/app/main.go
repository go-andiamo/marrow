package main

import "petstore/api"

func main() {
	a := api.NewApi()
	a.Start(8080)
}
