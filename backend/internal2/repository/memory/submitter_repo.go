package memory

import (
	"context"
	"sync"

	"backend/internal2/domain"
)

type SubmitterRepo struct {
	mu   sync.RWMutex
	data map[string]*domain.Submitter
}

func NewSubmitterRepo() *SubmitterRepo {
	return &SubmitterRepo{data: make(map[string]*domain.Submitter)}
}

func (r *SubmitterRepo) Get(ctx context.Context, submitterID string) (*domain.Submitter, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.data[submitterID]
	if !ok {
		// Unknown submitters default to a fresh, unverified, neutral-trust
		// profile rather than erroring — first-time farmers should still
		// be able to submit, just with lighter initial weight.
		return &domain.Submitter{
			ID:         submitterID,
			Role:       domain.RoleFarmer,
			Verified:   false,
			TrustScore: 0.5,
		}, nil
	}
	return s, nil
}

func (r *SubmitterRepo) Save(ctx context.Context, s *domain.Submitter) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.data[s.ID] = s
	return nil
}
