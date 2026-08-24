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

// ValidateHHMMInterval is a struct-level validator for schedule items with
// From/To HH:MM fields: the interval must end strictly after it starts.
// Format problems are reported by the hhmm tag and are ignored here.
func ValidateHHMMInterval(sl validator.StructLevel) {
	current := sl.Current()

	from := current.FieldByName("From")
	to := current.FieldByName("To")
	if !from.IsValid() || !to.IsValid() || from.Kind() != reflect.String || to.Kind() != reflect.String {
		return
	}

	fromTime, fromErr := time.Parse("15:04", from.String())
	toTime, toErr := time.Parse("15:04", to.String())
	if fromErr != nil || toErr != nil {
		return
	}

	if !toTime.After(fromTime) {
		sl.ReportError(to.String(), "To", "To", "hhmminterval", "")
	}
}
