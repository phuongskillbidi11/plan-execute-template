// Command cli wires the api/db/cache/auth/utils packages together.
package main

import (
	"fmt"

	"largecontextbench/api"
	"largecontextbench/auth"
)

func main() {
	fmt.Println(api.Handle("/status"))
	fmt.Println("token valid:", auth.ValidToken(""))
}
