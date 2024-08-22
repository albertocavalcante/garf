package enumflag

import "fmt"

type enumflag struct {
	target  *string
	options []string
}

func New(target *string, options ...string) *enumflag {
	return &enumflag{target: target, options: options}
}

func (f *enumflag) String() string {
	return *f.target
}

func (f *enumflag) Set(value string) error {
	for _, v := range f.options {
		if v == value {
			*f.target = value

			return nil
		}
	}

	return fmt.Errorf("expected one of %q, got %s", f.options, value)
}

func (f *enumflag) Type() string {
	return "enumflag"
}
