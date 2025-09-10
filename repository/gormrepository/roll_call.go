package gormrepository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/traPtitech/rucQ/model"
	"github.com/traPtitech/rucQ/repository"
)

func (r *Repository) CreateRollCall(ctx context.Context, rollCall *model.RollCall) error {
	if err := r.db.WithContext(ctx).Omit("Subjects.*").Create(rollCall).Error; err != nil {
		if errors.Is(err, gorm.ErrForeignKeyViolated) {
			// 外部キーエラーが起きたときはCampかUserが存在しないので、
			// どちらが存在しないかを確認して適切なエラーを返す
			campExists, err := r.campExists(ctx, rollCall.CampID)

			if err != nil {
				return err
			}

			if !campExists {
				return repository.ErrCampNotFound
			}

			return repository.ErrUserNotFound
		}

		return err
	}

	return nil
}

func (r *Repository) GetRollCalls(ctx context.Context, campID uint) ([]model.RollCall, error) {
	rollCalls, err := gorm.G[model.RollCall](r.db).
		Preload("Reactions", nil).
		Preload("Subjects", nil).
		Where("camp_id = ?", campID).
		Find(ctx)

	if err != nil {
		return nil, err
	}

	// RollCallが見つからなかった場合、Campが存在しない可能性を考慮してCampの存在確認を行う
	if len(rollCalls) == 0 {
		campExists, err := r.campExists(ctx, campID)

		if err != nil {
			return nil, err
		}

		if !campExists {
			return nil, repository.ErrCampNotFound
		}
	}

	return rollCalls, nil
}

func (r *Repository) GetLatestRollCall(ctx context.Context, campID uint) (*model.RollCall, error) {
	rollCall, err := gorm.G[model.RollCall](r.db).
		Preload("Reactions", nil).
		Preload("Subjects", nil).
		Where("camp_id = ?", campID).
		Order("id DESC").
		First(ctx)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Check if camp exists to provide better error message
			campExists, campErr := r.campExists(ctx, campID)
			if campErr != nil {
				return nil, campErr
			}
			if !campExists {
				return nil, repository.ErrCampNotFound
			}
			return nil, repository.ErrRollCallNotFound
		}
		return nil, err
	}

	return &rollCall, nil
}

func (r *Repository) rollCallExists(ctx context.Context, rollCallID uint) (bool, error) {
	var count int64

	if err := r.db.
		WithContext(ctx).
		Model(&model.RollCall{}).
		Where("id = ?", rollCallID).
		Count(&count).
		Error; err != nil {
		return false, err
	}

	return count > 0, nil
}
