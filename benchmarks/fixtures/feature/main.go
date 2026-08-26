package main

import (
	"fmt"
	"os"
)

type Record struct {
	ID   int
	Name string
}

var records = []Record{
	{1, "alpha"},
	{2, "beta"},
	{3, "gamma"},
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: recordstore <list|add>")
		os.Exit(1)
	}
	switch os.Args[1] {
	case "list":
		for _, r := range records {
			fmt.Printf("%d\t%s\n", r.ID, r.Name)
		}
	case "add":
		if len(os.Args) < 3 {
			fmt.Println("usage: recordstore add <name>")
			os.Exit(1)
		}
		records = append(records, Record{len(records) + 1, os.Args[2]})
		fmt.Println("added:", os.Args[2])
	default:
		fmt.Println("unknown command:", os.Args[1])
		os.Exit(1)
	}
}
