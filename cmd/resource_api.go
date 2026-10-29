package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/ds-cli/ds-cli/internal/dsapi"
	"github.com/ds-cli/ds-cli/internal/output"
	"github.com/spf13/cobra"
)

func newResourceCmd() *cobra.Command {
	var flags apiFlags
	cmd := &cobra.Command{Use: "resource", Aliases: []string{"res"}, Short: "Manage DolphinScheduler resources."}
	addAPIFlags(cmd, &flags)
	cmd.AddCommand(newResourceBaseDirCmd(&flags), newResourceTreeCmd(&flags), newResourceMkdirCmd(&flags), newResourceListCmd(&flags), newResourceCreateContentCmd(&flags), newResourceUploadCmd(&flags), newResourceUpdateContentCmd(&flags), newResourceViewCmd(&flags), newResourceDownloadCmd(&flags), newResourceDeleteCmd(&flags))
	return cmd
}

func newResourceTreeCmd(flags *apiFlags) *cobra.Command {
	var resourceType string
	c := &cobra.Command{Use: "tree", Short: "List the resource tree for the current user.", RunE: func(cmd *cobra.Command, args []string) error {
		return apiRun(cmd, *flags, "resource.tree", func(ctx context.Context, client *dsapi.Client) (*dsapi.Response, error) {
			return client.Form(ctx, http.MethodGet, "/resources/list", formValues("type", resourceType))
		})
	}}
	c.Flags().StringVar(&resourceType, "type", "FILE", "Resource type: FILE or UDF")
	return c
}

func newResourceBaseDirCmd(flags *apiFlags) *cobra.Command {
	var resourceType string
	c := &cobra.Command{Use: "base-dir", Short: "Get the current user's resource base directory.", RunE: func(cmd *cobra.Command, args []string) error {
		return apiRun(cmd, *flags, "resource.base-dir", func(ctx context.Context, client *dsapi.Client) (*dsapi.Response, error) {
			return client.Form(ctx, http.MethodGet, "/resources/base-dir", formValues("type", resourceType))
		})
	}}
	c.Flags().StringVar(&resourceType, "type", "FILE", "Resource type: FILE or UDF")
	return c
}

func newResourceMkdirCmd(flags *apiFlags) *cobra.Command {
	var resourceType, currentDir string
	c := &cobra.Command{Use: "mkdir <name>", Short: "Create a resource directory.", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		if currentDir == "" {
			return fmt.Errorf("--current-dir is required")
		}
		return apiRun(cmd, *flags, "resource.mkdir", func(ctx context.Context, client *dsapi.Client) (*dsapi.Response, error) {
			return client.Form(ctx, http.MethodPost, "/resources/directory", formValues("type", resourceType, "name", args[0], "currentDir", currentDir))
		})
	}}
	c.Flags().StringVar(&resourceType, "type", "FILE", "Resource type")
	c.Flags().StringVar(&currentDir, "current-dir", "", "Parent resource directory absolute path")
	return c
}

func newResourceListCmd(flags *apiFlags) *cobra.Command {
	var resourceType, search, currentDir string
	var pageNo, pageSize int
	c := &cobra.Command{Use: "list", Short: "List resources in a directory.", RunE: func(cmd *cobra.Command, args []string) error {
		if currentDir == "" {
			return fmt.Errorf("--current-dir is required by DolphinScheduler")
		}
		return apiRun(cmd, *flags, "resource.list", func(ctx context.Context, client *dsapi.Client) (*dsapi.Response, error) {
			return client.Form(ctx, http.MethodGet, "/resources", formValues("type", resourceType, "fullName", currentDir, "searchVal", search, "pageNo", strconv.Itoa(pageNo), "pageSize", strconv.Itoa(pageSize)))
		})
	}}
	c.Flags().StringVar(&resourceType, "type", "FILE", "Resource type: FILE or UDF")
	c.Flags().StringVar(&currentDir, "current-dir", "", "Resource directory absolute path")
	c.Flags().StringVar(&search, "search", "", "Search keyword")
	c.Flags().IntVar(&pageNo, "page-no", 1, "Page number")
	c.Flags().IntVar(&pageSize, "page-size", 20, "Page size")
	return c
}

func newResourceCreateContentCmd(flags *apiFlags) *cobra.Command {
	var resourceType, suffix, currentDir, contentFile string
	c := &cobra.Command{Use: "create-content <file-name>", Short: "Create a resource from text content.", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		if contentFile == "" {
			return fmt.Errorf("--content-file is required")
		}
		if currentDir == "" {
			return fmt.Errorf("--current-dir is required")
		}
		content, err := os.ReadFile(contentFile)
		if err != nil {
			return err
		}
		return apiRun(cmd, *flags, "resource.create-content", func(ctx context.Context, client *dsapi.Client) (*dsapi.Response, error) {
			return client.Form(ctx, http.MethodPost, "/resources/online-create", formValues("type", resourceType, "fileName", args[0], "suffix", suffix, "content", string(content), "currentDir", currentDir))
		})
	}}
	c.Flags().StringVar(&resourceType, "type", "FILE", "Resource type")
	c.Flags().StringVar(&suffix, "suffix", "sh", "File suffix without dot")
	c.Flags().StringVar(&currentDir, "current-dir", "", "Parent resource directory absolute path")
	c.Flags().StringVar(&contentFile, "content-file", "", "Local content file")
	return c
}

