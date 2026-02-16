package grpc

import (
	"context"
	"log/slog"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	disastersv1 "github.com/mr1hm/go-disaster-alerts/gen/disasters/v1"
	"github.com/mr1hm/go-disaster-alerts/internal/models"
	"github.com/mr1hm/go-disaster-alerts/internal/repository"
)

type Server struct {
	disastersv1.UnimplementedDisasterServiceServer
	repo        repository.DisasterRepository
	broadcaster *Broadcaster
	grpcServer  *grpc.Server
}

func NewServer(repo repository.DisasterRepository, broadcaster *Broadcaster) *Server {
	return &Server{
		repo:        repo,
		broadcaster: broadcaster,
	}
}

func (s *Server) Start(addr string) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	s.grpcServer = grpc.NewServer()
	disastersv1.RegisterDisasterServiceServer(s.grpcServer, s)

	slog.Info("gRPC server listening", "addr", addr)
	return s.grpcServer.Serve(lis)
}

func (s *Server) Stop() {
	if s.grpcServer != nil {
		s.grpcServer.GracefulStop()
	}
}

func (s *Server) GetDisaster(ctx context.Context, req *disastersv1.GetDisasterRequest) (*disastersv1.Disaster, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	disaster, err := s.repo.GetByID(ctx, req.Id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get disaster: %v", err)
	}
	if disaster == nil {
		return nil, status.Errorf(codes.NotFound, "disaster not found: %s", req.Id)
	}

	return toProto(disaster), nil
}

func (s *Server) ListDisasters(ctx context.Context, req *disastersv1.ListDisastersRequest) (*disastersv1.ListDisastersResponse, error) {
	filter := repository.Filter{
		Limit: int(req.Limit),
	}
	if req.Type != nil && *req.Type != disastersv1.DisasterType_UNSPECIFIED {
		filter.Type = req.Type
	}
	if req.MinMagnitude != nil {
		filter.MinMagnitude = req.MinMagnitude
	}
	if req.AlertLevel != nil && *req.AlertLevel != disastersv1.AlertLevel_UNKNOWN {
		filter.AlertLevel = req.AlertLevel
	}
	if req.MinAlertLevel != nil && *req.MinAlertLevel != disastersv1.AlertLevel_UNKNOWN {
		filter.MinAlertLevel = req.MinAlertLevel
	}
	if req.DiscordSent != nil {
		filter.DiscordSent = req.DiscordSent
	}

	disasters, err := s.repo.ListDisasters(ctx, filter)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list disasters: %v", err)
	}

	resp := &disastersv1.ListDisastersResponse{
		Disasters: make([]*disastersv1.Disaster, len(disasters)),
	}
	for i, d := range disasters {
		resp.Disasters[i] = toProto(&d)
	}
	return resp, nil
}

func (s *Server) StreamDisasters(req *disastersv1.StreamDisastersRequest, stream disastersv1.DisasterService_StreamDisastersServer) error {
	id, ch := s.broadcaster.Subscribe()
	defer s.broadcaster.Unsubscribe(id)

	slog.Info("client subscribed to disaster stream", "subscriber_id", id)

	for {
		select {
		case <-stream.Context().Done():
			slog.Info("client disconnected from disaster stream", "subscriber_id", id)
			return nil
		case d, ok := <-ch:
			if !ok {
				return nil
			}

			// Apply filters
			if req.Type != nil && *req.Type != disastersv1.DisasterType_UNSPECIFIED {
				if d.Type != *req.Type {
					continue
				}
			}
			if req.MinMagnitude != nil && d.Magnitude < *req.MinMagnitude {
				continue
			}
			if req.AlertLevel != nil && *req.AlertLevel != disastersv1.AlertLevel_UNKNOWN {
				if d.AlertLevel != *req.AlertLevel {
					continue
				}
			}
			if req.MinAlertLevel != nil && *req.MinAlertLevel != disastersv1.AlertLevel_UNKNOWN {
				if d.AlertLevel < *req.MinAlertLevel {
					continue
				}
			}

			if err := stream.Send(toProto(d)); err != nil {
				slog.Error("failed to send disaster to stream", "error", err, "subscriber_id", id)
				return err
			}
		}
	}
}

func (s *Server) AcknowledgeDisasters(ctx context.Context, req *disastersv1.AcknowledgeDisastersRequest) (*disastersv1.AcknowledgeDisastersResponse, error) {
	if len(req.Ids) == 0 {
		return &disastersv1.AcknowledgeDisastersResponse{AcknowledgedCount: 0}, nil
	}

	count, err := s.repo.MarkAsSent(ctx, req.Ids)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to acknowledge disasters: %v", err)
	}

	slog.Info("disasters acknowledged", "count", count, "ids", req.Ids)
	return &disastersv1.AcknowledgeDisastersResponse{AcknowledgedCount: count}, nil
}

func toProto(d *models.Disaster) *disastersv1.Disaster {
	return &disastersv1.Disaster{
		Id:         d.ID,
		Source:     d.Source,
		Type:       d.Type,
		Title:      d.Title,
		Magnitude:  d.Magnitude,
		AlertLevel: d.AlertLevel,
		Latitude:   d.Latitude,
		Longitude:  d.Longitude,
		Timestamp:  d.Timestamp.Unix(),
		Country:    d.Country,
		Population: d.Population,
		ReportUrl:  d.ReportURL,
	}
}
