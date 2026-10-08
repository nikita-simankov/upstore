package env

import (
	"os"
	"strconv"
)

type value interface {
	int | bool | string
}

func Env[T value](key string, fallback T) T {
	raw := os.Getenv(key)

	if raw == "" {
		return fallback
	}

	switch any(fallback).(type) {
	case int:
		value, err := strconv.Atoi(raw)

		if err != nil {
			return fallback
		}

		return any(value).(T)
	case bool:
		value, err := strconv.ParseBool(raw)

		if err != nil {
			return fallback
		}

		return any(value).(T)
	case string:
		return any(raw).(T)
	}

	return fallback
}
