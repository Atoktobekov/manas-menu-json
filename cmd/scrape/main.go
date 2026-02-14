package main

import (
	"fmt"
	"log"
)

func main() {
	kantin, err := scrapeKantin()
	if err != nil {
		log.Fatal(err)
	}
	buffet, err := scrapeBuffet1()
	if err != nil {
		log.Fatal(err)
	}

	if err := writeJSON("manas_kantin.json", kantin); err != nil {
		log.Fatal(err)
	}
	if err := writeJSON("buffet_1.json", buffet); err != nil {
		log.Fatal(err)
	}

	fmt.Println("OK: wrote manas_kantin.json and buffet_1.json")
}
