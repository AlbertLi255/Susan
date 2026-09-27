package cmd

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/pkg/browser"
	"github.com/spf13/cobra"

	"github.com/ollama/ollama/api"
)

// loginPollInterval is how often the CLI polls the local daemon for the
// device flow result. This is a local call and does not trigger the platform's
// slow_down handling.
const loginPollInterval = 2 * time.Second

// LoginHandler implements `susan login`. The device flow itself runs inside
// the daemon; the CLI only talks to the local API.
func LoginHandler(cmd *cobra.Command, args []string) error {
	client, err := api.ClientFromEnvironment()
	if err != nil {
		return err
	}

	ctx := cmd.Context()
	force, err := cmd.Flags().GetBool("force")
	if err != nil {
		return err
	}

	if !force {
		if user, err := client.Whoami(ctx); err == nil && user != nil && user.Name != "" {
			fmt.Printf("Already signed in: %s (%s)\n", user.Name, user.Email)
			return nil
		}
	}

	resp, err := client.StartDeviceSignin(ctx, &api.DeviceSigninRequest{
		ClientName: "Susan CLI",
		Force:      force,
	})
	if err != nil {
		return err
	}

	if resp.VerificationURIComplete != "" {
		_ = browser.OpenURL(resp.VerificationURIComplete)
	}

	fmt.Println("Please complete authorization in your browser:")
	fmt.Printf("    %s\n", resp.VerificationURIComplete)
	fmt.Printf("Verification code: %s (valid for 15 minutes)\n", resp.UserCode)
	fmt.Println("If the browser did not open, visit the address above manually.")
	fmt.Println("Waiting for authorization... (Ctrl+C to cancel)")

	ticker := time.NewTicker(loginPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}

		status, err := client.DeviceSigninStatus(ctx)
		if err != nil {
			return err
		}

		switch api.DeviceSigninState(status.State) {
		case api.DeviceStateAuthorized:
			return printSignedIn(ctx, client)
		case api.DeviceStateDenied:
			return errors.New("authorization was denied")
		case api.DeviceStateExpired:
			return errors.New("the verification code has expired; please run `susan login` again. If this device was previously signed in to another account, remove it from that account's devices page first")
		case api.DeviceStateFailed:
			return fmt.Errorf("sign in failed: %s", status.Error)
		}
	}
}

func printSignedIn(ctx context.Context, client *api.Client) error {
	user, err := client.Whoami(ctx)
	if err != nil {
		return err
	}
	fmt.Printf("Signed in successfully: %s (%s)\n", user.Name, user.Email)
	return nil
}

// LogoutHandler implements `susan logout`.
func LogoutHandler(cmd *cobra.Command, args []string) error {
	client, err := api.ClientFromEnvironment()
	if err != nil {
		return err
	}

	revoke, err := cmd.Flags().GetBool("revoke")
	if err != nil {
		return err
	}

	if revoke {
		err = client.RevokeAndSignout(cmd.Context())
	} else {
		err = client.Signout(cmd.Context())
	}
	if err != nil {
		var aErr api.AuthorizationError
		if errors.As(err, &aErr) && aErr.StatusCode == 401 {
			return errors.New("you are not currently signed in")
		}
		return err
	}

	fmt.Println("Signed out. Local credentials have been deleted.")
	if !revoke {
		fmt.Println("To also disable this device on the Susan platform, remove it from your account's devices page.")
	}
	return nil
}
