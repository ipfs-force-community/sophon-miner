package cli

import (
	"errors"
	"fmt"
	"os"

	ufcli "github.com/urfave/cli/v2"
)

type PrintHelpErr struct {
	Err error
	Ctx *ufcli.Context
}

func (e *PrintHelpErr) Error() string {
	return e.Err.Error()
}

func (e *PrintHelpErr) Unwrap() error {
	return e.Err
}

func (e *PrintHelpErr) Is(o error) bool {
	_, ok := o.(*PrintHelpErr)
	return ok
}

func RunApp(app *ufcli.App) {
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n\n", err) // nolint:errcheck
		var phe *PrintHelpErr
		if errors.As(err, &phe) {
			_ = ufcli.ShowCommandHelp(phe.Ctx, phe.Ctx.Command.Name)
		}
		os.Exit(1)
	}
}
