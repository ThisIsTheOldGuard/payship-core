package model

type OrderListResponse struct {
	Items []*Order `json:"items"`
	Total int      `json:"total"`
	Page  int      `json:"page"`
	Limit int      `json:"limit"`
}
