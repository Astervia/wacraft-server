package validators

import (
	validators "github.com/Rfluid/whatsapp-cloud-api/src/validators"
	"github.com/go-playground/validator/v10"
)

var validate *validator.Validate

func InitValidators() {
	validate = validator.New()

	validators.RegisterAllValidators(validate)
}

// Export validate to use in handlers
func Validator() *validator.Validate {
	return validate
}
