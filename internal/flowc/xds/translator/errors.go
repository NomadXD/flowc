package translator

import "fmt"

// Common errors
var (
	ErrInvalidConfig     = fmt.Errorf("invalid configuration")
	ErrMissingConfig     = fmt.Errorf("missing required configuration")
	ErrValidationFailed  = fmt.Errorf("validation failed")
	ErrTranslationFailed = fmt.Errorf("translation failed")
	ErrStrategyNotFound  = fmt.Errorf("strategy not found")
)

// ErrMissingStrategy returns an error for missing strategy
func ErrMissingStrategy(name string) error {
	return fmt.Errorf("missing required strategy: %s", name)
}

// ErrInvalidStrategyType returns an error for invalid strategy type
func ErrInvalidStrategyType(strategyKind, typeName string) error {
	return fmt.Errorf("invalid %s strategy type: %s", strategyKind, typeName)
}

// ErrStrategyConfigMissing returns an error for missing strategy configuration
func ErrStrategyConfigMissing(strategyKind string) error {
	return fmt.Errorf("%s strategy configuration is missing", strategyKind)
}
