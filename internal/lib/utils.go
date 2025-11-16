package lib

import "fmt"

func Err(op string, err error) error {
	return fmt.Errorf("%s: %w", op, err)
}
