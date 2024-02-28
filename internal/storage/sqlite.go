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
	//_, err := r.DB.Exec(`INSERT INTO requests(host, path, method, headers, body, params, cookies) VALUES($1,$2,$3,$4,$5,$6,$7)`,
	//	req.Host, req.Path, req.Method, req.Headers, req.Body, req, req.Cookies)
	//return err
}

func (r requestsRepo) ReadRequest(id int) (models.RequestResponse, error) {
	panic("Implement me")

}
func (r requestsRepo) ReadAllRequest() ([]models.RequestResponse, error) {
	var reqs []models.RequestResponse
	result := r.DB.Find(&reqs)

	if result.Error != nil {
		return nil, result.Error
	}

	return reqs, nil
}
