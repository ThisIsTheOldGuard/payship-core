// Package service реализует бизнес-логику микросервиса заказов.
//
// Этот пакет содержит доменные правила, валидацию и статус-машину.
// Сервисный слой не зависит от транспорта (HTTP) и хранилища (БД),
// получая их через интерфейсы (Dependency Injection).
//
// Пример использования:
//
//	svc := service.NewOrderService(repo)
//	err := svc.CreateOrder(ctx, "Alice", 100.0)
package service

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/ThisIsTheOldGuard/payship-core/internal/domain"
	"github.com/ThisIsTheOldGuard/payship-core/internal/model"
)

// OrderRepository определяет контракт для операций с заказами в хранилище.
//
// Методы:
//   - Create: создаёт новый заказ, заполняя автоинкрементный ID.
//   - GetByID: возвращает заказ по уникальному идентификатору.
//   - ListOrders: возвращает страницу заказов с пагинацией и общим количеством.
//   - UpdateOrderTransition: обновляет статус существующего заказа.
//
// Пример реализации:
//
//	type orderRepo struct { pool *pgxpool.Pool }
//	func (r *orderRepo) GetByID(ctx context.Context, id int64) (*model.Order, error) { ... }
type OrderRepository interface {

	// Create сохраняет новый заказ в базе данных.
	//
	// Метод выполняет INSERT с RETURNING id для заполнения поля ID
	// входной структуры. Операция выполняется в контексте переданного ctx.
	//
	// Параметры:
	//   - ctx: контекст для отмены/таймаута запроса.
	//   - order: указатель на модель заказа; поле ID и CreatedAt будет заполнено после успеха.
	//
	// Возвращает:
	//   - error: ошибка выполнения запроса или нарушения ограничений БД.
	//
	// Пример:
	//
	//	order := &model.Order{CustomerName: "Bob", Amount: 50.0, Status: model.StatusPending}
	//	if err := repo.Create(ctx, order); err != nil {
	//	    return err
	//	}
	//	fmt.Println("Created order ID:", order.ID)
	Create(ctx context.Context, order *model.Order) error

	// GetByID возвращает заказ по его уникальному идентификатору.
	//
	// Метод выполняет SELECT по первичному ключу.
	//
	// Параметры:
	//   - ctx: контекст для отмены/таймаута запроса.
	//   - id: уникальный идентификатор заказа.
	//
	// Возвращает:
	//   - *model.Order: указатель на найденный заказ.
	//   - error: если заказ отсутствует, или ошибка БД.
	//
	// Пример:
	//
	//	order, err := repo.GetByID(ctx, 42)
	//	if errors.Is(err, repository.ErrOrderNotFound) {
	//	    // обработать 404
	//	}
	GetByID(ctx context.Context, id int64) (*model.Order, error)
	// ListOrders возвращает страницу заказов с поддержкой пагинации.
	//
	// Метод выполняет два запроса:
	//  1. COUNT(*) для получения общего количества записей.
	//  2. SELECT с LIMIT/OFFSET для выборки данных страницы.
	//
	// Параметры:
	//   - ctx: контекст для отмены/таймаута запроса.
	//   - offset: смещение первой записи (0-based).
	//   - limit: максимальное количество записей на странице.
	//
	// Возвращает:
	//   - []*model.Order: срез заказов на текущей странице.
	//   - int: общее количество заказов (для расчёта страниц).
	//   - error: ошибка выполнения запросов.
	//
	// Пример:
	//
	//	orders, total, err := repo.List(ctx, 0, 20)
	//	pages := (total + 19) / 20 // округление вверх
	ListOrders(ctx context.Context, limit, offset int) ([]*model.Order, int, error)
	// UpdateOrderTransition обновляет статус существующего заказа.
	//
	// Метод выполняет UPDATE по первичному ключу. Если заказ с указанным ID
	// не существует, операция завершается без ошибки (обновлено 0 строк).
	//
	// Параметры:
	//   - ctx: контекст для отмены/таймаута запроса.
	//   - id: уникальный идентификатор заказа.
	//   - status: новый допустимый статус заказа.
	//
	// Возвращает:
	//   - error: ошибка выполнения запроса.
	//
	// Пример:
	//
	//	err := repo.UpdateStatus(ctx, 123, model.StatusCompleted)
	//	if err != nil {
	//	    // обработать ошибку БД
	//	}
	UpdateOrderTransition(ctx context.Context, id int64, status model.OrderStatus) error
}

