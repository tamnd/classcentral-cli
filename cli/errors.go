package cli

import (
	"errors"

	"github.com/tamnd/classcentral-cli/classcentral"
)

func isBlocked(err error) bool {
	return errors.Is(err, classcentral.ErrBlocked)
}

func isNotFound(err error) bool {
	return errors.Is(err, classcentral.ErrNotFound)
}
