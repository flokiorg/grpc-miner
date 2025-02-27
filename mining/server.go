// Copyright (c) 2024 The Flokicoin developers
// Distributed under the MIT software license, see the accompanying
// file COPYING or http://www.opensource.org/licenses/mit-license.php.

package mining

import (
	"context"
	"encoding/hex"
	"fmt"
	"net"
	"sync"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"

	"github.com/flokiorg/grpc-miner/mining/pb"
)

const (
	SERVER_PORT = 5055
)

type ServerCallbacks struct {
	SubmitBlocks     func(*pb.ValidBlock) ([]byte, error)
	NewCandidateBock func(*pb.CandidateRequest) (*pb.CandidateBlock, error)
	GenerateBlocks   func(uint32) ([]string, error)
}

type Server struct {
	pb.UnimplementedCandidateStreamServer
	clients           map[string]pb.CandidateStream_OpenServer
	clientsLock       sync.Mutex
	grpc              *grpc.Server
	callbacks         *ServerCallbacks
	lastHeaders       map[string]*pb.CandidateBlock
	candidateRequests map[string]*pb.CandidateRequest
}

func NewServer(callbacks *ServerCallbacks) *Server {
	return &Server{
		clients:           make(map[string]pb.CandidateStream_OpenServer),
		lastHeaders:       map[string]*pb.CandidateBlock{},
		candidateRequests: map[string]*pb.CandidateRequest{},
		callbacks:         callbacks,
	}
}

func (s *Server) Open(req *pb.CandidateRequest, stream pb.CandidateStream_OpenServer) error {

	clientID := fmt.Sprintf("%p", stream)
	block, err := s.callbacks.NewCandidateBock(req)
	if err != nil {
		return err
	}
	if err := stream.Send(block); err != nil {
		return err
	}

	s.clientsLock.Lock()
	s.clients[clientID] = stream
	s.lastHeaders[clientID] = block
	s.candidateRequests[clientID] = req
	s.clientsLock.Unlock()

	<-stream.Context().Done()

	s.clientsLock.Lock()
	delete(s.clients, clientID)
	delete(s.lastHeaders, clientID)
	delete(s.candidateRequests, clientID)
	s.clientsLock.Unlock()

	return nil
}

func (s *Server) Generate(ctx context.Context, req *pb.GenerateRequest) (*pb.GenerateResponse, error) {
	numBlocks := req.GetNumBlocks()
	if numBlocks <= 0 {
		return nil, fmt.Errorf("numBlocks must be greater than 0")
	}

	blocks, err := s.callbacks.GenerateBlocks(uint32(numBlocks))
	if err != nil {
		return nil, fmt.Errorf("failed to generate blocks: %w", err)
	}

	return &pb.GenerateResponse{
		Blocks: blocks,
	}, nil
}

func (s *Server) SubmitValidBlock(ctx context.Context, validBlock *pb.ValidBlock) (*pb.AckBlockSubmited, error) {

	header, err := s.callbacks.SubmitBlocks(validBlock)
	if err != nil {
		return nil, err
	}

	return &pb.AckBlockSubmited{
		Header: hex.EncodeToString(header),
	}, nil
}

// deprecated
func (s *Server) BroadcastInput(block *pb.CandidateBlock) {
	s.clientsLock.Lock()
	defer s.clientsLock.Unlock()

	for clientID, clientStream := range s.clients {
		if err := clientStream.Send(block); err != nil {
			delete(s.clients, clientID)
		}
	}
}

func (s *Server) SendNewWork() {
	s.clientsLock.Lock()
	defer s.clientsLock.Unlock()

	for clientID, clientStream := range s.clients {
		block, err := s.callbacks.NewCandidateBock(s.candidateRequests[clientID])
		if err != nil {
			continue
		}
		if err := clientStream.Send(block); err != nil {
			delete(s.clients, clientID)
		}
	}
}

type healthServer struct {
	pb.UnimplementedHealthServer
}

func (s *healthServer) Check(ctx context.Context, req *pb.HealthCheckRequest) (*pb.HealthCheckResponse, error) {
	return &pb.HealthCheckResponse{Status: pb.HealthStatus_SERVING}, nil
}

func Serve(callbacks *ServerCallbacks) (*Server, error) {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", SERVER_PORT))
	if err != nil {
		return nil, err
	}

	server := NewServer(callbacks)
	server.grpc = grpc.NewServer()

	pb.RegisterCandidateStreamServer(server.grpc, server)
	pb.RegisterHealthServer(server.grpc, &healthServer{})

	go func() {
		if err := server.grpc.Serve(lis); err != nil {
			log.Fatal().Err(err).Msg("failed")
		}
	}()

	log.Info().Msg("grpc server started")
	return server, nil
}