// orderService предоставляет методы для управления заказами.
//
// Сервис инкапсулирует бизнес-правила:
//   - Валидация входных данных (amount > 0, непустое имя).
//   - Статус-машина: разрешённые переходы между статусами.
//   - Маппинг ошибок репозитория в доменные ошибки.
//
// Зависимости:
//   - repo: реализация интерфейса OrderRepo для доступа к данным.
//
// Пример:
//
//	svc := NewOrderService(repo)
//	order, err := svc.GetOrder(ctx, 123)
type orderService struct {
	repo OrderRepository
}

type OrderService interface {
	// CreateOrder создаёт новый заказ.
	//
	// Параметры:
	//   - ctx: контекст для отмены/таймаута операции.
	//   - customerName: имя клиента (не пустая строка).
	//   - amount: сумма заказа (положительное число).
	//
	// Возвращает:
	//   - *model.Order: созданный заказ с заполненным ID.
	//   - error: ErrInvalidAmount при невалидных входных данных,
	//     или ошибка репозитория.
	CreateOrder(ctx context.Context, customerName string, amount float64) (*model.Order, error)
	// GetOrder возвращает заказ по его уникальному идентификатору.
	//
	// Параметры:
	//   - ctx: контекст для отмены/таймаута операции.
	//   - id: уникальный идентификатор заказа.
	//
	// Возвращает:
	//   - *model.Order: найденный заказ.
	//   - error: ошибку если заказ не найден или если что-то пошло не так.
	GetOrder(ctx context.Context, id int64) (*model.Order, error)
	// ListOrders возвращает страницу заказов с пагинацией.
	//
	// Параметры:
	//   - ctx: контекст для отмены/таймаута операции.
	//   - page: номер страницы (1-based).
	//   - limit: количество элементов на странице (макс. 100).
	//
	// Возвращает:
	//   - *model.OrderListResponse: ответ с заказами и метаданными пагинации.
	//   - error: ErrInvalidPage/ErrInvalidLimit при невалидных параметрах.
	ListOrders(ctx context.Context, page, limit int) ([]*model.Order, int, error)
	// UpdateOrderTransition изменяет статус заказа с проверкой правил перехода.
	//
	// Параметры:
	//   - ctx: контекст для отмены/таймаута операции.
	//   - id: уникальный идентификатор заказа.
	//   - status: новый статус в строковом формате.
	//
	// Возвращает:
	//   - error: ErrNotValidTransition, ErrOrderNotFound, ErrInvalidTransition
	//     или ошибка репозитория.
	UpdateOrderTransition(ctx context.Context, id int64, status string) error
}

// NewOrderService создаёт новый экземпляр сервиса заказов.
//
// Функция принимает реализацию интерфейса OrderRepo через
// зависимость-инъекцию. Это позволяет подменять репозиторий
// в тестах (mock) без изменения кода сервиса.
//
// Параметры:
//   - repo: реализация контракта доступа к заказам.
//
// Возвращает:
//   - *orderService: готовый к использованию сервис.
//
// Пример:
//
//	svc := service.NewOrderService(repo)
func NewOrderService(repo OrderRepository) OrderService {
	return &orderService{repo: repo}
}

// CreateOrder создаёт новый заказ.
//
// Параметры:
//   - ctx: контекст для отмены/таймаута операции.
//   - customerName: имя клиента (не пустая строка).
//   - amount: сумма заказа (положительное число).
//
// Возвращает:
//   - *model.Order: созданный заказ с заполненным ID.
//   - error: ErrInvalidAmount при невалидных входных данных,
//     или ошибка репозитория.
//
// Пример:
//
//	order, err := svc.CreateOrder(ctx, "Alice", 150.0)
//	if err != nil {
//	    // обработать ошибку
//	}
func (s *orderService) CreateOrder(ctx context.Context, customerName string, amount float64) (*model.Order, error) {

	if customerName == "" {
		return nil, domain.ErrEmptyCustomer
	}
	if amount <= 0 {
		return nil, domain.ErrInvalidAmount
	}

	order := &model.Order{
		CustomerName: customerName,
		Amount:       amount,
		Status:       model.StatusPending,
	}

	if err := s.repo.Create(ctx, order); err != nil {
		return nil, fmt.Errorf("service.CreateOrder: %w", err)
	}

	return order, nil
}

