package main

import (
	"fmt"
	"regexp"
)

func main() {
	var illegalProfileNameCharacters = regexp.MustCompile(`[\\[\];'" ]`)

	fmt.Println(illegalProfileNameCharacters.ReplaceAllString("hello[]\\;'", "-"))
}
