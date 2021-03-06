package sip

import (
	"fmt"
	"net"
	"net/http"
	"sync"

	"github.com/panjjo/gosip/utils"
	"github.com/sirupsen/logrus"
)

var (
	bufferSize uint16 = 65535 - 20 - 8 // IPv4 max size - IPv4 Header size - UDP Header size
)

// RequestHandler RequestHandler
type RequestHandler func(req *Request, tx *Transaction)

// Server Server
type Server struct {
	udpaddr         net.Addr
	conn            Connection
	txs             *transacionts
	hmu             *sync.RWMutex
	requestHandlers map[RequestMethod]RequestHandler
	port            *Port
	host            net.IP
}

// NewServer NewServer
func NewServer() *Server {
	activeTX = &transacionts{txs: map[string]*Transaction{}, rwm: &sync.RWMutex{}}
	srv := &Server{
		hmu:             &sync.RWMutex{},
		txs:             activeTX,
		requestHandlers: map[RequestMethod]RequestHandler{},
	}
	return srv
}

func (s *Server) newTX(key string) *Transaction {
	return s.txs.newTX(key, s.conn)
}
func (s *Server) getTX(key string) *Transaction {
	return s.txs.getTX(key)
}
func (s *Server) mustTX(key string) *Transaction {
	tx := s.txs.getTX(key)
	if tx == nil {
		tx = s.txs.newTX(key, s.conn)
	}
	return tx
}

// ListenUDPServer ListenUDPServer
func (s *Server) ListenUDPServer(addr string) {
	udpaddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		logrus.Fatal("net.ResolveUDPAddr err", err, addr)
	}
	s.udpaddr = udpaddr
	s.port = NewPort(udpaddr.Port)
	s.host, err = utils.ResolveSelfIP()
	if err != nil {
		logrus.Fatal("net.ListenUDP resolveip err", err, addr)
	}
	udp, err := net.ListenUDP("udp", udpaddr)
	if err != nil {
		logrus.Fatal("net.ListenUDP err", err, addr)
	}
	s.conn = newUDPConnection(udp)
	var (
		raddr net.Addr
		num   int
	)
	buf := make([]byte, bufferSize)
	parser := newParser()
	defer parser.stop()
	go s.handlerListen(parser.out)
	for {
		num, raddr, err = s.conn.ReadFrom(buf)
		// fmt.Println("11111111111111111111111111111111111111111111111111111")
		// fmt.Println(num)
		// fmt.Println(string(buf[:num]))
		// fmt.Println("22222222222222222222222222222222222222222222222222222")
		if err != nil {
			logrus.Errorln("udp.ReadFromUDP err", err)
			continue
		}
		parser.in <- newPacket(buf[:num], raddr)
	}
}

// RegistHandler RegistHandler
func (s *Server) RegistHandler(method RequestMethod, handler RequestHandler) {
	s.hmu.Lock()
	s.requestHandlers[method] = handler
	s.hmu.Unlock()
}

func (s *Server) handlerListen(msgs chan Message) {
	var msg Message
	for {
		msg = <-msgs
		switch msg.(type) {
		case *Request:
			req := msg.(*Request)
			req.SetDestination(s.udpaddr)
			s.handlerRequest(req)
		case *Response:
			resp := msg.(*Response)
			resp.SetDestination(s.udpaddr)
			s.handlerResponse(resp)
		}
	}
}

func (s *Server) handlerRequest(msg *Request) {
	tx := s.mustTX(getTXKey(msg))
	logrus.Traceln("receive request from:", msg.Source(), ",method:", msg.Method(), "txKey:", tx.key, "message: \n", msg.String())
	s.hmu.RLock()
	handler, ok := s.requestHandlers[msg.Method()]
	s.hmu.RUnlock()
	if !ok {
		logrus.Errorln("not found handler func,requestMethod:", msg.Method(), msg.String())
		go handlerMethodNotAllowed(msg, tx)
		return
	}

	go handler(msg, tx)
}

func (s *Server) handlerResponse(msg *Response) {
	// fmt.Println("response: ", msg)
	tx := s.getTX(getTXKey(msg))
	if tx != nil {
		tx.receiveResponse(msg)
		logrus.Traceln("receive response from:", msg.Source(), "txKey:", tx.key, "message: \n", msg.String())
	} else {
		logrus.Infoln("not found tx. receive response from:", msg.Source(), "message: \n", msg.String())
	}
}

// Request Request
func (s *Server) Request(req *Request) (*Transaction, error) {
	viaHop, ok := req.ViaHop()
	if !ok {
		return nil, fmt.Errorf("missing required 'Via' header")
	}
	viaHop.Host = s.host.String()
	viaHop.Port = s.port
	if viaHop.Params == nil {
		viaHop.Params = NewParams().Add("branch", String{Str: GenerateBranch()})
	}
	if !viaHop.Params.Has("rport") {
		viaHop.Params.Add("rport", nil)
	}
	// test
	// if !viaHop.Params.Has("rport") {
	// 	viaHop.Params.Add("rport", String{Str: strconv.Itoa(int(*viaHop.Port))})
	// }
	// if !viaHop.Params.Has("received") {
	// 	viaHop.Params.Add("received", String{Str: viaHop.Host})
	// }
	//

	tx := s.mustTX(getTXKey(req))
	return tx, tx.Request(req)
}

func handlerMethodNotAllowed(req *Request, tx *Transaction) {
	resp := NewResponseFromRequest("", req, http.StatusMethodNotAllowed, http.StatusText(http.StatusMethodNotAllowed), "")
	tx.Response(resp)
}