// GetOrder возвращает заказ по его уникальному идентификатору.
//
// Параметры:
//   - ctx: контекст для отмены/таймаута операции.
//   - id: уникальный идентификатор заказа.
//
// Возвращает:
//   - *model.Order: найденный заказ.
//   - error: ошибку если заказ не найден или если что-то пошло не так.
//
// Пример:
//
//	order, err := svc.GetOrder(ctx, 42)
//	if errors.Is(err, service.ErrOrderNotFound) {
//	    // вернуть 404 клиенту
//	}
func (s *orderService) GetOrder(ctx context.Context, id int64) (*model.Order, error) {
	order, err := s.repo.GetByID(ctx, id)
	if errors.Is(err, domain.ErrOrderNotFound) {
		return nil, domain.ErrOrderNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("service.GetOrder: %w", err)
	}
	return order, nil
}

// ListOrders возвращает страницу заказов с пагинацией.
//
// Параметры:
//   - ctx: контекст для отмены/таймаута операции.
//   - page: номер страницы (1-based).
//   - limit: количество элементов на странице (макс. 100).
//
// Возвращает:
//   - *model.OrderListResponse: ответ с заказами и метаданными пагинации.
//   - error: ErrInvalidPage/ErrInvalidLimit при невалидных параметрах.
//
// Пример:
//
//	// Запрос: /orders?page=2&limit=20
//	resp, err := svc.ListOrders(ctx, 2, 20)
func (s *orderService) ListOrders(ctx context.Context, limit, page int) ([]*model.Order, int, error) {
	if page < 1 {
		return nil, 0, domain.ErrInvalidPage
	}
	if limit < 1 || limit > 100 {
		return nil, 0, domain.ErrInvalidLimit
	}

	offset := (page - 1) * limit
	orders, count, err := s.repo.ListOrders(ctx, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("service.ListOrders: %w", err)
	}

	return orders, count, nil
}

// UpdateOrderTransition изменяет статус заказа с проверкой правил перехода.
//
// Параметры:
//   - ctx: контекст для отмены/таймаута операции.
//   - id: уникальный идентификатор заказа.
//   - status: новый статус в строковом формате.
//
// Возвращает:
//   - error: ErrNotValidTransition, ErrOrderNotFound, ErrInvalidTransition
//     или ошибка репозитория.
//
// Пример:
//
//	// Разрешено: pending → processing
//	err := svc.UpdateOrderTransition(ctx, 123, "processing")
//	// Запрещено: pending → completed (вернёт ErrInvalidTransition)
func (s *orderService) UpdateOrderTransition(ctx context.Context, id int64, status string) error {

	transition := model.OrderStatus(status)
	if !transition.Valid() {
		return domain.ErrNotValidTransition
	}

	order, err := s.repo.GetByID(ctx, id)
	if errors.Is(err, domain.ErrOrderNotFound) {
		return domain.ErrOrderNotFound
	}
	if err != nil {
		return fmt.Errorf("service.UpdateOrderTransition: %w", err)
	}

	if !validateTransition(order.Status, transition) {
		return domain.ErrInvalidTransition
	}

	return s.repo.UpdateOrderTransition(ctx, id, transition)

}

// allowedTransitions - карта допустимых переходов между статусами заказа.
//
// Определяет строгую бизнес-логику жизненного цикла заказа. Ключом карты является текущий статус заказа,
// а значением - срез статусов, в которые разрешен переход из данного состояния.
// Попытка перевести заказ в статус, отсутствующий в этом списке, будет считаться невалидной.
//
// Текущие правила:
//   - StatusPending (Ожидает):     можно перевести в StatusProcessing (В работе) или StatusCancelled (Отменен).
//   - StatusProcessing (В работе): можно перевести в StatusCompleted (Завершен) или StatusCancelled (Отменен).
var allowedTransitions = map[model.OrderStatus][]model.OrderStatus{
	model.StatusPending:    {model.StatusProcessing, model.StatusCancelled},
	model.StatusProcessing: {model.StatusCompleted, model.StatusCancelled},
}

// validateTransition проверяет, разрешен ли переход заказа из одного статуса в другой.
//
// Сверяет запрашиваемый переход с бизнес-правилами, описанными в карте allowedTransitions.
//
// Параметры:
//   - from: текущий статус заказа, из которого осуществляется переход.
//   - to:   целевой статус, в который планируется перевести заказ.
//
// Возвращает:
//   - bool: true, если переход легитимен и разрешен бизнес-логикой; false, если переход запрещен.
func validateTransition(from, to model.OrderStatus) bool {
	allowed, ok := allowedTransitions[from]
	if !ok {
		return false
	}
	return slices.Contains(allowed, to)
}
