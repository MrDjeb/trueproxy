package storage

import (
	"github.com/mrdjeb/trueproxy/internal/models"
	"gorm.io/gorm"
)

type requestsRepo struct {
	DB *gorm.DB
}

func NewRequestsRepo(db *gorm.DB) RequestsRepo {
	return &requestsRepo{
		DB: db,
	}
}

func (r requestsRepo) CreateRequest(req *models.RequestResponse) error {
	r.DB.Create(req)
	return nil
}

func (r requestsRepo) ReadRequest(ID uint) (models.RequestResponse, error) {
	var req models.RequestResponse
	result := r.DB.Find(&req, ID)

	if result.Error != nil {
		return models.RequestResponse{}, result.Error
	}

	if result.RowsAffected == 0 {
		return models.RequestResponse{}, ErrRequestNotFound
	}

	return req, nil

}
func (r requestsRepo) ReadAllRequest() ([]models.RequestResponse, error) {
	var reqs []models.RequestResponse
	result := r.DB.Find(&reqs)

	if result.Error != nil {
		return nil, result.Error
	}

	if result.RowsAffected == 0 {
		return nil, ErrRequestNotFound
	}

	return reqs, nil
}
