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
	StatusCode int
	Headers    map[string][]string `gorm:"serializer:json"`
	Cookies    map[string]string   `gorm:"serializer:json"`
	PostParams map[string][]string `gorm:"serializer:json"`
	Body       string
	Raw        string
}
