// Package retry_test тестирует механизмы повторных попыток
package retry

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultPolicy(t *testing.T) {
	policy := DefaultPolicy()

	assert.Equal(t, 3, policy.MaxAttempts)
	assert.Equal(t, 100*time.Millisecond, policy.InitialBackoff)
	assert.Equal(t, 10*time.Second, policy.MaxBackoff)
	assert.Equal(t, 2.0, policy.BackoffFactor)
	assert.True(t, policy.Jitter)
}

func TestLightPolicy(t *testing.T) {
	policy := LightPolicy()

	assert.Equal(t, 2, policy.MaxAttempts)
	assert.Equal(t, 50*time.Millisecond, policy.InitialBackoff)
	assert.Equal(t, 1*time.Second, policy.MaxBackoff)
	assert.Equal(t, 1.5, policy.BackoffFactor)
	assert.True(t, policy.Jitter)
}

func TestHeavyPolicy(t *testing.T) {
	policy := HeavyPolicy()

	assert.Equal(t, 5, policy.MaxAttempts)
	assert.Equal(t, 200*time.Millisecond, policy.InitialBackoff)
	assert.Equal(t, 30*time.Second, policy.MaxBackoff)
	assert.Equal(t, 2.5, policy.BackoffFactor)
	assert.True(t, policy.Jitter)
}

func TestSuccessfulRetry(t *testing.T) {
	attempts := 0
	successAtAttempt := 2

	fn := func() error {
		attempts++
		if attempts < successAtAttempt {
			return errors.New("temporary error")
		}
		return nil
	}

	policy := Policy{
		MaxAttempts:    3,
		InitialBackoff: 10 * time.Millisecond,
		MaxBackoff:     100 * time.Millisecond,
		BackoffFactor:  2.0,
		Jitter:         false,
	}

	err := Do(policy, fn)

	assert.NoError(t, err)
	assert.Equal(t, successAtAttempt, attempts)
}

func TestFailedRetry(t *testing.T) {
	attempts := 0

	fn := func() error {
		attempts++
		return errors.New("permanent error")
	}

	policy := Policy{
		MaxAttempts:    3,
		InitialBackoff: 10 * time.Millisecond,
		MaxBackoff:     100 * time.Millisecond,
		BackoffFactor:  2.0,
		Jitter:         false,
	}

	err := Do(policy, fn)

	assert.Error(t, err)
	assert.Equal(t, 3, attempts)
	assert.Equal(t, "permanent error", err.Error())
}

func TestContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	// Отменяем контекст немедленно
	cancel()

	fn := func(ctx context.Context) error {
		return errors.New("should not reach here")
	}

	policy := Policy{
		MaxAttempts:    3,
		InitialBackoff: 10 * time.Millisecond,
		MaxBackoff:     100 * time.Millisecond,
		BackoffFactor:  2.0,
		Jitter:         false,
	}

	err := DoWithContext(ctx, policy, fn)

	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

func TestBackoffCalculation(t *testing.T) {
	attempts := 0

	fn := func() error {
		attempts++
		if attempts < 3 {
			return errors.New("temporary error")
		}
		return nil
	}

	policy := Policy{
		MaxAttempts:    5,
		InitialBackoff: 10 * time.Millisecond,
		MaxBackoff:     100 * time.Millisecond,
		BackoffFactor:  2.0,
		Jitter:         false,
	}

	start := time.Now()
	err := Do(policy, fn)
	duration := time.Since(start)

	assert.NoError(t, err)
	assert.Equal(t, 3, attempts)

	// Проверяем, что общее время выполнения соответствует ожиданиям
	// Первая попытка: 10ms задержка
	// Вторая попытка: 20ms задержка
	// Третья попытка: успех
	// Общее время: ~30ms + выполнение
	expectedMin := 30 * time.Millisecond
	assert.True(t, duration >= expectedMin, "Duration should be at least %v, got %v", expectedMin, duration)
}

func TestJitterEffect(t *testing.T) {
	attempts := 0

	fn := func() error {
		attempts++
		if attempts < 2 {
			return errors.New("temporary error")
		}
		return nil
	}

	policyWithJitter := Policy{
		MaxAttempts:    2,
		InitialBackoff: 50 * time.Millisecond,
		MaxBackoff:     200 * time.Millisecond,
		BackoffFactor:  1.0,
		Jitter:         true,
	}

	policyWithoutJitter := Policy{
		MaxAttempts:    2,
		InitialBackoff: 50 * time.Millisecond,
		MaxBackoff:     200 * time.Millisecond,
		BackoffFactor:  1.0,
		Jitter:         false,
	}

	// Выполняем оба теста и проверяем, что они работают
	start1 := time.Now()
	err1 := Do(policyWithJitter, fn)
	duration1 := time.Since(start1)

	require.NoError(t, err1)

	// Сбрасываем счетчик
	attempts = 0

	start2 := time.Now()
	err2 := Do(policyWithoutJitter, fn)
	duration2 := time.Since(start2)

	require.NoError(t, err2)

	// Оба должны завершиться успешно, но время может немного отличаться из-за jitter
	assert.True(t, duration1 > 0)
	assert.True(t, duration2 > 0)
}

func TestZeroAttemptsPolicy(t *testing.T) {
	policy := Policy{
		MaxAttempts:    0,
		InitialBackoff: 10 * time.Millisecond,
		MaxBackoff:     100 * time.Millisecond,
		BackoffFactor:  2.0,
		Jitter:         false,
	}

	attempts := 0
	fn := func() error {
		attempts++
		return errors.New("error")
	}

	err := Do(policy, fn)

	assert.Error(t, err)
	assert.Equal(t, 1, attempts) // Должна быть только одна попытка
}

func TestImmediateSuccess(t *testing.T) {
	attempts := 0

	fn := func() error {
		attempts++
		return nil // Успешно с первой попытки
	}

	policy := DefaultPolicy()

	err := Do(policy, fn)

	assert.NoError(t, err)
	assert.Equal(t, 1, attempts)
}

func TestMaxBackoffLimit(t *testing.T) {
	attempts := 0

	fn := func() error {
		attempts++
		if attempts < 4 {
			return errors.New("temporary error")
		}
		return nil
	}

	// Политика с маленькой начальной задержкой, но большим фактором
	// Это проверит, что максимальная задержка ограничена
	policy := Policy{
		MaxAttempts:    4,
		InitialBackoff: 10 * time.Millisecond,
		MaxBackoff:     50 * time.Millisecond, // Максимум 50ms
		BackoffFactor:  10.0,
		Jitter:         false,
	}

	start := time.Now()
	err := Do(policy, fn)
	duration := time.Since(start)

	assert.NoError(t, err)
	assert.Equal(t, 4, attempts)

	// Даже с большим фактором роста, общее время не должно быть слишком большим
	// из-за ограничения maxBackoff
	assert.True(t, duration < 1*time.Second, "Duration should be reasonable, got %v", duration)
}
