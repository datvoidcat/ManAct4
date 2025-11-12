package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"mutex/node"
	pb "mutex/proto"

	"google.golang.org/grpc"
)

//run: go run main.go -id 1 -port 50051 -peers "localhost:50051,localhost:50052,localhost:50053"
// and: go run main.go -id 2 -port 50052 -peers "localhost:50051,localhost:50052,localhost:50053"
// and: go run main.go -id 3 -port 50053 -peers "localhost:50051,localhost:50052,localhost:50053"

func main() {
	// Command line flags
	nodeID := flag.Int("id", 1, "Node ID")
	port := flag.Int("port", 50051, "Port to listen on")
	peersStr := flag.String("peers", "", "Comma-separated list of peer addresses (e.g., 'localhost:50051,localhost:50052')")
	flag.Parse()

	// Parse peers
	peerAddrs := strings.Split(*peersStr, ",")
	peers := make(map[int32]string)
	for i, addr := range peerAddrs {
		if addr != "" {
			peers[int32(i+1)] = addr
		}
	}

	// Create node
	n := node.NewNode(int32(*nodeID), peers)

	// Start gRPC server
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	server := grpc.NewServer()
	pb.RegisterMutexServiceServer(server, n)

	go func() {
		if err := server.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	// Connect to peers
	log.Printf("Node %d connecting to peers...", *nodeID)
	if err := n.ConnectToPeers(); err != nil {
		log.Fatalf("failed to connect to peers: %v", err)
	}
	log.Printf("Node %d connected to peers successfully", *nodeID)

	// Periodically request critical section
	go func() {
		for {
			time.Sleep(2 * time.Second)
			n.RequestCriticalSection()
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	// Cleanup
	server.GracefulStop()
}
