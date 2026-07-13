package utils

import (
	"net/http"

	"github.com/adhocore/gronx"
	"github.com/go-playground/validator/v10"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
)

var validate *validator.Validate

type ValidationErrors = validator.ValidationErrors

func Validator() *validator.Validate {

	if validate == nil {
		validate = validator.New()

		customValidations := map[string]validator.Func{"cron": validateCron, "httpMethod": validateHttpMethod}

		for tagName, vf := range customValidations {
			err := validate.RegisterValidation(tagName, vf)

			if err != nil {
				slog.Errorf("Could not initialize validator %s", tagName)
			}
		}

	}

	return validate

}

func validateCron(fl validator.FieldLevel) bool {
	cronExp := fl.Field().String()

	gron := gronx.New()
	return gron.IsValid(cronExp)
}

func validateHttpMethod(fl validator.FieldLevel) bool {
	method := fl.Field().String()

	methods := []string{http.MethodDelete, http.MethodGet, http.MethodPost}

	included := false
	for _, v := range methods {
		if v == method {
			included = true
		}
	}

	return included
}
