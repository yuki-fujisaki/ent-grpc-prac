package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"

	"github.com/go-sql-driver/mysql"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"ent-grpc-prac/ent"
	gw "ent-grpc-prac/gw"
	entgrpcpracpb "ent-grpc-prac/pkg/protos"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	"google.golang.org/grpc/credentials/insecure"
)

func startHTTPServer(grpcPort, httpPort int) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	err := gw.RegisterUserServiceHandlerFromEndpoint(ctx, mux, fmt.Sprintf("localhost:%d", grpcPort), opts)
	if err != nil {
		log.Fatalf("Failed to start HTTP gateway: %v", err)
	}

	log.Printf("start HTTP server port: %v", httpPort)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", httpPort), mux); err != nil {
		log.Fatalf("Failed to serve HTTP server: %v", err)
	}
}

type myServer struct {
	entgrpcpracpb.UnimplementedUserServiceServer
	UserService *UserService
}

func (s *myServer) GetAllUsers(ctx context.Context, req *entgrpcpracpb.GetAllUsersRequest) (*entgrpcpracpb.GetAllUsersResponse, error) {
	users, err := s.UserService.ent.User.Query().All(ctx)
	if err != nil {
		return nil, err
	}

	res := &entgrpcpracpb.GetAllUsersResponse{}
	for _, user := range users {
		res.Users = append(res.Users, &entgrpcpracpb.User{
			Id:   int32(user.ID),
			Name: user.Name,
		})
	}

	return res, nil
}

func NewMyServer(mc mysql.Config, entOptions ...ent.Option) *myServer {
	client, err := ent.Open("mysql", mc.FormatDSN(), entOptions...)
	if err != nil {
		log.Fatalf("Error open mysql ent client: %v\n", err)
	}
	return &myServer{
		UserService: &UserService{
			ent: client,
		},
	}
}

type (
	UserService struct {
		ent *ent.Client
	}
)

func NewUserService(ent *ent.Client) *UserService {
	return &UserService{
		ent: ent,
	}
}

func main() {
	entOptions := []ent.Option{}

	entOptions = append(entOptions, ent.Debug())

	mc := mysql.Config{
		User:                 "user",
		DBName:               "ent-grpc-prac-mysql",
		Passwd:               "password",
		Net:                  "tcp",
		Addr:                 "localhost" + ":" + "3333",
		AllowNativePasswords: true,
		ParseTime:            true,
	}

	client, err := ent.Open("mysql", mc.FormatDSN(), entOptions...)
	if err != nil {
		log.Fatalf("Error open mysql ent client: %v\n", err)
	}

	defer client.Close()

	if err := client.Schema.Create(context.Background()); err != nil {
		log.Fatalf("failed creating schema resources: %v", err)
	}

	grpcPort := 8080
	httpPort := 8081 // HTTPサーバー用のポートを設定

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", grpcPort))
	if err != nil {
		panic(err)
	}

	s := grpc.NewServer()
	entgrpcpracpb.RegisterUserServiceServer(s, NewMyServer(mc))
	reflection.Register(s)

	go func() {
		log.Printf("start gRPC server port: %v", grpcPort)
		if err := s.Serve(listener); err != nil {
			log.Fatalf("Failed to serve gRPC server: %v", err)
		}
	}()

	go startHTTPServer(grpcPort, httpPort) // HTTPサーバーを非同期で起動

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	log.Println("stopping gRPC server...")
	s.GracefulStop()
}
