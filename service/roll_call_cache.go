package service

import (
	"context"
	"sync"

	"github.com/traPtitech/rucQ/model"
	"github.com/traPtitech/rucQ/repository"
)

// RollCallCacheService provides caching for latest roll call information
type RollCallCacheService struct {
	repo  repository.RollCallRepository
	cache map[uint]uint // campID -> latest roll call ID
	mu    sync.RWMutex
}

// NewRollCallCacheService creates a new RollCallCacheService
func NewRollCallCacheService(repo repository.RollCallRepository) *RollCallCacheService {
	return &RollCallCacheService{
		repo:  repo,
		cache: make(map[uint]uint),
	}
}

// GetLatestRollCall gets the latest roll call for a camp with caching
func (s *RollCallCacheService) GetLatestRollCall(ctx context.Context, campID uint) (*model.RollCall, error) {
	// Try to get from cache first
	s.mu.RLock()
	cachedID, exists := s.cache[campID]
	s.mu.RUnlock()

	if exists {
		// We have a cached latest ID, but we still need to fetch the actual roll call
		// In case of updates to the roll call content
		rollCall, err := s.repo.GetLatestRollCall(ctx, campID)
		if err != nil {
			return nil, err
		}
		
		// Update cache if we found a newer roll call
		if rollCall.ID > cachedID {
			s.mu.Lock()
			s.cache[campID] = rollCall.ID
			s.mu.Unlock()
		}
		
		return rollCall, nil
	}

	// No cache, get from repository and cache the result
	rollCall, err := s.repo.GetLatestRollCall(ctx, campID)
	if err != nil {
		return nil, err
	}

	// Cache the latest ID
	s.mu.Lock()
	s.cache[campID] = rollCall.ID
	s.mu.Unlock()

	return rollCall, nil
}

// InvalidateCache removes cache entry for a camp (call when new roll call is created)
func (s *RollCallCacheService) InvalidateCache(campID uint) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.cache, campID)
}