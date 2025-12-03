package adapter

import "github.com/kizuna-org/akari/internal/message/domain"

type validationRepository struct{}

func NewValidationRepository() domain.ValidationRepository {
	return &validationRepository{}
}

func (r *validationRepository) ShouldProcessMessage(message *domain.Message) bool {
	if message == nil || message.Content == "" {
		return false
	}

	return true
}
