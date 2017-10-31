# gluasocket


## example
```go
package main

import (
	"fmt"
	"time"

	"github.com/Greyh4t/dnscache"
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
	resolver := dnscache.New(time.Minute * 10)
	//L.PreloadModule("socket", gluasocket.NewSocketModule(nil).Loader)
	L.PreloadModule("socket", gluasocket.NewSocketModule(resolver).Loader)
	err := L.DoString(
		`socket=require("socket")
		s=socket.new("tcp")
		s:settimeout(5)
		err = s:connect("www.example.com",80)
		if err==nil
		then
			s:send("GET /\r\n\r\n")
			x=s:readn(40)
			y=s:read()
			print(x)
			print("---------------")
			print(y)
			s:close()
		else
			print(err)
		end`,
	)
	if err != nil {
		fmt.Println(err)
	}
}
```