func newResourceUploadCmd(flags *apiFlags) *cobra.Command {
	var resourceType, currentDir, name string
	c := &cobra.Command{Use: "upload <local-file>", Short: "Upload a local file into the resource center.", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		if currentDir == "" {
			return fmt.Errorf("--current-dir is required")
		}
		if name == "" {
			name = filepath.Base(args[0])
		}
		return apiRun(cmd, *flags, "resource.upload", func(ctx context.Context, client *dsapi.Client) (*dsapi.Response, error) {
			return client.MultipartFile(ctx, http.MethodPost, "/resources", formValues("type", resourceType, "name", name, "currentDir", currentDir), "file", args[0])
		})
	}}
	c.Flags().StringVar(&resourceType, "type", "FILE", "Resource type")
	c.Flags().StringVar(&currentDir, "current-dir", "", "Parent resource directory absolute path")
	c.Flags().StringVar(&name, "name", "", "Remote file name; defaults to the local basename")
	return c
}

func newResourceViewCmd(flags *apiFlags) *cobra.Command {
	var skip, limit int
	c := &cobra.Command{Use: "view <full-name>", Short: "View resource text content.", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		return apiRun(cmd, *flags, "resource.view", func(ctx context.Context, client *dsapi.Client) (*dsapi.Response, error) {
			return client.Form(ctx, http.MethodGet, "/resources/view", formValues("fullName", args[0], "skipLineNum", strconv.Itoa(skip), "limit", strconv.Itoa(limit)))
		})
	}}
	c.Flags().IntVar(&skip, "skip-line", 0, "Lines to skip")
	c.Flags().IntVar(&limit, "limit", -1, "Maximum lines; -1 reads all lines")
	return c
}

func newResourceUpdateContentCmd(flags *apiFlags) *cobra.Command {
	var contentFile string
	c := &cobra.Command{Use: "update-content <full-name>", Short: "Replace a text resource from a local file.", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		if contentFile == "" {
			return fmt.Errorf("--content-file is required")
		}
		content, err := os.ReadFile(contentFile)
		if err != nil {
			return err
		}
		return apiRun(cmd, *flags, "resource.update-content", func(ctx context.Context, client *dsapi.Client) (*dsapi.Response, error) {
			return client.Form(ctx, http.MethodPut, "/resources/update-content", formValues("fullName", args[0], "content", string(content)))
		})
	}}
	c.Flags().StringVar(&contentFile, "content-file", "", "Local replacement content file")
	return c
}

func newResourceDownloadCmd(flags *apiFlags) *cobra.Command {
	var outputFile string
	c := &cobra.Command{Use: "download <full-name>", Short: "Download a resource to a local file.", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		if outputFile == "" {
			return fmt.Errorf("--output is required")
		}
		client, profile, err := apiClient(*flags)
		if err != nil {
			writeAPIError(cmd, "resource.download", "CONFIG_ERROR", err)
			return err
		}
		ctx, cancel := context.WithTimeout(context.Background(), profile.Timeout)
		defer cancel()
		resp, err := client.RawGet(ctx, "/resources/download", formValues("fullName", args[0]))
		if err != nil {
			writeAPIError(cmd, "resource.download", "DS_API_ERROR", err)
			return err
		}
		if err := os.WriteFile(outputFile, resp.Body, 0o600); err != nil {
			writeAPIError(cmd, "resource.download", "LOCAL_IO_ERROR", err)
			return err
		}
		e := output.NewEnvelope("resource.download")
		e.Summary = map[string]any{"cluster": profile.Name, "api_url": profile.APIURL, "http_status": resp.HTTPStatus, "output": outputFile, "bytes": len(resp.Body)}
		return e.Write(cmd.OutOrStdout())
	}}
	c.Flags().StringVar(&outputFile, "output", "", "Local output file")
	return c
}

func newResourceDeleteCmd(flags *apiFlags) *cobra.Command {
	return &cobra.Command{Use: "delete <full-name>", Short: "Delete a resource.", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		return apiRun(cmd, *flags, "resource.delete", func(ctx context.Context, client *dsapi.Client) (*dsapi.Response, error) {
			return client.Form(ctx, http.MethodDelete, "/resources", formValues("fullName", args[0]))
		})
	}}
}
