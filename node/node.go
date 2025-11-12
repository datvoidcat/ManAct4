package node

import (
	"context"
	"log"
	"sync"
	"time"

	pb "mutex/proto"

	"google.golang.org/grpc"
)

type Node struct {
	ID              int32
	RequestTime     int64
	RequestingCS    bool
	InCS            bool
	DeferredReplies []int32
	ReplyCount      int
	Peers           map[int32]string // nodeID -> address
	Clients         map[int32]pb.MutexServiceClient
	mu              sync.Mutex
	pb.UnimplementedMutexServiceServer
}

// NewNode creates a new node with the given ID and peer addresses
func NewNode(id int32, peers map[int32]string) *Node {
	return &Node{
		ID:              id,
		RequestingCS:    false,
		InCS:            false,
		DeferredReplies: make([]int32, 0),
		Peers:           peers,
		Clients:         make(map[int32]pb.MutexServiceClient),
	}
}

// ConnectToPeers establishes gRPC connections to all peers
func (n *Node) ConnectToPeers() error {
	for id, addr := range n.Peers {
		if id == n.ID {
			continue
		}
		conn, err := grpc.Dial(addr, grpc.WithInsecure())
		if err != nil {
			return err
		}
		n.Clients[id] = pb.NewMutexServiceClient(conn)
	}
	return nil
}

// RequestAccess handles incoming request messages from other nodes
func (n *Node) RequestAccess(ctx context.Context, req *pb.RequestMessage) (*pb.ReplyMessage, error) {
	n.mu.Lock()
	defer n.mu.Unlock()

	log.Printf("Node %d received request from Node %d with timestamp %d", n.ID, req.NodeId, req.Timestamp)

	// If we're in CS or requesting it and have higher priority, defer reply
	if n.InCS || (n.RequestingCS && (n.RequestTime < req.Timestamp ||
		(n.RequestTime == req.Timestamp && n.ID < req.NodeId))) {
		n.DeferredReplies = append(n.DeferredReplies, req.NodeId)
		log.Printf("Node %d deferred reply to Node %d", n.ID, req.NodeId)
		return &pb.ReplyMessage{Ok: false}, nil
	}

	// Otherwise, reply immediately
	log.Printf("Node %d sending OK to Node %d", n.ID, req.NodeId)
	return &pb.ReplyMessage{Ok: true}, nil
}

// RequestCriticalSection initiates the process of requesting access to the critical section
func (n *Node) RequestCriticalSection() {
	n.mu.Lock()
	n.RequestingCS = true
	n.RequestTime = time.Now().UnixNano()
	n.ReplyCount = 0
	n.mu.Unlock()

	log.Printf("Node %d requesting critical section at time %d", n.ID, n.RequestTime)

	// Request permission from all other nodes
	for peerId, client := range n.Clients {
		go func(id int32, c pb.MutexServiceClient) {
			req := &pb.RequestMessage{
				NodeId:    n.ID,
				Timestamp: n.RequestTime,
			}
			reply, err := c.RequestAccess(context.Background(), req)
			if err != nil {
				log.Printf("Error requesting access from Node %d: %v", id, err)
				return
			}

			n.mu.Lock()
			if reply.Ok {
				n.ReplyCount++
				if n.ReplyCount == len(n.Peers)-1 {
					n.enterCriticalSection()
				}
			}
			n.mu.Unlock()
		}(peerId, client)
	}
}

// enterCriticalSection is called when a node has received permission from all peers
func (n *Node) enterCriticalSection() {
	n.InCS = true
	n.RequestingCS = false
	log.Printf("Node %d entering critical section", n.ID)

	// Simulate some work in the critical section
	go func() {
		time.Sleep(2 * time.Second)
		n.exitCriticalSection()
	}()
}

// exitCriticalSection is called when a node is done with the critical section
func (n *Node) exitCriticalSection() {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.InCS = false
	log.Printf("Node %d exiting critical section", n.ID)

	// Send deferred replies
	for _, nodeId := range n.DeferredReplies {
		if client, ok := n.Clients[nodeId]; ok {
			go func(id int32, c pb.MutexServiceClient) {
				req := &pb.RequestMessage{
					NodeId:    id,
					Timestamp: time.Now().UnixNano(),
				}
				_, err := c.RequestAccess(context.Background(), req)
				if err != nil {
					log.Printf("Error sending deferred reply to Node %d: %v", id, err)
				}
			}(nodeId, client)
		}
	}
	n.DeferredReplies = make([]int32, 0)
}
