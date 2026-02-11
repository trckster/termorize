package validators

import (
	"reflect"
	"termorize/src/enums"

	"github.com/go-playground/validator/v10"
)

func ValidateEnum(fl validator.FieldLevel) bool {
	param := fl.Param()
	field := fl.Field()

	if field.Kind() != reflect.String {
		return false
	}

	value := field.String()

	enumFuncs := map[string]func() []string{
		"Language": enums.AllLanguages,
	}

	fn, ok := enumFuncs[param]
	if !ok {
		return false // unknown enum
	}

	for _, v := range fn() {
		if v == value {
			return true
		}
	}

	return false
}
