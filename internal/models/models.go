package models

import (
	"gorm.io/gorm"
)

type RequestResponse struct {
	gorm.Model
	Request  Request  `gorm:"embedded"`
	Response Response `gorm:"embedded"`
}

type Request struct {
	Method     string
	Path       string
	Host       string
	GetParams  map[string][]string `gorm:"serializer:json"`
	Headers    map[string][]string `gorm:"serializer:json"`
	Cookies    map[string]string   `gorm:"serializer:json"`
	PostParams map[string][]string `gorm:"serializer:json"`
	Body       string
	Raw        string
}

type Response struct {
	StatusCode int                 `gorm:"column:Response_StatusCode"`
	Headers    map[string][]string `gorm:"serializer:json;column:Response_Headers"`
	Cookies    map[string]string   `gorm:"serializer:json;column:Response_Cookies"`
	PostParams map[string][]string `gorm:"serializer:json;column:Response_PostParams"`
	Body       string              `gorm:"column:Response_Body"`
	Raw        string              `gorm:"column:Response_Raw"`
}
