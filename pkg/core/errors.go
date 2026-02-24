package core

import "fmt"

var (
	ErrUnsupportedSource    = fmt.Errorf("unsupported source")
	ErrUnsupportedFormat    = fmt.Errorf("unsupported format")
	ErrComplianceFramework  = fmt.Errorf("unsupported compliance framework")
	ErrMissingEvalFile      = fmt.Errorf("missing eval file")
	ErrSchemaValidationFail = fmt.Errorf("schema validation failed")
)

// Wrap provides consistent operation context for errors.
func Wrap(op string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", op, err)
}
