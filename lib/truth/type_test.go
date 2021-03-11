package truth

import (
	"fmt"
	"testing"
)

func TestIsOfType(t *testing.T) {
	testEach(t, map[string]error{
		`that(None).is_of_type("NoneType")`:            nil,
		`that("").is_of_type("string")`:                nil,
		`that(0).is_of_type("int")`:                    nil,
		`that(0.0).is_of_type("float")`:                nil,
		`that(set([])).is_of_type("set")`:              nil,
		`that(someCmp).is_of_type(type(lambda _: 42))`: nil,
		`that([]).is_of_type("list")`:                  nil,
		`that(()).is_of_type("tuple")`:                 nil,
		`that({}).is_of_type("dict")`:                  nil,
		`that({}).is_of_type("int")`:                   fail(`{}`, `is of type <"int">`, ` However, it is of type <"dict">`),
		`that({}).is_of_type(type({}))`:                nil,
		`that({}).is_of_type({})`:                      fmt.Errorf(`Invalid assertion .is_of_type({}) on value of type dict`),
	})
}

func TestIsNotOfType(t *testing.T) {
	testEach(t, map[string]error{
		`that(None).is_not_of_type("int")`:    nil,
		`that("").is_not_of_type("int")`:      nil,
		`that(0).is_not_of_type("dict")`:      nil,
		`that(0.0).is_not_of_type("int")`:     nil,
		`that(set([])).is_not_of_type("int")`: nil,
		`that(someCmp).is_not_of_type("int")`: nil,
		`that([]).is_not_of_type("int")`:      nil,
		`that(()).is_not_of_type("int")`:      nil,
		`that({}).is_not_of_type("int")`:      nil,
		`that({}).is_not_of_type("dict")`:     fail(`{}`, `is not of type <"dict">`, ` However, it is of type <"dict">`),
		`that({}).is_not_of_type(type([{}]))`: nil,
		`that([]).is_not_of_type({})`:         fmt.Errorf(`Invalid assertion .is_not_of_type({}) on value of type list`),
	})
}
