package main

import (
	"fmt"
	"os"

	budget "github.com/ryuichi1208/mackerel-errorbudget-calculator/lib"
)

func main() {
	fmt.Println("exit")
	os.Exit(budget.Do())
}
