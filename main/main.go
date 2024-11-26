package main

import (
	"ethpruner/utils"
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Error! Must indicate the option: $go run main/main.go [prune/query]")
		return
	}

	switch os.Args[1] {
	case "prune":
		utils.DoPrune(os.Args[2:])
	case "query":
		utils.DoQuery(os.Args[2:])
	default:
		fmt.Println("Error! Must indicate the option: $go run main/main.go [prune/query]")
	}
}
