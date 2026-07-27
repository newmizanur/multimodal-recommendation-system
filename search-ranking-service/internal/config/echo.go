package config

import (
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type requestValidator struct {
	validate *validator.Validate
}

func (rv *requestValidator) Validate(i any) error {
	return rv.validate.Struct(i)
}

// NewEcho wires the global middleware every request goes through and
// attaches struct-tag validation so controllers can just call c.Bind +
// c.Validate and let apperror.Validation handle the rest.
func NewEcho() *echo.Echo {
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.RequestID())
	e.Validator = &requestValidator{validate: validator.New()}
	return e
}
