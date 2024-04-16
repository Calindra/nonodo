package proxygraphql

import (
	"fmt"
	_ "log"
	_ "net/http"

	_ "github.com/99designs/gqlgen/handler"
)

func Test(a int, b int) int {
	c := a + b
	fmt.Println("test")
	return c
}

func Query() {
}
