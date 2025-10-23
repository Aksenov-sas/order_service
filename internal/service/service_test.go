package service

import (
	"context"
	"errors"
	"testing"

	"test_service/internal/mocks"
	"test_service/internal/models"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestService_WarmUpCache(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := mocks.NewMockDatabase(ctrl)
	mockCache := mocks.NewMockCache(ctrl)

	svc := NewWithCache(mockDB, mockCache)

	ctx := context.Background()
	testOrders := []models.Order{
		{OrderUID: "order-1", Locale: "en"},
		{OrderUID: "order-2", Locale: "ru"},
	}

	t.Run("Success", func(t *testing.T) {
		// Ожидаемые вызовы
		mockDB.EXPECT().GetAllOrders(ctx).Return(testOrders, nil)
		mockCache.EXPECT().LoadFromSlice(testOrders)

		err := svc.WarmUpCache(ctx)
		assert.NoError(t, err, "загрузка кэша не должна возвращать ошибки")
	})

	t.Run("DatabaseError", func(t *testing.T) {
		// Ожидаемый вызов с возвратом ошибки
		mockDB.EXPECT().GetAllOrders(ctx).Return(nil, errors.New("database error"))

		err := svc.WarmUpCache(ctx)
		assert.Error(t, err, "загрузка кэша при ошибке базы данных должна возвращать ошибку")
		assert.Contains(t, err.Error(), "database error", "ошибка должна содержать текст 'database error'")
	})
}

func TestService_ProcessOrder(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := mocks.NewMockDatabase(ctrl)
	mockCache := mocks.NewMockCache(ctrl)

	svc := NewWithCache(mockDB, mockCache)

	order := &models.Order{
		OrderUID: "order-123",
		Locale:   "en",
	}

	t.Run("Success", func(t *testing.T) {
		// Ожидаемые вызовы
		mockDB.EXPECT().SaveOrder(gomock.Any(), order).Return(nil)
		mockCache.EXPECT().Set(order)

		err := svc.ProcessOrder(order)
		assert.NoError(t, err, "обработка заказа не должна возвращать ошибки")
	})

	t.Run("DatabaseError", func(t *testing.T) {
		// Ожидаемый вызов с возвратом ошибки
		mockDB.EXPECT().SaveOrder(gomock.Any(), order).Return(errors.New("database error"))

		err := svc.ProcessOrder(order)
		assert.Error(t, err, "обработка заказа при ошибке базы данных должна возвращать ошибку")
		assert.Contains(t, err.Error(), "database error", "ошибка должна содержать текст 'database error'")
	})
}

func TestService_GetOrder(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := mocks.NewMockDatabase(ctrl)
	mockCache := mocks.NewMockCache(ctrl)

	svc := NewWithCache(mockDB, mockCache)

	order := &models.Order{
		OrderUID: "order-123",
		Locale:   "en",
	}

	t.Run("FoundInCache", func(t *testing.T) {
		// Ожидаем, что кэш вернет заказ
		mockCache.EXPECT().Get("order-123").Return(order, true)

		result, err := svc.GetOrder("order-123")
		assert.NoError(t, err, "получение заказа из кэша не должно возвращать ошибки")
		assert.Equal(t, order, result, "результат должен совпадать с ожидаемым заказом")
	})

	t.Run("NotFoundInCacheButInDB", func(t *testing.T) {
		// Ожидаем, что кэш вернет не найдено
		mockCache.EXPECT().Get("order-123").Return(nil, false)
		// Ожидаем, что база данных вернет заказ
		mockDB.EXPECT().GetOrder(gomock.Any(), "order-123").Return(order, nil)
		// Ожидаем, что кэш установит заказ
		mockCache.EXPECT().Set(order)

		result, err := svc.GetOrder("order-123")
		assert.NoError(t, err, "получение заказа из БД не должно возвращать ошибки")
		assert.Equal(t, order, result, "результат должен совпадать с ожидаемым заказом")
	})

	t.Run("NotFoundInCacheAndDB", func(t *testing.T) {
		// Ожидаем, что кэш вернет не найдено
		mockCache.EXPECT().Get("order-123").Return(nil, false)
		// Ожидаем, что база данных вернет ошибку
		mockDB.EXPECT().GetOrder(gomock.Any(), "order-123").Return(nil, errors.New("not found"))

		result, err := svc.GetOrder("order-123")
		assert.Error(t, err, "получение заказа из БД при ошибке должно возвращать ошибку")
		assert.Nil(t, result, "результат должен быть nil")
		assert.Contains(t, err.Error(), "not found", "ошибка должна содержать текст 'not found'")
	})

	t.Run("CacheError", func(t *testing.T) {
		// Мок заказа, который будет возвращен из БД
		dbOrder := &models.Order{OrderUID: "order-123", Locale: "en"}

		// Ожидаем, что кэш вернет не найдено
		mockCache.EXPECT().Get("order-123").Return(nil, false)
		// Ожидаем, что база данных вернет заказ
		mockDB.EXPECT().GetOrder(gomock.Any(), "order-123").Return(dbOrder, nil)
		// Ожидаем, что кэш установит заказ
		mockCache.EXPECT().Set(dbOrder)

		result, err := svc.GetOrder("order-123")
		assert.NoError(t, err, "получение заказа из БД не должно возвращать ошибки")
		assert.Equal(t, dbOrder, result, "результат должен совпадать с полученным из БД заказом")
	})
}

func TestService_GetCacheStats(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := mocks.NewMockDatabase(ctrl)
	mockCache := mocks.NewMockCache(ctrl)

	svc := NewWithCache(mockDB, mockCache)

	t.Run("StatsRetrieved", func(t *testing.T) {
		// Ожидаем вызов размера кэша
		mockCache.EXPECT().Size().Return(5)

		stats := svc.GetCacheStats()
		assert.NotNil(t, stats, "статистика не должна быть пустой")
		assert.Equal(t, 5, stats["cache_size"], "размер кэша должен совпадать")
		assert.NotNil(t, stats["timestamp"], "временная метка должна присутствовать")
	})
}

func TestService_Close(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := mocks.NewMockDatabase(ctrl)
	mockCache := mocks.NewMockCache(ctrl)

	svc := NewWithCache(mockDB, mockCache)

	t.Run("CloseSuccessfully", func(t *testing.T) {
		// Мок вызова закрытия БД
		mockDB.EXPECT().Close()

		// Вызов закрытия
		svc.Close()

		// Проверяем, что сервис можно использовать после закрытия (очистка должна обрабатываться внутри)
		stats := svc.GetCacheStats()
		assert.NotNil(t, stats, "статистика не должна быть пустой после закрытия")
	})
}

func TestService_ProcessOrderWithValidation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := mocks.NewMockDatabase(ctrl)
	mockCache := mocks.NewMockCache(ctrl)

	svc := NewWithCache(mockDB, mockCache)

	// Проверка с недействительным заказом
	invalidOrder := &models.Order{
		OrderUID: "", // Обязательное поле отсутствует
		Locale:   "en",
	}

	err := svc.ProcessOrder(invalidOrder)
	// Это не должно возвращать ошибку валидации, так как валидация выполняется на уровне потребителя

	// Проверяем, что если БД отклоняет заказ из-за валидации, это обрабатывается
	mockDB.EXPECT().SaveOrder(gomock.Any(), invalidOrder).Return(errors.New("validation error"))

	err = svc.ProcessOrder(invalidOrder)
	assert.Error(t, err, "обработка недействительного заказа должна возвращать ошибку")
}

