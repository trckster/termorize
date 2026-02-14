package validators

import (
	"reflect"
	"time"

	"github.com/go-playground/validator/v10"
)

func ValidateTimezone(fl validator.FieldLevel) bool {
	field := fl.Field()

	if field.Kind() != reflect.String {
		return false
	}

	_, err := time.LoadLocation(field.String())

	return err == nil
}

func ValidateHHMM(fl validator.FieldLevel) bool {
	field := fl.Field()

	if field.Kind() != reflect.String {
		return false
	}

	_, err := time.Parse("15:04", field.String())

	return err == nil
}
