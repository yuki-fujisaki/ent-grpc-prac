package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"

	"github.com/go-sql-driver/mysql"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"ent-grpc-prac/ent"
	entgrpcpracpb "ent-grpc-prac/pkg/protos"
)

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

	port := 8080
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		panic(err)
	}

	s := grpc.NewServer()

	entgrpcpracpb.RegisterUserServiceServer(s, NewMyServer(mc))

	reflection.Register(s)

	go func() {
		log.Printf("start gRPC server port: %v", port)
		s.Serve(listener)
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	log.Println("stopping gRPC server...")
	s.GracefulStop()
}
