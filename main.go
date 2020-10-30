package main

import (
	"flag"
	"log"
	"net"
	"os"
	"strconv"

	pb "github.com/scayle/proto/go/user_service"

	"github.com/scayle/common-go"

	"google.golang.org/grpc"
)

func main() {
	runGrpc()
}

func secret() string {
	secret := flag.String("jwt-secret", "", "the jwt secret to be used, can also be provided using the environment variable 'JWT_SECRET'")

	flag.Parse()

	if *secret == "" {
		envSecret := os.Getenv("JWT_SECRET")
		secret = &envSecret
	}

	if *secret == "" {
		log.Fatal("empty secret")
	}

	return *secret
}

func runGrpc() {
	registration := common.RegisterConsulService(
		"user-service",
		common.WithDefaultPort(8100),
		common.WithHTTPHealthCheck(8101))

	listener, err := net.Listen("tcp", ":"+strconv.Itoa(registration.Port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	srv := grpc.NewServer()
	pb.RegisterUserServiceServer(srv, &handler{
		repo: &inMemoryRepository{},
		auth: &jwtAuthenticator{[]byte(secret())},
	})

	log.Println("setup finished - starting service")
	if e := srv.Serve(listener); e != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
