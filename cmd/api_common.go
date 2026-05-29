package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/ds-cli/ds-cli/internal/dsapi"
	"github.com/ds-cli/ds-cli/internal/output"
	"github.com/spf13/cobra"
)

type apiFlags struct {
	Cluster   string
	APIURL    string
	Username  string
	Password  string
	Token     string
	SessionID string
	Timeout   time.Duration
}

func addAPIFlags(cmd *cobra.Command, f *apiFlags) {
	cmd.PersistentFlags().StringVar(&f.Cluster, "cluster", "", "named DolphinScheduler API cluster profile")
	cmd.PersistentFlags().StringVar(&f.APIURL, "api-url", "", "DolphinScheduler API base URL, e.g. http://localhost:12345/dolphinscheduler")
	cmd.PersistentFlags().StringVar(&f.Username, "user", "", "DolphinScheduler username")
	cmd.PersistentFlags().StringVar(&f.Password, "password", "", "DolphinScheduler password")
	cmd.PersistentFlags().StringVar(&f.Token, "token", "", "DolphinScheduler access token")
	cmd.PersistentFlags().StringVar(&f.SessionID, "session-id", "", "DolphinScheduler sessionId")
	cmd.PersistentFlags().DurationVar(&f.Timeout, "api-timeout", 0, "DolphinScheduler API timeout, e.g. 30s")
}

func apiClient(flags apiFlags) (*dsapi.Client, dsapi.Profile, error) {
	profile, err := dsapi.ResolveProfile("", dsapi.APIOverrides{
		Cluster:   flags.Cluster,
		APIURL:    flags.APIURL,
		Username:  flags.Username,
		Password:  flags.Password,
		Token:     flags.Token,
		SessionID: flags.SessionID,
		Timeout:   flags.Timeout,
	})
	if err != nil {
		return nil, dsapi.Profile{}, err
	}
	return dsapi.NewClient(profile), profile, nil
}

func writeDataEnvelope(cmd *cobra.Command, command string, data any) error {
	e := output.NewEnvelope(command)
	e.Data = data
	return e.Write(cmd.OutOrStdout())
}

func writeAPIResponse(cmd *cobra.Command, command string, profile dsapi.Profile, resp *dsapi.Response) error {
	var body any
	if len(resp.Body) > 0 {
		if err := json.Unmarshal(resp.Body, &body); err != nil {
			body = string(resp.Body)
		}
	}
	e := output.NewEnvelope(command)
	e.Summary = map[string]any{
		"cluster":     profile.Name,
		"api_url":     profile.APIURL,
		"http_status": resp.HTTPStatus,
	}
	e.Data = body
	return e.Write(cmd.OutOrStdout())
}

func apiRun(cmd *cobra.Command, flags apiFlags, command string, run func(context.Context, *dsapi.Client) (*dsapi.Response, error)) error {
	client, profile, err := apiClient(flags)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), profile.Timeout)
	defer cancel()
	resp, err := run(ctx, client)
	if err != nil {
		e := output.NewEnvelope(command).WithError(output.EnvelopeError{Code: "DS_API_ERROR", Message: err.Error()})
		_ = e.Write(cmd.OutOrStdout())
		return err
	}
	return writeAPIResponse(cmd, command, profile, resp)
}

func formValues(kv ...string) url.Values {
	values := url.Values{}
	for i := 0; i+1 < len(kv); i += 2 {
		if kv[i+1] != "" {
			values.Set(kv[i], kv[i+1])
		}
	}
	return values
}

func int64Arg(s, name string) (int64, error) {
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid %s %q: %w", name, s, err)
	}
	return v, nil
}

func intArg(s, name string) (int, error) {
	v, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("invalid %s %q: %w", name, s, err)
	}
	return v, nil
}

func releaseWorkflow(ctx context.Context, client *dsapi.Client, projectCode, code int64, state string) (*dsapi.Response, error) {
	return client.Form(ctx, http.MethodPost,
		fmt.Sprintf("/projects/%d/workflow-definition/%d/release", projectCode, code),
		formValues("releaseState", state),
	)
}
