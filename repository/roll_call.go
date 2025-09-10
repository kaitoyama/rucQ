//go:generate go tool mockgen -source=$GOFILE -destination=mockrepository/$GOFILE -package=mockrepository
package repository

import (
	"context"
	"errors"

	"github.com/traPtitech/rucQ/model"
)

var ErrRollCallNotFound = errors.New("roll call not found")

type RollCallRepository interface {
	CreateRollCall(ctx context.Context, rollCall *model.RollCall) error
	GetRollCalls(ctx context.Context, campID uint) ([]model.RollCall, error)
	GetLatestRollCall(ctx context.Context, campID uint) (*model.RollCall, error)
}
