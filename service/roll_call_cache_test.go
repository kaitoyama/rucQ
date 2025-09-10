package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/traPtitech/rucQ/model"
	"github.com/traPtitech/rucQ/repository"
	"github.com/traPtitech/rucQ/repository/mockrepository"
)

func TestRollCallCacheService_GetLatestRollCall(t *testing.T) {
	t.Run("First call should fetch from repository and cache", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		
		mockRepo := mockrepository.NewMockRollCallRepository(ctrl)
		service := NewRollCallCacheService(mockRepo)
		
		expectedRollCall := &model.RollCall{
			Name:        "Test Roll Call",
			Description: "Test Description",
			CampID:      1,
		}
		expectedRollCall.ID = 5 // Simulate gorm.Model ID
		
		mockRepo.EXPECT().GetLatestRollCall(gomock.Any(), uint(1)).Return(expectedRollCall, nil)
		
		ctx := context.Background()
		result, err := service.GetLatestRollCall(ctx, 1)
		
		assert.NoError(t, err)
		assert.Equal(t, expectedRollCall, result)
		
		// Verify cache was populated
		service.mu.RLock()
		cachedID, exists := service.cache[1]
		service.mu.RUnlock()
		
		assert.True(t, exists)
		assert.Equal(t, uint(5), cachedID)
	})
	
	t.Run("Second call should still fetch from repository to check for updates", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		
		mockRepo := mockrepository.NewMockRollCallRepository(ctrl)
		service := NewRollCallCacheService(mockRepo)
		
		// Set up cache
		service.cache[1] = 5
		
		expectedRollCall := &model.RollCall{
			Name:        "Updated Roll Call",
			Description: "Updated Description",
			CampID:      1,
		}
		expectedRollCall.ID = 6 // Newer roll call
		
		mockRepo.EXPECT().GetLatestRollCall(gomock.Any(), uint(1)).Return(expectedRollCall, nil)
		
		ctx := context.Background()
		result, err := service.GetLatestRollCall(ctx, 1)
		
		assert.NoError(t, err)
		assert.Equal(t, expectedRollCall, result)
		
		// Verify cache was updated with new ID
		service.mu.RLock()
		cachedID, exists := service.cache[1]
		service.mu.RUnlock()
		
		assert.True(t, exists)
		assert.Equal(t, uint(6), cachedID)
	})
	
	t.Run("Should handle repository errors", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		
		mockRepo := mockrepository.NewMockRollCallRepository(ctrl)
		service := NewRollCallCacheService(mockRepo)
		
		mockRepo.EXPECT().GetLatestRollCall(gomock.Any(), uint(1)).Return(nil, repository.ErrRollCallNotFound)
		
		ctx := context.Background()
		result, err := service.GetLatestRollCall(ctx, 1)
		
		assert.Error(t, err)
		assert.Equal(t, repository.ErrRollCallNotFound, err)
		assert.Nil(t, result)
	})
}

func TestRollCallCacheService_InvalidateCache(t *testing.T) {
	t.Run("Should remove cache entry", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		
		mockRepo := mockrepository.NewMockRollCallRepository(ctrl)
		service := NewRollCallCacheService(mockRepo)
		
		// Set up cache
		service.cache[1] = 5
		service.cache[2] = 10
		
		// Invalidate cache for camp 1
		service.InvalidateCache(1)
		
		// Verify only camp 1 was removed
		service.mu.RLock()
		_, exists1 := service.cache[1]
		_, exists2 := service.cache[2]
		service.mu.RUnlock()
		
		assert.False(t, exists1)
		assert.True(t, exists2)
	})
}