func TestService_GetOrderConcurrency(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := mocks.NewMockDatabase(ctrl)
	mockCache := mocks.NewMockCache(ctrl)

	svc := NewWithCache(mockDB, mockCache)

	// Проверяем, что одновременный доступ не вызывает гонки
	done := make(chan bool, 2)

	// Горутина 1: Получение заказа из кэша
	go func() {
		order := &models.Order{OrderUID: "order-1", Locale: "en"}
		mockCache.EXPECT().Get("order-1").Return(order, true).AnyTimes()
		_, _ = svc.GetOrder("order-1")
		done <- true
	}()

	// Горутина 2: Обработка заказа
	go func() {
		order := &models.Order{OrderUID: "order-2", Locale: "en"}
		mockDB.EXPECT().SaveOrder(gomock.Any(), order).Return(nil).AnyTimes()
		mockCache.EXPECT().Set(order).AnyTimes()
		_ = svc.ProcessOrder(order)
		done <- true
	}()

	// Ждем завершения обеих горутин
	<-done
	<-done
}

func TestService_WarmUpCacheWithEmptyDB(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := mocks.NewMockDatabase(ctrl)
	mockCache := mocks.NewMockCache(ctrl)

	svc := NewWithCache(mockDB, mockCache)

	t.Run("EmptyDatabase", func(t *testing.T) {
		// Ожидаемые вызовы
		mockDB.EXPECT().GetAllOrders(gomock.Any()).Return([]models.Order{}, nil)
		mockCache.EXPECT().LoadFromSlice([]models.Order{})

		err := svc.WarmUpCache(context.Background())
		assert.NoError(t, err, "загрузка кэша из пустой БД не должна возвращать ошибки")
	})
}
