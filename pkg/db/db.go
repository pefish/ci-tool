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

type CiParams struct {
	Env           string `json:"env" validate:"required"`
	Name          string `json:"name"`
	ImageName     string `json:"image_name"`
	Repo          string `json:"repo" validate:"required"`
	FetchCodeKey  string `json:"fetch_code_key" validate:"required"`
	DockerNetwork string `json:"docker_network"`
}

type Project struct {
	IdType
	Name          string    `json:"name"`
	Params        *CiParams `json:"params"`
	Config        *string   `json:"config"`
	Port          uint64    `json:"port"`
	Status        uint64    `json:"status"`
	IsAutoRestart uint64    `json:"is_auto_restart"`
	Restart       uint64    `json:"restart"`
	Stop          uint64    `json:"stop"`
	Start         uint64    `json:"start"`
	Rebuild       uint64    `json:"rebuild"`
	DbTime
}
