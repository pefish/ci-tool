package db

import (
	"time"
)

type DbTime struct {
	CreatedAt time.Time `json:"created_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

type IdType struct {
	Id uint64 `json:"id,omitempty"`
}

type Project struct {
	IdType
	Name          string  `json:"name"`
	Config        *string `json:"config"`
	Port          uint64  `json:"port"`
	Status        uint64  `json:"status"`
	IsAutoRestart uint64  `json:"is_auto_restart"`
	DbTime
}
