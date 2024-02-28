package storage

import "github.com/mrdjeb/trueproxy/internal/models"

type RequestsRepo interface {
	CreateRequest(*models.RequestResponse) error
	ReadRequest(id int) (models.RequestResponse, error)
	ReadAllRequest() ([]models.RequestResponse, error)
}

/*
/requests – список запросов
/requests/id – вывод 1 запроса
/repeat/id – повторная отправка запроса
/scan/id – сканирование запроса)
*/
