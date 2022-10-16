package main

import (
	"fmt"
	"os"

	budget "github.com/ryuichi1208/mackerel-errorbudget-calculator/lib"
)

func init() {
	if os.Getenv("MACKEREL_TOKEN") == "" {
		fmt.Println("Set environment variable MACKEREL_TOKEN")
		os.Exit(1)
	}
}

func main() {
	os.Exit(budget.Do())
}
