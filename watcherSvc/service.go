package watcherSvc

import (
	"context"
	"fmt"

	"github.com/DenysNahurnyi/deal/pb/generated/pb"
	"github.com/go-kit/kit/log"

	"github.com/mongodb/mongo-go-driver/mongo"
)

type service struct {
	envType       string
	dbClient      *mongo.Client
	dataSvcClient *pb.DataServiceClient
}

// NewService creates new service of watchSvc that allows to call it's functions to handle watcherSvc domain
func NewService(logger log.Logger, mgc *mongo.Client, dataSvcClient *pb.DataServiceClient) (Service, error) {
	return &service{
		envType:       "test",
		dbClient:      mgc,
		dataSvcClient: dataSvcClient,
	}, nil
}

type Service interface {
	HoldAndWatch(ctx context.Context, dealID string) error
}

func (s *service) HoldAndWatch(ctx context.Context, dealID string) error {
	fmt.Println("[LOG]:", "HoldAndWatch method with dealID: ", dealID)
	return nil
}
