package gluasocket

import (
	"io/ioutil"
	"net"
	"strconv"
	"time"

	"github.com/Greyh4t/dnscache"
	"github.com/yuin/gopher-lua"
)

type socketModule struct {
	resolver *dnscache.Resolver
}

type Socket struct {
	t       string
	timeout time.Duration
	conn    net.Conn
}

func NewSocketModule(resolver *dnscache.Resolver) *socketModule {
	return &socketModule{
		resolver: resolver,
	}
}

func (self *socketModule) Loader(L *lua.LState) int {
	socket := L.SetFuncs(L.NewTable(), map[string]lua.LGFunction{
		"new": self.newSocket,
	})

	mt := L.NewTypeMetatable("socket")
	L.SetField(mt, "__index", L.SetFuncs(L.NewTable(), map[string]lua.LGFunction{
		"settimeout": self.settimeout,
		"connect":    self.connect,
		"send":       self.send,
		"read":       self.read,
		"readn":      self.readN,
		"close":      self.close,
	}))
	L.SetField(socket, "socket", mt)

	L.Push(socket)
	return 1
}

func (self *socketModule) AsyncLoader(L *lua.LState) int {
	socket := L.SetFuncs(L.NewTable(), map[string]lua.LGFunction{
		"new": self.newSocket,
	})

	mt := L.NewTypeMetatable("socket")
	L.SetField(mt, "__index", L.SetFuncs(L.NewTable(), map[string]lua.LGFunction{
		"settimeout": self.settimeout,
		"connect":    self.asyncConnect,
		"send":       self.asyncSend,
		"read":       self.asyncRead,
		"readn":      self.asyncReadN,
		"close":      self.close,
	}))
	L.SetField(socket, "socket", mt)

	L.Push(socket)
	return 1
}

func (self *socketModule) newSocket(L *lua.LState) int {
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

func (self *socketModule) checkSocket(L *lua.LState) *Socket {
	ud := L.CheckUserData(1)
	if v, ok := ud.Value.(*Socket); ok {
		return v
	}
	L.ArgError(1, "socket.socket expected")
	return nil
}

func (self *socketModule) settimeout(L *lua.LState) int {
	s := self.checkSocket(L)
	s.timeout = time.Duration(L.CheckInt(2)) * time.Second
	return 0
}

func (self *socketModule) close(L *lua.LState) int {
	s := self.checkSocket(L)
	s.conn.Close()
	return 0
}

//sync
func (self *socketModule) connect(L *lua.LState) int {
	s := self.checkSocket(L)
	host := L.CheckString(2)
	port := strconv.Itoa(L.CheckInt(3))

	if self.resolver != nil {
		var err error
		host, err = self.resolver.FetchOneString(host)
		if err != nil {
			L.Push(lua.LString(err.Error()))
			return 1
		}
	}

	conn, err := net.DialTimeout(s.t, net.JoinHostPort(host, port), s.timeout)
	if err != nil {
		L.Push(lua.LString(err.Error()))
		return 1
	}
	s.conn = conn
	return 0
}

func (self *socketModule) send(L *lua.LState) int {
	s := self.checkSocket(L)
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

func (self *socketModule) read(L *lua.LState) int {
	s := self.checkSocket(L)
	s.conn.SetReadDeadline(time.Now().Add(s.timeout))
	buf, err := ioutil.ReadAll(s.conn)
	s.conn.SetReadDeadline(time.Time{})
	L.Push(lua.LString(string(buf)))
	if err != nil {
		L.Push(lua.LString(err.Error()))
		return 2
	}
	return 1
}

func (self *socketModule) readN(L *lua.LState) int {
	s := self.checkSocket(L)
	l := L.CheckInt(2)
	buf := make([]byte, l)
	s.conn.SetReadDeadline(time.Now().Add(s.timeout))
	n, err := s.conn.Read(buf)
	s.conn.SetReadDeadline(time.Time{})
	L.Push(lua.LString(string(buf[:n])))
	if err != nil {
		L.Push(lua.LString(err.Error()))
		return 2
	}
	return 1
}

//async
func (self *socketModule) asyncConnect(L *lua.LState) int {
	s := self.checkSocket(L)
	host := L.CheckString(2)
	port := strconv.Itoa(L.CheckInt(3))
	resultChan := make(chan lua.LValue, 1)

	go func(s *Socket, host, port string) {
		if self.resolver != nil {
			var err error
			host, err = self.resolver.FetchOneString(host)
			if err != nil {
				resultChan <- lua.LString(err.Error())
				close(resultChan)
				return
			}
		}

		conn, err := net.DialTimeout(s.t, net.JoinHostPort(host, port), s.timeout)
		if err != nil {
			resultChan <- lua.LString(err.Error())
			close(resultChan)
			return
		}
		s.conn = conn
		close(resultChan)
	}(s, host, port)

	return L.Yield(lua.LChannel(resultChan))
}

func (self *socketModule) asyncSend(L *lua.LState) int {
	s := self.checkSocket(L)
	resultChan := make(chan lua.LValue, 2)

	go func(s *Socket, resultChan chan lua.LValue) {
		s.conn.SetWriteDeadline(time.Now().Add(s.timeout))
		n, err := s.conn.Write([]byte(L.CheckString(2)))
		s.conn.SetWriteDeadline(time.Time{})
		resultChan <- lua.LNumber(n)
		if err != nil {
			resultChan <- lua.LString(err.Error())
		}
		close(resultChan)
	}(s, resultChan)

	return L.Yield(lua.LChannel(resultChan))
}

func (self *socketModule) asyncRead(L *lua.LState) int {
	s := self.checkSocket(L)
	resultChan := make(chan lua.LValue, 2)

	go func(s *Socket, resultChan chan lua.LValue) {
		s.conn.SetReadDeadline(time.Now().Add(s.timeout))
		buf, err := ioutil.ReadAll(s.conn)
		s.conn.SetReadDeadline(time.Time{})
		resultChan <- lua.LString(string(buf))
		if err != nil {
			resultChan <- lua.LString(err.Error())
		}
		close(resultChan)
	}(s, resultChan)

	return L.Yield(lua.LChannel(resultChan))
}

func (self *socketModule) asyncReadN(L *lua.LState) int {
	s := self.checkSocket(L)
	l := L.CheckInt(2)
	resultChan := make(chan lua.LValue, 2)

	go func(s *Socket, l int, resultChan chan lua.LValue) {
		buf := make([]byte, l)
		s.conn.SetReadDeadline(time.Now().Add(s.timeout))
		n, err := s.conn.Read(buf)
		s.conn.SetReadDeadline(time.Time{})
		resultChan <- lua.LString(string(buf[:n]))
		if err != nil {
			resultChan <- lua.LString(err.Error())
		}
		close(resultChan)
	}(s, l, resultChan)

	return L.Yield(lua.LChannel(resultChan))
}
