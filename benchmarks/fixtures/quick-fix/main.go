package main

import "fmt"

// ReconnectTimeoutMS is how long the client waits before retrying a dropped connection.
const ReconnectTimeoutMS = 1000

func main() {
	fmt.Println("reconnect timeout:", ReconnectTimeoutMS, "ms")
}
