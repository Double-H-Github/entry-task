// @Author: 2014BDuck
// @Date: 2021/8/5

package erpc

import (
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"net"
	"reflect"
	"runtime/debug"
	"sync"
)

type Server struct {
	addr  string
	funcs map[string]reflect.Value
}

func NewServer(addr string) *Server {
	return &Server{addr: addr, funcs: make(map[string]reflect.Value)}
}

func (s *Server) Run() {
	l, err := net.Listen("tcp", s.addr)
	if err != nil {
		log.Printf("listen on %s err: %v\n", s.addr, err)
		return
	}
	for {
		conn, err := l.Accept()
		if err != nil {
			log.Printf("accept err: %v\n", err)
			continue
		}

		go func() {
			srvTransport := NewTransport(conn, &sync.Mutex{})
			defer func() {
				if e := recover(); e != nil {
					//_ = conn.Close()
					recoveryLog := "recovery log:  message: %v, stack: %s"
					log.Printf(recoveryLog, e, string(debug.Stack()[:]))
				}
			}()
			for {
				// read request from client
				req, err := srvTransport.Receive()
				if err != nil {
					if err != io.EOF {
						log.Printf("read err: %v\n", err)
					}
					return
				}
				// get method by name
				f, ok := s.funcs[req.Name]
				if !ok { // if method requested does not exist
					e := fmt.Sprintf("func %s does not exist", req.Name)
					log.Println(e)
					if err = srvTransport.Send(Data{Name: req.Name, Err: e}); err != nil {
						log.Printf("transport write err: %v\n", err)
					}
					continue
				}
				log.Printf("func %s is called\n", req.Name)
				// unpackage request arguments
				inArgs := make([]reflect.Value, len(req.Args))
				for i := range req.Args {
					inArgs[i] = reflect.ValueOf(req.Args[i])
				}
				// invoke requested method
				out := f.Call(inArgs)
				// package response arguments (except error)
				outArgs := make([]interface{}, len(out)-1)
				for i := 0; i < len(out)-1; i++ {
					//if out[i].IsNil(){
					//	outArgs[i] = reflect.Zero(f.Type().Out(i))
					//}else{
					//	outArgs[i] = out[i].Interface()
					//}
					outArgs[i] = out[i].Interface()
				}
				// package error argument
				var e string
				if _, ok := out[len(out)-1].Interface().(error); !ok {
					e = ""
				} else {
					e = out[len(out)-1].Interface().(error).Error()
				}
				// send response to client
				err = srvTransport.Send(Data{Name: req.Name, Args: outArgs, Err: e})
				if err != nil {
					log.Printf("transport write err: %v\n", err)
				}
			}
		}()
	}
}

func (s *Server) Register(name string, f interface{}, reqStruct, respStruct interface{}) {
	if _, ok := s.funcs[name]; ok {
		panic(fmt.Sprintf("erpc.Server.Register: service existed: %v", name))
	}
	gob.Register(reqStruct)
	gob.Register(respStruct)
	s.funcs[name] = reflect.ValueOf(f)
}
