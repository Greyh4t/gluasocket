package gluasocket

import (
	"io/ioutil"
	"net"
	"strconv"
	"time"

	"github.com/yuin/gopher-lua"
)

type Socket struct {
	t       string
	timeout time.Duration
	conn    net.Conn
}

func Loader(L *lua.LState) int {
	socket := L.NewTypeMetatable("socket")
	L.SetGlobal("socket", socket)
	L.SetField(socket, "new", L.NewFunction(newSocket))
	L.SetField(socket, "__index", L.SetFuncs(L.NewTable(), map[string]lua.LGFunction{
		"settimeout": settimeout,
		"connect":    connect,
		"send":       send,
		"read":       read,
		"readn":      readN,
		"close":      close,
	}))
	L.Push(socket)
	return 1
}

func newSocket(L *lua.LState) int {
	t := L.CheckString(1)
	if t != "tcp" && t != "udp" {
		L.ArgError(1, "type must be tcp or udp")
	}
	ud := L.NewUserData()
	ud.Value = &Socket{
		t:       t,
		timeout: 30 * time.Second,
	}
	L.SetMetatable(ud, L.GetTypeMetatable("socket"))
	L.Push(ud)
	return 1
}

func checkSocket(L *lua.LState) *Socket {
	ud := L.CheckUserData(1)
	if v, ok := ud.Value.(*Socket); ok {
		return v
	}
	L.ArgError(1, "socket.socket expected")
	return nil
}

func settimeout(L *lua.LState) int {
	s := checkSocket(L)
	s.timeout = time.Duration(L.CheckInt(2)) * time.Second
	return 0
}

func connect(L *lua.LState) int {
	s := checkSocket(L)
	addr := L.CheckString(2) + ":" + strconv.Itoa(L.CheckInt(3))
	conn, err := net.DialTimeout(s.t, addr, s.timeout)
	if err != nil {
		L.Push(lua.LString(err.Error()))
		return 1
	}
	s.conn = conn
	return 0
}

func send(L *lua.LState) int {
	s := checkSocket(L)
	s.conn.SetWriteDeadline(time.Now().Add(s.timeout))
	n, err := s.conn.Write([]byte(L.CheckString(2)))
	s.conn.SetWriteDeadline(time.Time{})
	L.Push(lua.LNumber(n))
	if err != nil {
		L.Push(lua.LString(err.Error()))
		return 2
	}
	return 1
}

func read(L *lua.LState) int {
	s := checkSocket(L)
	s.conn.SetReadDeadline(time.Now().Add(s.timeout))
	buf, err := ioutil.ReadAll(s.conn)
	L.Push(lua.LString(string(buf)))
	if err != nil {
		L.Push(lua.LString(err.Error()))
		return 2
	}
	return 1
}

func readN(L *lua.LState) int {
	s := checkSocket(L)
	l := L.CheckInt(2)
	buf := make([]byte, l)
	s.conn.SetReadDeadline(time.Now().Add(s.timeout))
	n, err := s.conn.Read(buf)
	L.Push(lua.LString(string(buf[:n])))
	if err != nil {
		L.Push(lua.LString(err.Error()))
		return 2
	}
	return 1
}

func close(L *lua.LState) int {
	s := checkSocket(L)
	s.conn.Close()
	return 0
}
