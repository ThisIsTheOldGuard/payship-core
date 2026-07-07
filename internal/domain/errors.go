package domain

import "errors"

var (
	// ErrEmptyCustomer возвращается если имя заказчика пустое.
	ErrEmptyCustomer = errors.New("customer_name is required")
	// ErrInvalidAmount возвращается при попытке создать заказ с невалидной суммой.
	ErrInvalidAmount = errors.New("amount must be greater than 0")
	// ErrOrderNotFound возвращается, когда запрошенный заказ не существует.
	ErrOrderNotFound = errors.New("order not found")
	// ErrInvalidPage возвращается при попытке запроса несуществующей страницы.
	ErrInvalidPage = errors.New("page must be >= 1")
	// ErrInvalidLimit возвращается при превышении лимита запрашиваемых страниц.
	ErrInvalidLimit = errors.New("limit must be between 1 and 100")
	// ErrNotValidTransition возвращается при передаче неизвестного значения статуса.
	ErrNotValidTransition = errors.New("not valid transition")
	// ErrInvalidTransition возвращается при попытке недопустимого перехода статуса.
	ErrInvalidTransition = errors.New("invalid transition")
)
