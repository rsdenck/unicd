package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

const (
	s3DefaultEndpoint = "s3.unifique.cloud"
	s3DefaultRegion   = "us-east-1"
	s5cmdVersion      = "2.3.0"
)

type S3Config struct {
	Endpoint  string `json:"endpoint"`
	AccessKey string `json:"accessKey"`
	SecretKey string `json:"secretKey"`
	Region    string `json:"region"`
	UseSSL    bool   `json:"useSSL"`
}

func s3ConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".unicd", "s3.json")
}

func s3Load() (*S3Config, error) {
	data, err := os.ReadFile(s3ConfigPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("S3 not configured, run 'unicd s3 auth' first")
		}
		return nil, err
	}
	var cfg S3Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func s3Save(cfg *S3Config) error {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".unicd")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s3ConfigPath(), data, 0600)
}

// s3EnvConfig builds an S3Config from environment variables (highest priority).
func s3EnvConfig() *S3Config {
	endpoint := os.Getenv("UNICD_S3_ENDPOINT")
	if endpoint == "" {
		endpoint = s3DefaultEndpoint
	}
	return &S3Config{
		Endpoint:  endpoint,
		AccessKey: os.Getenv("UNICD_S3_ACCESS_KEY"),
		SecretKey: os.Getenv("UNICD_S3_SECRET"),
		Region:    envDefault("UNICD_S3_REGION", s3DefaultRegion),
		UseSSL:    true,
	}
}

func envDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func findS5cmd() (string, error) {
	p, err := exec.LookPath("s5cmd")
	if err != nil {
		return "", fmt.Errorf("s5cmd not found in PATH. Install it with 'unicd s3 install' or manually (https://github.com/peak/s5cmd)")
	}
	return p, nil
}

