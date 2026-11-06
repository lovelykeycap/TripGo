package config

import (
	"errors"
	"fmt"
	"math"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

func validateTelemetryEnv() error {
	if _, err := requiredEnv("OTEL_SERVICE_NAME"); err != nil {
		return err
	}
	if err := validateOTLPEndpoint("OTEL_EXPORTER_OTLP_ENDPOINT", true); err != nil {
		return err
	}
	for _, name := range []string{"OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", "OTEL_EXPORTER_OTLP_METRICS_ENDPOINT"} {
		if err := validateOTLPEndpoint(name, false); err != nil {
			return err
		}
	}
	for _, name := range []string{
		"OTEL_EXPORTER_OTLP_PROTOCOL", "OTEL_EXPORTER_OTLP_TRACES_PROTOCOL", "OTEL_EXPORTER_OTLP_METRICS_PROTOCOL",
	} {
		value := os.Getenv(name)
		if value == "" && name != "OTEL_EXPORTER_OTLP_PROTOCOL" {
			continue
		}
		if value != "http/protobuf" {
			return fmt.Errorf("%s must be http/protobuf: only OTLP/HTTP protobuf exporters are configured", name)
		}
	}
	if err := validateResourceAttributes(); err != nil {
		return err
	}
	if err := validateSampler(); err != nil {
		return err
	}
	for _, name := range []string{
		"OTEL_EXPORTER_OTLP_TIMEOUT", "OTEL_EXPORTER_OTLP_TRACES_TIMEOUT", "OTEL_EXPORTER_OTLP_METRICS_TIMEOUT",
		"OTEL_METRIC_EXPORT_INTERVAL", "OTEL_METRIC_EXPORT_TIMEOUT",
	} {
		value := os.Getenv(name)
		if value == "" {
			continue
		}
		milliseconds, err := strconv.ParseInt(value, 10, 64)
		if err != nil || milliseconds <= 0 || milliseconds > math.MaxInt64/int64(time.Millisecond) {
			return fmt.Errorf("%s must be a positive integer duration in milliseconds within the time.Duration range", name)
		}
	}
	return nil
}

func validateOTLPEndpoint(name string, required bool) error {
	value := os.Getenv(name)
	if value == "" {
		if required {
			return fmt.Errorf("%s is required", name)
		}
		return nil
	}
	endpoint, err := url.Parse(value)
	if err != nil || (endpoint.Scheme != "http" && endpoint.Scheme != "https") || endpoint.Hostname() == "" {
		return fmt.Errorf("%s must be an absolute http:// or https:// URL with a host", name)
	}
	if endpoint.RawQuery != "" || endpoint.ForceQuery || endpoint.Fragment != "" {
		return fmt.Errorf("%s must not contain a query string or fragment", name)
	}
	if port := endpoint.Port(); port != "" {
		parsed, err := strconv.ParseUint(port, 10, 16)
		if err != nil || parsed == 0 {
			return fmt.Errorf("%s port must be between 1 and 65535", name)
		}
	}
	return nil
}

func validateResourceAttributes() error {
	value, err := requiredEnv("OTEL_RESOURCE_ATTRIBUTES")
	if err != nil {
		return err
	}
	attrs := make(map[string]string)
	for _, pair := range strings.Split(value, ",") {
		key, raw, ok := strings.Cut(pair, "=")
		key = strings.TrimSpace(key)
		if !ok || key == "" {
			return errors.New("OTEL_RESOURCE_ATTRIBUTES must contain comma-separated nonempty key=value pairs")
		}
		decoded, err := url.PathUnescape(strings.TrimSpace(raw))
		if err != nil {
			return fmt.Errorf("OTEL_RESOURCE_ATTRIBUTES has an invalid escaped value for %s", key)
		}
		attrs[key] = decoded
	}
	for _, key := range []string{"service.namespace", "deployment.environment"} {
		if strings.TrimSpace(attrs[key]) == "" {
			return fmt.Errorf("OTEL_RESOURCE_ATTRIBUTES must include a nonempty %s", key)
		}
	}
	return nil
}

func validateSampler() error {
	value, err := requiredEnv("OTEL_TRACES_SAMPLER")
	if err != nil {
		return err
	}
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "always_on", "always_off", "parentbased_always_on", "parentbased_always_off":
		return nil
	case "traceidratio", "parentbased_traceidratio":
		argument, err := requiredEnv("OTEL_TRACES_SAMPLER_ARG")
		if err != nil {
			return err
		}
		ratio, err := strconv.ParseFloat(strings.TrimSpace(argument), 64)
		if err != nil || !(ratio >= 0 && ratio <= 1) {
			return errors.New("OTEL_TRACES_SAMPLER_ARG must be a finite number between 0 and 1")
		}
		return nil
	default:
		return errors.New("OTEL_TRACES_SAMPLER must be always_on, always_off, parentbased_always_on, parentbased_always_off, traceidratio or parentbased_traceidratio")
	}
}
