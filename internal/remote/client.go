package remote

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/sftp"
	xssh "golang.org/x/crypto/ssh"
)

type Config struct {
	Host       string
	Port       int
	User       string
	PrivateKey string
	Timeout    time.Duration
}

type Client struct {
	conn *xssh.Client
}

type ExecResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

func Dial(cfg Config) (*Client, error) {
	if cfg.Timeout == 0 {
		cfg.Timeout = 10 * time.Second
	}
	if cfg.Port == 0 {
		cfg.Port = 22
	}
	path := expand(cfg.PrivateKey)
	key, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read private key %s: %w", path, err)
	}
	signer, err := xssh.ParsePrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("parse private key %s: %w", path, err)
	}
	clientCfg := &xssh.ClientConfig{
		User:            cfg.User,
		Auth:            []xssh.AuthMethod{xssh.PublicKeys(signer)},
		HostKeyCallback: xssh.InsecureIgnoreHostKey(),
		Timeout:         cfg.Timeout,
	}
	conn, err := xssh.Dial("tcp", net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port)), clientCfg)
	if err != nil {
		return nil, err
	}
	return &Client{conn: conn}, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) Exec(ctx context.Context, script string) (*ExecResult, error) {
	sess, err := c.conn.NewSession()
	if err != nil {
		return nil, err
	}
	defer sess.Close()
	var stdout, stderr bytes.Buffer
	sess.Stdout = &stdout
	sess.Stderr = &stderr
	done := make(chan error, 1)
	go func() { done <- sess.Run(script) }()
	select {
	case <-ctx.Done():
		_ = sess.Signal(xssh.SIGKILL)
		return nil, ctx.Err()
	case err := <-done:
		exit := 0
		if err != nil {
			var xerr *xssh.ExitError
			if errors.As(err, &xerr) {
				exit = xerr.ExitStatus()
			} else {
				return &ExecResult{Stdout: stdout.String(), Stderr: stderr.String()}, err
			}
		}
		return &ExecResult{Stdout: stdout.String(), Stderr: stderr.String(), ExitCode: exit}, nil
	}
}

func (c *Client) WriteFile(remote string, content []byte, mode os.FileMode) error {
	sc, err := sftp.NewClient(c.conn)
	if err != nil {
		return err
	}
	defer sc.Close()
	if err := sc.MkdirAll(filepath.Dir(remote)); err != nil {
		return err
	}
	f, err := sc.Create(remote)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Write(content); err != nil {
		return err
	}
	return sc.Chmod(remote, mode)
}

func expand(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		if home != "" {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}
