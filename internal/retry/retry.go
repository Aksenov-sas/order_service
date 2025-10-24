// Package retry предоставляет механизмы повторных попыток для отказоустойчивости
package retry

import (
	"context"
	"math/rand"
	"time"
)

// Policy определяет политику повторных попыток
type Policy struct {
	MaxAttempts    int           // Максимальное количество попыток
	InitialBackoff time.Duration // Начальная задержка между попытками
	MaxBackoff     time.Duration // Максимальная задержка между попытками
	BackoffFactor  float64       // Фактор увеличения задержки
	Jitter         bool          // Добавлять ли случайную задержку (jitter)
}

// DefaultPolicy возвращает стандартную политику повторных попыток
func DefaultPolicy() Policy {
	return Policy{
		MaxAttempts:    3,
		InitialBackoff: 100 * time.Millisecond,
		MaxBackoff:     10 * time.Second,
		BackoffFactor:  2.0,
		Jitter:         true,
	}
}

// LightPolicy возвращает легкую политику повторных попыток для быстрых операций
func LightPolicy() Policy {
	return Policy{
		MaxAttempts:    2,
		InitialBackoff: 50 * time.Millisecond,
		MaxBackoff:     1 * time.Second,
		BackoffFactor:  1.5,
		Jitter:         true,
	}
}

// HeavyPolicy возвращает строгую политику повторных попыток для критических операций
func HeavyPolicy() Policy {
	return Policy{
		MaxAttempts:    5,
		InitialBackoff: 200 * time.Millisecond,
		MaxBackoff:     30 * time.Second,
		BackoffFactor:  2.5,
		Jitter:         true,
	}
}

// RetryableFunc тип функции, которую можно повторять
type RetryableFunc func() error

// ContextRetryableFunc тип функции с контекстом, которую можно повторять
type ContextRetryableFunc func(context.Context) error

// Do выполняет функцию с повторными попытками согласно политике
func Do(policy Policy, fn RetryableFunc) error {
	return DoWithContext(context.Background(), policy, func(_ context.Context) error {
		return fn()
	})
}

// DoWithContext выполняет функцию с контекстом и повторными попытками согласно политике
func DoWithContext(ctx context.Context, policy Policy, fn ContextRetryableFunc) error {
	if policy.MaxAttempts <= 0 {
		policy.MaxAttempts = 1
	}

	backoff := policy.InitialBackoff
	var lastErr error

	for attempt := 0; attempt < policy.MaxAttempts; attempt++ {
		// Проверяем контекст на отмену
		select {
		case <-ctx.Done():
			if lastErr != nil {
				return lastErr
			}
			return ctx.Err()
		default:
		}

		// Выполняем функцию
		err := fn(ctx)
		if err == nil {
			// Успешно выполнено
			return nil
		}

		// Сохраняем последнюю ошибку
		lastErr = err

		// Если это была последняя попытка, возвращаем ошибку
		if attempt == policy.MaxAttempts-1 {
			break
		}

		// Рассчитываем задержку
		delay := backoff

		// Добавляем jitter если требуется
		if policy.Jitter {
			jitter := time.Duration(rand.Int63n(int64(backoff / 2)))
			delay += jitter
		}

		// Ограничиваем максимальную задержку
		if delay > policy.MaxBackoff {
			delay = policy.MaxBackoff
		}

		// Ждем перед следующей попыткой или пока контекст не будет отменен
		timer := time.NewTimer(delay)
		select {
		case <-timer.C:
			// Время задержки истекло, продолжаем
		case <-ctx.Done():
			// Контекст отменен
			timer.Stop()
			return ctx.Err()
		}
		timer.Stop()

		// Увеличиваем задержку для следующей попытки
		backoff = time.Duration(float64(backoff) * policy.BackoffFactor)
	}

	return lastErr
}

// IsRetryableError проверяет, является ли ошибка повторяемой
func IsRetryableError(err error) bool {
	// В реальной системе здесь можно было бы проверять конкретные типы ошибок
	// Например, сетевые ошибки, таймауты, временные ошибки БД и т.д.

	// Для простоты считаем, что любая ошибка может быть повторяемой
	// В production системе следует реализовать более точную проверку
	return err != nil
}
