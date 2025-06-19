package server

import (
	"context"
	"errors"
	"strings"

	"github.com/AlexMickh/speak-chat/internal/models"
	"github.com/AlexMickh/speak-chat/pkg/logger"
	"github.com/AlexMickh/speak-protos/pkg/api/chat"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Service interface {
	CreateChat(
		ctx context.Context,
		name string,
		description string,
		avatar []byte,
		chatOwnerId string,
	) (string, error)
	GetChat(ctx context.Context, id string) (models.Chat, error)
	AddParticipant(ctx context.Context, userId, chatId, participantId string) error
	UpdateChatInfo(
		ctx context.Context,
		userId string,
		chatId string,
		name string,
		description string,
		avatar []byte,
	) (models.Chat, error)
	DeleteChat(ctx context.Context, userId, chatId string) error
}

type AuthClient interface {
	GetUserId(ctx context.Context, token string) (string, error)
}

type Server struct {
	chat.UnimplementedChatServer
	service    Service
	authClient AuthClient
}

func New(service Service, authClient AuthClient) *Server {
	return &Server{
		service:    service,
		authClient: authClient,
	}
}

func (s *Server) CreateChat(ctx context.Context, req *chat.CreateChatRequest) (*chat.CreateChatResponse, error) {
	const op = "server.CreateChat"

	ctx = logger.GetFromCtx(ctx).With(ctx, zap.String("op", op))

	if req.GetName() == "" {
		logger.GetFromCtx(ctx).Error(ctx, "name is empty")
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	token, err := getAuthToken(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	userId, err := s.authClient.GetUserId(ctx, token)
	if err != nil {
		logger.GetFromCtx(ctx).Error(ctx, "failed to get user id from token")
		return nil, status.Error(codes.Internal, "failed to get user id")
	}
	// userId := uuid.NewString()

	chatId, err := s.service.CreateChat(ctx, req.GetName(), req.GetDescription(), req.GetChatImage(), userId)
	if err != nil {
		logger.GetFromCtx(ctx).Error(ctx, "failed to create chat", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to create chat")
	}

	return &chat.CreateChatResponse{
		Id: chatId,
	}, nil
}

func (s *Server) GetChat(ctx context.Context, req *chat.GetChatRequest) (*chat.GetChatResponse, error) {
	const op = "grpc.server.GetChat"

	ctx = logger.GetFromCtx(ctx).With(
		ctx,
		zap.String("op", op),
		zap.String("chat_id", req.GetId()),
	)

	if req.GetId() == "" {
		logger.GetFromCtx(ctx).Error(ctx, "id is empty")
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	chatInfo, err := s.service.GetChat(ctx, req.GetId())
	if err != nil {
		logger.GetFromCtx(ctx).Error(ctx, "failed to get chat", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to get chat")
	}

	return &chat.GetChatResponse{
		Chat: &chat.ChatType{
			Id:             chatInfo.ID,
			Name:           chatInfo.Name,
			Description:    chatInfo.Description,
			ChatImageUrl:   chatInfo.ChatImageUrl,
			ChatOwnerId:    chatInfo.ChatOwnerId,
			ParticipantsId: chatInfo.ParticipantsId,
		},
	}, nil
}

func (s *Server) AddParticipant(ctx context.Context, req *chat.AddParticipantRequest) (*emptypb.Empty, error) {
	const op = "grpc.server.AddParticipant"

	ctx = logger.GetFromCtx(ctx).With(
		ctx,
		zap.String("op", op),
		zap.String("chat_id", req.GetChatId()),
	)

	if req.GetChatId() == "" {
		logger.GetFromCtx(ctx).Error(ctx, "chat id is empty")
		return nil, status.Error(codes.InvalidArgument, "chat id is required")
	}
	if req.GetParticipantId() == "" {
		logger.GetFromCtx(ctx).Error(ctx, "participant id is empty")
		return nil, status.Error(codes.InvalidArgument, "participant id is required")
	}

	token, err := getAuthToken(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	userId, err := s.authClient.GetUserId(ctx, token)
	if err != nil {
		logger.GetFromCtx(ctx).Error(ctx, "failed to get user id from token")
		return nil, status.Error(codes.Internal, "failed to get user id")
	}
	// userId := "01c13427-f08b-4b93-9971-08466fc1b038"

	err = s.service.AddParticipant(ctx, userId, req.GetChatId(), req.GetParticipantId())
	if err != nil {
		logger.GetFromCtx(ctx).Error(ctx, "failed to add participant to the chat", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to add participant to the chat")
	}

	return &emptypb.Empty{}, nil
}

func (s *Server) UpdateChatInfo(ctx context.Context, req *chat.UpdateChatInfoRequest) (*chat.UpdateChatInfoResponse, error) {
	const op = "grpc.server.UpdateChatInfo"

	ctx = logger.GetFromCtx(ctx).With(
		ctx,
		zap.String("op", op),
		zap.String("chat_id", req.GetId()),
	)

	if req.GetId() == "" {
		logger.GetFromCtx(ctx).Error(ctx, "chat id is empty")
		return nil, status.Error(codes.InvalidArgument, "chat id is required")
	}

	if req.GetName() == "" && req.GetDescription() == "" && req.GetChatImage() == nil {
		logger.GetFromCtx(ctx).Error(ctx, "nothing to update")
		return nil, status.Error(codes.InvalidArgument, "nothing to update")
	}

	token, err := getAuthToken(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	userId, err := s.authClient.GetUserId(ctx, token)
	if err != nil {
		logger.GetFromCtx(ctx).Error(ctx, "failed to get user id from token")
		return nil, status.Error(codes.Internal, "failed to get user id")
	}

	chatInfo, err := s.service.UpdateChatInfo(
		ctx,
		userId,
		req.GetId(),
		req.GetName(),
		req.GetDescription(),
		req.GetChatImage(),
	)
	if err != nil {
		logger.GetFromCtx(ctx).Error(ctx, "failed to update chat info", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to update chat info")
	}

	return &chat.UpdateChatInfoResponse{
		Chat: &chat.ChatType{
			Id:             chatInfo.ID,
			Name:           chatInfo.Name,
			Description:    chatInfo.Description,
			ChatImageUrl:   chatInfo.ChatImageUrl,
			ChatOwnerId:    chatInfo.ChatOwnerId,
			ParticipantsId: chatInfo.ParticipantsId,
		},
	}, nil
}

func (s *Server) DeleteChat(ctx context.Context, req *chat.DeleteChatRequest) (*emptypb.Empty, error) {
	const op = "grpc.server.DeleteChat"

	ctx = logger.GetFromCtx(ctx).With(
		ctx,
		zap.String("op", op),
		zap.String("chat_id", req.GetId()),
	)

	if req.GetId() == "" {
		logger.GetFromCtx(ctx).Error(ctx, "chat id is empty")
		return nil, status.Error(codes.InvalidArgument, "chat id is required")
	}

	token, err := getAuthToken(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	userId, err := s.authClient.GetUserId(ctx, token)
	if err != nil {
		logger.GetFromCtx(ctx).Error(ctx, "failed to get user id from token")
		return nil, status.Error(codes.Internal, "failed to get user id")
	}

	err = s.service.DeleteChat(ctx, userId, req.GetId())
	if err != nil {
		logger.GetFromCtx(ctx).Error(ctx, "failed to delete chat", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to delete chat")
	}

	return &emptypb.Empty{}, nil
}

func getAuthToken(ctx context.Context) (string, error) {
	const op = "server.getAuthToken"

	ctx = logger.GetFromCtx(ctx).With(ctx, zap.String("op", op))

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		logger.GetFromCtx(ctx).Error(ctx, "failed to get metadata")
		return "", errors.New("metadata is empty")
	}

	auth, ok := md["authorization"]
	if !ok {
		logger.GetFromCtx(ctx).Error(ctx, "failed to get auth header")
		return "", errors.New("authorization header is empty")
	}

	if strings.Split(auth[0], " ")[0] != "Bearer" {
		logger.GetFromCtx(ctx).Error(ctx, "wrong token type")
		return "", errors.New("wrong token type, need Bearer")
	}

	token := strings.Split(auth[0], " ")[1]

	return token, nil
}
