package fstr

import (
	"fmt"
)

type FormatError struct {
	Format string
	Err    error
	Pos    int
}

func (e *FormatError) Error() string {
	if e.Pos >= 0 {
		return fmt.Sprintf("format error at position %d: %v", e.Pos, e.Err)
	}
	return fmt.Sprintf("format error: %v", e.Err)
}
