package main

import (
	"log"

	"github.com/joho/godotenv"
	"github.com/nikita-simankov/upstore/services/users/internal/config"
)

func init() {
	if err := godotenv.Load(); err != nil {
		log.Fatal(err)
	}
}

func main() {
	c := config.Load()
	log.Printf("%+v", c)
}
