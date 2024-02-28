package storage

import (
	"errors"

	"github.com/mrdjeb/trueproxy/internal/models"
)

var (
	ErrRequestNotFound = errors.New("request not found")
)

type RequestsRepo interface {
	CreateRequest(*models.RequestResponse) error
	ReadRequest(uint) (models.RequestResponse, error)
	ReadAllRequest() ([]models.RequestResponse, error)
}

/*
/requests – список запросов
/requests/id – вывод 1 запроса
/repeat/id – повторная отправка запроса
/scan/id – сканирование запроса)
*/
