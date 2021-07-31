package models

type Config struct {
	Interval string `json:"interval"`
	Labels    map[string]string `json:"labels"`
	Namespace string            `json:"namespace"`
}
