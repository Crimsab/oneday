package main

import (
	"errors"
	"flag"
	"fmt"
	"io"

	"github.com/crimsab/oneday/internal/dockerbootstrap"
)

func runDockerCommand(args []string, out io.Writer) error {
	if len(args) == 0 {
		return errors.New("usage: oneday docker init|token [--root directory]")
	}
	switch args[0] {
	case "init":
		flags := flag.NewFlagSet("oneday docker init", flag.ContinueOnError)
		flags.SetOutput(out)
		root := flags.String("root", ".", "OneDay checkout containing the public configuration templates")
		if err := flags.Parse(args[1:]); err != nil {
			return err
		}
		if flags.NArg() != 0 {
			return errors.New("usage: oneday docker init [--root directory]")
		}
		result, err := dockerbootstrap.Prepare(*root)
		if err != nil {
			return err
		}
		if result.CreatedConfig {
			fmt.Fprintln(out, "Created private config.yaml from config.example.yaml.")
		}
		if result.CreatedEnv {
			fmt.Fprintln(out, "Created private .env from .env.example.")
		}
		if result.GeneratedToken {
			fmt.Fprintln(out, "Generated a private reusable browser bootstrap token.")
		}
		if !result.CreatedConfig && !result.CreatedEnv && !result.GeneratedToken && !result.UpdatedHostRules {
			fmt.Fprintln(out, "Existing OneDay Docker configuration is ready.")
		} else {
			fmt.Fprintln(out, "OneDay Docker configuration is ready.")
		}
		fmt.Fprintln(out, "Next: docker compose up -d")
		fmt.Fprintln(out, "When login asks for the credential:")
		fmt.Fprintln(out, "  docker compose run --rm oneday-tools docker token")
		return nil
	case "token":
		flags := flag.NewFlagSet("oneday docker token", flag.ContinueOnError)
		flags.SetOutput(out)
		root := flags.String("root", ".", "OneDay checkout containing the private .env")
		if err := flags.Parse(args[1:]); err != nil {
			return err
		}
		if flags.NArg() != 0 {
			return errors.New("usage: oneday docker token [--root directory]")
		}
		token, err := dockerbootstrap.BootstrapToken(*root)
		if err != nil {
			return err
		}
		fmt.Fprintln(out, token)
		return nil
	default:
		return fmt.Errorf("unknown docker command %q; use init or token", args[0])
	}
}
