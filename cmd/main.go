package main

import (
	"CodeRewievService/internal/bootstrap"
)

func main() {
	service := bootstrap.InitApplication()
	if err := service.Run(); err != nil {
		panic(err)
	}
}
