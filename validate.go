package fstr

import (
	"errors"
)

var (
	ErrUnexpectedClosingBrace = errors.New("unexpected closing brace")
	ErrUnclosedBrace          = errors.New("unclosed brace")
)

func Validate(format string) error {
	stack := []rune{}
	for pos, ch := range format {
		switch ch {
		case '{':
			stack = append(stack, ch)
		case '}':
			if len(stack) == 0 {
				return &FormatError{
					Format: format,
					Err:    ErrUnexpectedClosingBrace,
					Pos:    pos,
				}
			}
			stack = stack[:len(stack)-1]
		}
	}

	if len(stack) > 0 {
		return &FormatError{
			Format: format,
			Err:    ErrUnclosedBrace,
			Pos:    -1,
		}
	}
	return nil
}
