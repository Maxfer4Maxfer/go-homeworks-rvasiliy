package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"reflect"
	"strings"
	"sync"
	"time"

	"../protobuf"
	"github.com/maxfer4maxfer/goDebuger"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
)

const (
	// debug switch option
	DEBUG = false

	// какой адрес-порт слушать серверу
	listenAddr string = "127.0.0.1:8082"
)

type logMessage struct {
	Timestamp int64
	Consumer  string
	Method    string
	Host      string
}

type statMessage struct {
	Consumer string
	Method   string
}

type logReciver struct {
	log   chan *logMessage
	close chan struct{}
}

type statReciver struct {
	stat  chan *statMessage
	close chan struct{}
}

// --------------GRPCServer--------------
type GRPCServer struct {
	mutex         *sync.RWMutex
	grpcServer    *grpc.Server
	acl           *ACLManager
	loggers       []*logReciver
	statGatherers []*statReciver
}

func NewGRPCServer(aclm *ACLManager) *GRPCServer {
	return &GRPCServer{
		mutex:         &sync.RWMutex{},
		acl:           aclm,
		loggers:       make([]*logReciver, 0),
		statGatherers: make([]*statReciver, 0),
	}
}

func (s *GRPCServer) sendStatMessage(consumer string, method string) {
	s.mutex.RLock()
	for _, sg := range s.statGatherers {
		go func(sg *statReciver) {
			statMessage := &statMessage{
				Consumer: consumer,
				Method:   method,
			}
			sg.stat <- statMessage
		}(sg)
	}
	s.mutex.RUnlock()
}

func (s *GRPCServer) sendLogMessage(consumer string, method string, host string) {
	s.mutex.RLock()
	for _, l := range s.loggers {
		go func(l *logReciver) {
			logMessage := &logMessage{
				Timestamp: time.Now().UnixNano(),
				Consumer:  consumer,
				Method:    method,
				Host:      host,
			}
			l.log <- logMessage
		}(l)
	}
	s.mutex.RUnlock()
}

func (s *GRPCServer) streamInterceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	if DEBUG {
		fmt.Println(goDebuger.GetCurrentFunctionName())
	}

	start := time.Now()

	md, _ := metadata.FromIncomingContext(ss.Context())
	err := s.acl.checkACL(md, info.FullMethod)
	if err != nil {
		return err
	}

	// send a log message
	s.sendLogMessage(md["consumer"][0], info.FullMethod, "127.0.0.1:")
	s.sendStatMessage(md["consumer"][0], info.FullMethod)

	err = handler(srv, ss) // Request a GRPC function

	if DEBUG {
		fmt.Printf(`--
			after incoming call=%v
			srv=%#v
			ServerStream=%#v
			md=%v
			time=%v
			err=%v
		`, info.FullMethod, srv, ss, md, time.Since(start), err)
	}
	return nil
}

func (s *GRPCServer) unaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	if DEBUG {
		fmt.Println(goDebuger.GetCurrentFunctionName())
	}

	start := time.Now()

	md, _ := metadata.FromIncomingContext(ctx)
	err := s.acl.checkACL(md, info.FullMethod)
	if err != nil {
		return nil, err
	}

	// send a log message
	s.sendLogMessage(md["consumer"][0], info.FullMethod, "127.0.0.1:")
	s.sendStatMessage(md["consumer"][0], info.FullMethod)

	reply, err := handler(ctx, req) // Request a GRPC function
	if DEBUG {
		fmt.Printf(`--
			after incoming call=%v
			req=%#v
			reply=%#v
			md=%v
			time=%v
			err=%v
		`, info.FullMethod, req, reply, md, time.Since(start), err)
	}
	return reply, err

}

func (s *GRPCServer) start(ctx context.Context, listenAddr string) {
	lis, err := net.Listen("tcp", ":8082")
	if err != nil {
		log.Fatalln("cant listet port", err)
	}

	s.grpcServer = grpc.NewServer(
		grpc.UnaryInterceptor(s.unaryInterceptor),
		grpc.StreamInterceptor(s.streamInterceptor),
	)

	// fmt.Println("--> starting server at :8082")

	protobuf.RegisterAdminServer(s.grpcServer, NewAdminServer(&s.loggers, &s.statGatherers, s.mutex))
	protobuf.RegisterBizServer(s.grpcServer, NewBizServer())

	go s.grpcServer.Serve(lis)

	select {
	case <-ctx.Done():
		for _, l := range s.loggers {
			l.close <- struct{}{}
		}
		for _, sg := range s.statGatherers {
			sg.close <- struct{}{}
		}
		s.grpcServer.Stop()
	}
}

// --------------ACLManager--------------
type ACLManager struct {
	ACL map[string][]string
}

func NewACLManager(ACLData string) (*ACLManager, error) {
	ACLManagerInstance := &ACLManager{
		ACL: make(map[string][]string),
	}

	err := json.Unmarshal([]byte(ACLData), &ACLManagerInstance.ACL)
	if err != nil {
		return nil, fmt.Errorf("expacted error on bad acl json, have nil")
	}

	return ACLManagerInstance, nil
}

func (acl *ACLManager) checkACL(md metadata.MD, reqMethod string) error {

	if len(md["consumer"]) == 0 {
		return grpc.Errorf(codes.Unauthenticated, "ACL consumer metadata variable is not set")
	}

	consumer := md["consumer"][0]

	userACL, ok := acl.ACL[consumer]
	if !ok {
		return grpc.Errorf(codes.Unauthenticated, "Specified user %v does not exist", consumer)
	}
	for _, method := range userACL {
		methodSlice := strings.Split(method, "/")
		reqMethodSlice := strings.Split(reqMethod, "/")
		if reflect.DeepEqual(methodSlice, reqMethodSlice) {
			return nil
		}
		if methodSlice[1] == reqMethodSlice[1] && methodSlice[2] == "*" {
			return nil
		}
	}
	return grpc.Errorf(codes.Unauthenticated,
		"Consumer %v is not authenticated to call %v",
		md["consumer"][0],
		reqMethod,
	)
}

func StartMyMicroservice(ctx context.Context, listenAddr string, ACLData string) error {

	ACLManager, err := NewACLManager(ACLData)
	if err != nil {
		return err
	}

	GPRCserver := NewGRPCServer(ACLManager)

	go GPRCserver.start(ctx, listenAddr)

	return nil
}

func main() {
	ACLData := `{
	"logger":    ["/main.Admin/Logging"],
	"stat":      ["/main.Admin/Statistics"],
	"biz_user":  ["/main.Biz/Check", "/main.Biz/Add"],
	"biz_admin": ["/main.Biz/*"]
}`
	StartMyMicroservice(context.Background(), listenAddr, ACLData)
}
