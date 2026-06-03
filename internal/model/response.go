package model

// OrderListResponse - структура возврата списка заказов.
type OrderListResponse struct {
	Items []*Order `json:"items"`
	Total int      `json:"total"`
	Page  int      `json:"page"`
	Limit int      `json:"limit"`
}
