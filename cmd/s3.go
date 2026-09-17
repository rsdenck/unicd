package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

const (
	s3DefaultEndpoint = "s3.unifique.cloud"
	s3DefaultRegion   = "us-east-1"
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

// s3ResolveConfig merges saved credentials with environment overrides.
func s3ResolveConfig() (*S3Config, error) {
	env := s3EnvConfig()

	cfg, err := s3Load()
	if err != nil {
		if env.AccessKey == "" || env.SecretKey == "" {
			return nil, err
		}
		cfg = env
	} else {
		if env.AccessKey != "" && env.SecretKey != "" {
			cfg.AccessKey = env.AccessKey
			cfg.SecretKey = env.SecretKey
		}
		if os.Getenv("UNICD_S3_ENDPOINT") != "" {
			cfg.Endpoint = env.Endpoint
		}
		if os.Getenv("UNICD_S3_REGION") != "" {
			cfg.Region = env.Region
		}
	}

	if cfg.AccessKey == "" || cfg.SecretKey == "" {
		return nil, fmt.Errorf("missing S3 credentials, run 'unicd s3 auth' first")
	}
	if cfg.Region == "" {
		cfg.Region = s3DefaultRegion
	}
	return cfg, nil
}

// s3Client creates a native S3-compatible client (path-style, SigV4).
func s3Client(cfg *S3Config) (*minio.Client, error) {
	cl, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:        credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure:       cfg.UseSSL,
		Region:       cfg.Region,
		BucketLookup: minio.BucketLookupPath,
	})
	if err != nil {
		return nil, err
	}
	return cl, nil
}

func s3ClientFromConfig() (*minio.Client, error) {
	cfg, err := s3ResolveConfig()
	if err != nil {
		return nil, err
	}
	return s3Client(cfg)
}

