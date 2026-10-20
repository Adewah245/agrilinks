package memory

import (
	"context"
	"sync"

	"backend/internal/models"
)

type SubmitterRepo struct {
	mu   sync.RWMutex
	data map[string]*models.Submitter
}

func NewSubmitterRepo() *SubmitterRepo {
	return &SubmitterRepo{data: make(map[string]*models.Submitter)}
}

func (r *SubmitterRepo) Get(ctx context.Context, submitterID string) (*models.Submitter, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.data[submitterID]
	if !ok {
		// Unknown submitters default to a fresh, unverified, neutral-trust
		// profile rather than erroring — first-time farmers should still
		// be able to submit, just with lighter initial weight.
		return &models.Submitter{
			ID:         submitterID,
			Role:       models.RoleFarmer,
			Verified:   false,
			TrustScore: 0.5,
		}, nil
	}
	return s, nil
}

func (r *SubmitterRepo) Save(ctx context.Context, s *models.Submitter) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.data[s.ID] = s
	return nil
}
