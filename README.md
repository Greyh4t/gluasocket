# gluasocket


## example
```go
package main

import (
	"fmt"

	"github.com/Greyh4t/gluasocket"
	"github.com/yuin/gopher-lua"
)

func main() {
	L := lua.NewState(
		lua.Options{
			CallStackSize: 512,
			RegistrySize:  512,
		},
	)
	L.PreloadModule("socket", gluasocket.Loader)
	err := L.DoString(
		`socket=require("socket")
		s=socket.new("tcp")
		s:settimeout(5)
		ok, err = s:connect("www.example.com",80)
		if ok
		then
			s:send("GET /\r\n\r\n")
			x=s:read()
			print(x)
			s:close()
		else
			print(err)
		end
		`,
	)
	if err != nil {
		fmt.Println(err)
	}
}
```