func newS3Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "s3",
		Short: "Object storage (S3-compatible)",
		Long: `Object storage commands for the Unifique S3-compatible service
(endpoint ` + s3DefaultEndpoint + `).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(
		newS3AuthCmd(),
		newS3ConfigCmd(),
		newS3LogoutCmd(),
		newS3LsCmd(),
		newS3CpCmd(),
		newS3MvCmd(),
		newS3RmCmd(),
		newS3MbCmd(),
		newS3RbCmd(),
		newS3CatCmd(),
		newS3DuCmd(),
		newS3HeadCmd(),
		newS3PresignCmd(),
		newS3SyncCmd(),
		newS3PipeCmd(),
	)
	return cmd
}

func newS3AuthCmd() *cobra.Command {
	var accessKey, secretKey, endpoint, region string
	var insecure, check bool
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Authenticate to the object storage interactively",
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

func s3Verify(cfg *S3Config) error {
	cl, err := s3Client(cfg)
	if err != nil {
		return err
	}
	if _, err := cl.ListBuckets(context.Background()); err != nil {
		return err
	}
	return nil
}

// ---------------------------------------------------------------------------
// Path and filter helpers
// ---------------------------------------------------------------------------

type s3Path struct {
	isS3   bool
	bucket string
	key    string
}

func parseS3Path(s string) s3Path {
	if !strings.HasPrefix(s, "s3://") {
		return s3Path{}
	}
	rest := strings.TrimPrefix(s, "s3://")
	bucket := rest
	key := ""
	if i := strings.IndexByte(rest, '/'); i >= 0 {
		bucket = rest[:i]
		key = rest[i+1:]
	}
	return s3Path{isS3: true, bucket: bucket, key: key}
}

// splitWildcard splits a key into a literal prefix and a glob pattern.
func splitWildcard(key string) (prefix, pattern string) {
	i := strings.IndexAny(key, "*?[")
	if i < 0 {
		return key, ""
	}
	head := key[:i]
	if j := strings.LastIndex(head, "/"); j >= 0 {
		return head[:j+1], key[j+1:]
	}
	return "", key
}

func s3JoinKey(base, name string) string {
	if base == "" {
		return strings.TrimPrefix(name, "/")
	}
	return strings.TrimSuffix(base, "/") + "/" + strings.TrimPrefix(name, "/")
}

func s3ObjectKeyFor(dstKey, base string) string {
	if dstKey == "" || strings.HasSuffix(dstKey, "/") {
		return s3JoinKey(dstKey, base)
	}
	return dstKey
}

func localTarget(dst, base string) string {
	if dst == "" || strings.HasSuffix(dst, "/") || isExistingDir(dst) {
		return filepath.Join(dst, base)
	}
	return dst
}

func isExistingDir(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.IsDir()
}

// s3GlobMatch matches a glob against the full key and its basename.
func s3GlobMatch(pattern, key string) bool {
	if ok, err := path.Match(pattern, key); err == nil && ok {
		return true
	}
	ok, err := path.Match(pattern, path.Base(key))
	return err == nil && ok
}

// s3MatchFilters applies --include/--exclude rules to a key.
func s3MatchFilters(key string, include, exclude []string) bool {
	if len(include) > 0 {
		matched := false
		for _, p := range include {
			if s3GlobMatch(p, key) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	for _, p := range exclude {
		if s3GlobMatch(p, key) {
			return false
		}
	}
	return true
}

func s3ListAll(ctx context.Context, cl *minio.Client, bucket, prefix string, recursive, useV1 bool) ([]minio.ObjectInfo, error) {
	var out []minio.ObjectInfo
	for obj := range cl.ListObjects(ctx, bucket, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: recursive,
		UseV1:     useV1,
	}) {
		if obj.Err != nil {
			return nil, obj.Err
		}
		out = append(out, obj)
	}
	return out, nil
}

func formatSize(n int64, human bool) string {
	if !human {
		return fmt.Sprintf("%d", n)
	}
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%dB", n)
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%ciB", float64(n)/float64(div), "KMGTPE"[exp])
}

// ---------------------------------------------------------------------------
// ls / du / head / cat / presign
// ---------------------------------------------------------------------------

func newS3LsCmd() *cobra.Command {
	var humanize, sum, v2 bool
	cmd := &cobra.Command{
		Use:   "ls [s3://bucket[/prefix]]",
		Short: "List buckets or objects",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, err := s3ClientFromConfig()
			if err != nil {
				return err
			}
			ctx := context.Background()

			if len(args) == 0 {
				buckets, err := cl.ListBuckets(ctx)
				if err != nil {
					return err
				}
				for _, b := range buckets {
					name := "s3://" + b.Name
					if b.CreationDate.IsZero() {
						fmt.Println(name)
					} else {
						fmt.Printf("%s  %s\n", b.CreationDate.Format("2006/01/02 15:04:05"), name)
					}
				}
				return nil
			}

			u := parseS3Path(args[0])
			if !u.isS3 || u.bucket == "" {
				return fmt.Errorf("expected an s3://bucket[/prefix] argument")
			}
			prefix, pattern := splitWildcard(u.key)
			recursive := pattern != ""

			var count int
			var total int64
			for obj := range cl.ListObjects(ctx, u.bucket, minio.ListObjectsOptions{
				Prefix:    prefix,
				Recursive: recursive,
				UseV1:     !v2,
			}) {
				if obj.Err != nil {
					return obj.Err
				}
				if pattern != "" {
					if ok, _ := path.Match(pattern, path.Base(obj.Key)); !ok {
						continue
					}
				}
				if !recursive && strings.HasSuffix(obj.Key, "/") {
					fmt.Printf("%19s  %s\n", "DIR", obj.Key)
					continue
				}
				count++
				total += obj.Size
				fmt.Printf("%s %19s  %s\n",
					obj.LastModified.Format("2006/01/02 15:04:05"),
					formatSize(obj.Size, humanize), obj.Key)
			}
			if sum {
				fmt.Printf("total: %s (%d objects)\n", formatSize(total, humanize), count)
			}
			return nil
		},
	}
	cmd.Flags().BoolVarP(&humanize, "humanize", "H", false, "Human readable sizes")
	cmd.Flags().BoolVarP(&sum, "sum", "s", false, "Print total size and object count")
	cmd.Flags().BoolVar(&v2, "list-objects-v2", false, "Use ListObjectsV2 instead of V1")
	return cmd
}

func newS3DuCmd() *cobra.Command {
	var humanize, v2 bool
	cmd := &cobra.Command{
		Use:   "du s3://bucket[/prefix]",
		Short: "Show total size of objects under a prefix",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, err := s3ClientFromConfig()
			if err != nil {
				return err
			}
			u := parseS3Path(args[0])
			if !u.isS3 || u.bucket == "" {
				return fmt.Errorf("expected an s3://bucket[/prefix] argument")
			}
			prefix, _ := splitWildcard(u.key)
			objs, err := s3ListAll(context.Background(), cl, u.bucket, prefix, true, !v2)
			if err != nil {
				return err
			}
			var total int64
			for _, o := range objs {
				total += o.Size
			}
			fmt.Printf("total: %s (%d objects)\n", formatSize(total, humanize), len(objs))
			return nil
		},
	}
	cmd.Flags().BoolVarP(&humanize, "humanize", "H", false, "Human readable sizes")
	cmd.Flags().BoolVar(&v2, "list-objects-v2", false, "Use ListObjectsV2 instead of V1")
	return cmd
}

func newS3HeadCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "head s3://bucket/key",
		Short: "Show object metadata",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, err := s3ClientFromConfig()
			if err != nil {
				return err
			}
			u := parseS3Path(args[0])
			if !u.isS3 || u.bucket == "" || u.key == "" {
				return fmt.Errorf("expected an s3://bucket/key argument")
			}
			info, err := cl.StatObject(context.Background(), u.bucket, u.key, minio.StatObjectOptions{})
			if err != nil {
				return err
			}
			meta := map[string]string{}
			for k, v := range info.Metadata {
				if len(v) > 0 {
					meta[strings.ToLower(k)] = v[0]
				}
			}
			out := struct {
				Key          string            `json:"key"`
				Size         int64             `json:"size"`
				ETag         string            `json:"etag"`
				ContentType  string            `json:"content_type"`
				LastModified time.Time         `json:"last_modified"`
				StorageClass string            `json:"storage_class"`
				Metadata     map[string]string `json:"metadata"`
			}{
				Key:          u.key,
				Size:         info.Size,
				ETag:         strings.Trim(info.ETag, "\""),
				ContentType:  info.ContentType,
				LastModified: info.LastModified,
				StorageClass: info.StorageClass,
				Metadata:     meta,
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(out)
		},
	}
}

func newS3CatCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "cat s3://bucket/key",
		Short: "Print an object to stdout",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, err := s3ClientFromConfig()
			if err != nil {
				return err
			}
			u := parseS3Path(args[0])
			if !u.isS3 || u.bucket == "" || u.key == "" {
				return fmt.Errorf("expected an s3://bucket/key argument")
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			obj, err := cl.GetObject(ctx, u.bucket, u.key, minio.GetObjectOptions{})
			if err != nil {
				return err
			}
			defer obj.Close()
			if _, err := io.Copy(os.Stdout, obj); err != nil {
				return err
			}
			return nil
		},
	}
}

func newS3PresignCmd() *cobra.Command {
	var expire time.Duration
	var method string
	cmd := &cobra.Command{
		Use:   "presign s3://bucket/key",
		Short: "Generate a presigned URL",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, err := s3ClientFromConfig()
			if err != nil {
				return err
			}
			u := parseS3Path(args[0])
			if !u.isS3 || u.bucket == "" || u.key == "" {
				return fmt.Errorf("expected an s3://bucket/key argument")
			}
			var uri *url.URL
			switch strings.ToUpper(method) {
			case "GET":
				uri, err = cl.PresignedGetObject(context.Background(), u.bucket, u.key, expire, url.Values{})
			case "PUT":
				uri, err = cl.PresignedPutObject(context.Background(), u.bucket, u.key, expire)
			default:
				return fmt.Errorf("unsupported --method %q (use GET or PUT)", method)
			}
			if err != nil {
				return err
			}
			fmt.Println(uri.String())
			return nil
		},
	}
	cmd.Flags().DurationVarP(&expire, "expire", "e", time.Hour, "URL validity duration")
	cmd.Flags().StringVarP(&method, "method", "X", "GET", "HTTP method (GET or PUT)")
	return cmd
}

// ---------------------------------------------------------------------------
// mb / rb / rm
// ---------------------------------------------------------------------------

func newS3MbCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "mb s3://bucket",
		Short: "Create a bucket",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := s3ResolveConfig()
			if err != nil {
				return err
			}
			cl, err := s3Client(cfg)
			if err != nil {
				return err
			}
			u := parseS3Path(args[0])
			if !u.isS3 || u.bucket == "" {
				return fmt.Errorf("expected an s3://bucket argument")
			}
			if err := cl.MakeBucket(context.Background(), u.bucket, minio.MakeBucketOptions{Region: cfg.Region}); err != nil {
				return err
			}
			fmt.Printf("Bucket created: s3://%s\n", u.bucket)
			return nil
		},
	}
}

func newS3RbCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rb s3://bucket",
		Short: "Remove an empty bucket",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, err := s3ClientFromConfig()
			if err != nil {
				return err
			}
			u := parseS3Path(args[0])
			if !u.isS3 || u.bucket == "" {
				return fmt.Errorf("expected an s3://bucket argument")
			}
			if err := cl.RemoveBucket(context.Background(), u.bucket); err != nil {
				return err
			}
			fmt.Printf("Bucket removed: s3://%s\n", u.bucket)
			return nil
		},
	}
}

func newS3RmCmd() *cobra.Command {
	var recursive, v2, dryRun bool
	var include, exclude []string
	cmd := &cobra.Command{
		Use:   "rm s3://bucket/key [s3://bucket/key ...]",
		Short: "Remove objects",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, err := s3ClientFromConfig()
			if err != nil {
				return err
			}
			ctx := context.Background()
			for _, a := range args {
				u := parseS3Path(a)
				if !u.isS3 || u.bucket == "" || u.key == "" {
					return fmt.Errorf("expected an s3://bucket/key argument: %s", a)
				}
				prefix, pattern := splitWildcard(u.key)
				if recursive || pattern != "" {
					objs, err := s3ListAll(ctx, cl, u.bucket, prefix, true, !v2)
					if err != nil {
						return err
					}
					for _, o := range objs {
						if strings.HasSuffix(o.Key, "/") {
							continue
						}
						if pattern != "" {
							if ok, _ := path.Match(pattern, path.Base(o.Key)); !ok {
								continue
							}
						}
						if !s3MatchFilters(o.Key, include, exclude) {
							continue
						}
						if dryRun {
							fmt.Printf("rm s3://%s/%s (dry-run)\n", u.bucket, o.Key)
							continue
						}
						if err := cl.RemoveObject(ctx, u.bucket, o.Key, minio.RemoveObjectOptions{}); err != nil {
							return err
						}
						fmt.Printf("rm s3://%s/%s\n", u.bucket, o.Key)
					}
					continue
				}
				if dryRun {
					fmt.Printf("rm s3://%s/%s (dry-run)\n", u.bucket, u.key)
					continue
				}
				if err := cl.RemoveObject(ctx, u.bucket, u.key, minio.RemoveObjectOptions{}); err != nil {
					return err
				}
				fmt.Printf("rm s3://%s/%s\n", u.bucket, u.key)
			}
			return nil
		},
	}
	cmd.Flags().BoolVarP(&recursive, "recursive", "r", false, "Remove all objects under the prefix")
	cmd.Flags().StringArrayVar(&include, "include", nil, "Only remove keys matching the glob (repeatable)")
	cmd.Flags().StringArrayVar(&exclude, "exclude", nil, "Skip keys matching the glob (repeatable)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be removed")
	cmd.Flags().BoolVar(&v2, "list-objects-v2", false, "Use ListObjectsV2 instead of V1")
	return cmd
}

func init() {
	rootCmd.AddCommand(newS3Cmd())
}
