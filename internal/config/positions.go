package config

import "fmt"

func (cfg *Config) loadPositionConfig() error {
	var err error
	cfg.PositionCheckInterval, err = durationEnv("POSITION_CHECK_INTERVAL")
	if err != nil {
		return err
	}
	cfg.PositionStaleAfter, err = durationEnv("POSITION_STALE_AFTER")
	if err != nil {
		return err
	}
	cfg.PositionWorkers, err = positiveInt32Env("POSITION_WORKERS")
	if err != nil {
		return err
	}
	cfg.PositionQueueSize, err = positiveInt32Env("POSITION_QUEUE_SIZE")
	if err != nil {
		return err
	}
	cfg.PushHTTPURL, err = requiredEnv("PUSH_HTTP_URL")
	if err != nil {
		return err
	}
	cfg.PushRequestTimeout, err = durationEnv("PUSH_REQUEST_TIMEOUT")
	return err
}

func positiveInt32Env(name string) (int32, error) {
	value, err := int32Env(name)
	if err != nil {
		return 0, err
	}
	if value <= 0 {
		return 0, fmt.Errorf("%s must be greater than zero", name)
	}
	return value, nil
}
