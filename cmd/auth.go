package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/czekaj/statcounter-cli/internal/output"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func newLoginCmd() *cobra.Command {
	var username, password string
	var noRemember bool
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Sign in to StatCounter and save the session",
		Long: `Sign in to the StatCounter web dashboard and persist the session locally
(~/.config/statcounter/session.json, mode 0600).

Credentials are read from flags, then the STATCOUNTER_USERNAME / STATCOUNTER_PASSWORD
environment variables, then interactive prompts. The password is never stored —
only the resulting session cookies are.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if username == "" {
				username = os.Getenv("STATCOUNTER_USERNAME")
			}
			if password == "" {
				password = os.Getenv("STATCOUNTER_PASSWORD")
			}
			if username == "" {
				v, err := prompt("StatCounter username: ")
				if err != nil {
					return err
				}
				username = v
			}
			if password == "" {
				v, err := promptPassword("StatCounter password: ")
				if err != nil {
					return err
				}
				password = v
			}
			if username == "" || password == "" {
				return fmt.Errorf("username and password are required")
			}

			svc, err := newService()
			if err != nil {
				return err
			}
			if err := svc.Login(username, password, !noRemember); err != nil {
				return err
			}
			if outFormat == output.FormatJSON {
				return output.JSON(cmd.OutOrStdout(), map[string]string{"status": "ok", "user": username})
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Logged in as %s\n", username)
			return nil
		},
	}
	cmd.Flags().StringVarP(&username, "username", "u", "", "StatCounter username (env STATCOUNTER_USERNAME)")
	cmd.Flags().StringVar(&password, "password", "", "StatCounter password (prefer the prompt or env STATCOUNTER_PASSWORD)")
	cmd.Flags().BoolVar(&noRemember, "no-remember", false, "do not request a long-lived session")
	return cmd
}

func newLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Remove the saved session",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			svc, err := newService()
			if err != nil {
				return err
			}
			if err := svc.Logout(); err != nil {
				return err
			}
			if outFormat == output.FormatJSON {
				return output.JSON(cmd.OutOrStdout(), map[string]string{"status": "ok"})
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Logged out.")
			return nil
		},
	}
}

func newWhoamiCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "Show the current session status",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			svc, err := newService()
			if err != nil {
				return err
			}
			ok, err := svc.CheckAuth()
			if err != nil {
				return err
			}
			data := map[string]any{"authenticated": ok, "user": svc.Username()}
			if outFormat == output.FormatJSON {
				return output.JSON(cmd.OutOrStdout(), data)
			}
			if ok {
				u := svc.Username()
				if u == "" {
					u = "(unknown)"
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Authenticated as %s\n", u)
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), "Not authenticated.")
			}
			return nil
		},
	}
}

func prompt(label string) (string, error) {
	fmt.Fprint(os.Stderr, label)
	r := bufio.NewReader(os.Stdin)
	line, err := r.ReadString('\n')
	if err != nil && line == "" {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

func promptPassword(label string) (string, error) {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return "", fmt.Errorf("cannot prompt for password: stdin is not a terminal (use STATCOUNTER_PASSWORD)")
	}
	fmt.Fprint(os.Stderr, label)
	b, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}