func newS3Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "s3",
		Short: "S3 operations (Dell ECS / s5cmd)",
		Long:  `S3 commands for the Unifique object storage (Dell ECS, endpoint ` + s3DefaultEndpoint + `). Uses the s5cmd CLI.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			return runS5(cmd, args)
		},
	}
	cmd.AddCommand(newS3AuthCmd())
	cmd.AddCommand(newS3LogoutCmd())
	cmd.AddCommand(newS3ConfigCmd())
	cmd.AddCommand(newS3InstallCmd())
	return cmd
}

func newS3AuthCmd() *cobra.Command {
	var accessKey, secretKey, endpoint, region string
	var insecure, check bool
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Authenticate to Dell ECS interactively",
		RunE: func(cmd *cobra.Command, args []string) error {
			if accessKey == "" {
				accessKey = os.Getenv("UNICD_S3_ACCESS_KEY")
			}
			if accessKey == "" {
				fmt.Print("Access Key ID: ")
				fmt.Scanln(&accessKey)
			}
			if secretKey == "" {
				secretKey = os.Getenv("UNICD_S3_SECRET")
			}
			if secretKey == "" {
				fmt.Print("Secret Access Key: ")
				raw, err := term.ReadPassword(int(syscall.Stdin))
				if err != nil {
					return fmt.Errorf("reading secret key: %w", err)
				}
				secretKey = string(raw)
				fmt.Println()
			}
			if endpoint == "" {
				endpoint = os.Getenv("UNICD_S3_ENDPOINT")
			}
			if endpoint == "" {
				endpoint = s3DefaultEndpoint
			}
			if region == "" {
				region = os.Getenv("UNICD_S3_REGION")
			}
			if region == "" {
				region = s3DefaultRegion
			}

			cfg := &S3Config{
				Endpoint:  strings.TrimPrefix(strings.TrimSuffix(endpoint, "/"), "https://"),
				AccessKey: accessKey,
				SecretKey: secretKey,
				Region:    region,
				UseSSL:    !insecure,
			}
			if err := s3Save(cfg); err != nil {
				return err
			}

			if check {
				if err := s3Verify(cfg); err != nil {
					return fmt.Errorf("credentials saved but verification failed: %w", err)
				}
			}

			fmt.Printf("S3 credentials saved for %s (user %s)\n", cfg.Endpoint, accessKey)
			return nil
		},
	}
	cmd.Flags().StringVarP(&accessKey, "access-key", "u", "", "Access Key ID")
	cmd.Flags().StringVarP(&secretKey, "secret-key", "p", "", "Secret Access Key")
	cmd.Flags().StringVar(&endpoint, "endpoint", "", "S3 endpoint (default: "+s3DefaultEndpoint+")")
	cmd.Flags().StringVar(&region, "region", "", "S3 region (default: "+s3DefaultRegion+")")
	cmd.Flags().BoolVar(&insecure, "insecure", false, "Use http instead of https")
	cmd.Flags().BoolVarP(&check, "check", "c", true, "Verify connectivity after saving")
	return cmd
}

func newS3LogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Remove saved S3 credentials",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := os.Remove(s3ConfigPath()); err != nil && !os.IsNotExist(err) {
				return err
			}
			fmt.Println("S3 credentials removed")
			return nil
		},
	}
}

func newS3ConfigCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "config",
		Short: "Show current S3 configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := s3Load()
			if err != nil {
				return err
			}
			fmt.Printf("endpoint:   %s\n", cfg.Endpoint)
			fmt.Printf("region:     %s\n", cfg.Region)
			fmt.Printf("access key: %s\n", cfg.AccessKey)
			fmt.Printf("secret key: %s\n", truncate(cfg.SecretKey, 8))
			fmt.Printf("use ssl:    %v\n", cfg.UseSSL)
			return nil
		},
	}
}

func newS3InstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "install",
		Short: "Install/upgrade the s5cmd binary on any Linux distro",
		RunE: func(cmd *cobra.Command, args []string) error {
			return installS5cmd()
		},
	}
}

func installS5cmd() error {
	tmpBin := filepath.Join(os.TempDir(), "s5cmd")
	tgz := tmpBin + ".tar.gz"
	url := fmt.Sprintf("https://github.com/peak/s5cmd/releases/download/v%s/s5cmd_%s_Linux-64bit.tar.gz", s5cmdVersion, s5cmdVersion)

	curl := exec.Command("curl", "-fL", url, "-o", tgz)
	curl.Stdout = os.Stdout
	curl.Stderr = os.Stderr
	if err := curl.Run(); err != nil {
		return fmt.Errorf("downloading s5cmd: %w", err)
	}
	defer os.Remove(tgz)

	tar := exec.Command("tar", "-xzf", tgz, "-C", os.TempDir(), "s5cmd")
	tar.Stdout = os.Stdout
	tar.Stderr = os.Stderr
	if err := tar.Run(); err != nil {
		return fmt.Errorf("extracting s5cmd: %w", err)
	}
	defer os.Remove(tmpBin)

	install := exec.Command("install", "-m", "0755", tmpBin, "/usr/local/bin/s5cmd")
	install.Stdout = os.Stdout
	install.Stderr = os.Stderr
	if err := install.Run(); err != nil {
		return fmt.Errorf("installing s5cmd: %w (consider running with sudo)", err)
	}

	ver := exec.Command("s5cmd", "version")
	ver.Stdout = os.Stdout
	ver.Stderr = os.Stderr
	ver.Run()
	return nil
}

func s3Verify(cfg *S3Config) error {
	binary, err := findS5cmd()
	if err != nil {
		return err
	}
	if err := runS5Args(binary, cfg, []string{"ls"}); err != nil {
		return err
	}
	return nil
}

// runS5 forwards arbitrary s5cmd subcommands using the saved credentials.
func runS5(cmd *cobra.Command, args []string) error {
	envCfg := s3EnvConfig()

	var cfg *S3Config
	cfg, err := s3Load()
	if err != nil {
		if envCfg.AccessKey == "" || envCfg.SecretKey == "" {
			return err
		}
		cfg = envCfg
	}

	if envCfg.AccessKey != "" && envCfg.SecretKey != "" {
		cfg.AccessKey = envCfg.AccessKey
		cfg.SecretKey = envCfg.SecretKey
		if os.Getenv("UNICD_S3_ENDPOINT") != "" {
			cfg.Endpoint = envCfg.Endpoint
		}
	}

	binary, err := findS5cmd()
	if err != nil {
		return err
	}
	return runS5Args(binary, cfg, args)
}

func s3CredsFile(cfg *S3Config) (string, error) {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".unicd")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	path := filepath.Join(dir, "s3.credentials")
	region := cfg.Region
	if region == "" {
		region = s3DefaultRegion
	}
	content := fmt.Sprintf(
		"[default]\naws_access_key_id=%s\naws_secret_access_key=%s\nregion=%s\n",
		cfg.AccessKey, cfg.SecretKey, region,
	)
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		return "", err
	}
	return path, nil
}

func runS5Args(binary string, cfg *S3Config, args []string) error {
	credsFile, err := s3CredsFile(cfg)
	if err != nil {
		return err
	}

	full := []string{
		"--credentials-file", credsFile,
		"--profile", "default",
		"--endpoint-url", s3EndpointURL(cfg),
	}
	full = append(full, args...)

	s5 := exec.Command(binary, full...)
	s5.Stdin = os.Stdin
	s5.Stdout = os.Stdout
	s5.Stderr = os.Stderr
	return s5.Run()
}

func s3EndpointURL(cfg *S3Config) string {
	scheme := "https"
	if !cfg.UseSSL {
		scheme = "http"
	}
	return fmt.Sprintf("%s://%s", scheme, cfg.Endpoint)
}

func init() {
	rootCmd.AddCommand(newS3Cmd())
}
