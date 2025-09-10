package router

import (
	"context"

	"github.com/sesopenko/genericpubsub"

	"github.com/traPtitech/rucQ/api"
	"github.com/traPtitech/rucQ/repository"
	"github.com/traPtitech/rucQ/service"
)

type reactionEvent struct {
	rollCallID uint
	data       api.RollCallReactionEvent
}

type Server struct {
	repo                 repository.Repository
	notificationService  service.NotificationService
	traqService          service.TraqService
	rollCallCacheService *service.RollCallCacheService
	reactionPubSub       *genericpubsub.PubSub[reactionEvent]
	isDev                bool
}

const maxReactionEventBuffer = 100

func NewServer(
	ctx context.Context,
	repo repository.Repository,
	notificationService service.NotificationService,
	traqService service.TraqService,
	isDev bool,
) *Server {
	rollCallCacheService := service.NewRollCallCacheService(repo)
	
	return &Server{
		repo:                 repo,
		notificationService:  notificationService,
		traqService:          traqService,
		rollCallCacheService: rollCallCacheService,
		reactionPubSub: genericpubsub.New[reactionEvent](
			ctx,
			maxReactionEventBuffer,
		),
		isDev: isDev,
	}
}
