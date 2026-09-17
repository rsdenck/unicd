package cmd

import (
	"context"
	"fmt"
	"io"
	"mime"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/minio/minio-go/v7"
	"github.com/spf13/cobra"
)

// s3TransferOptions holds the shared flags for upload/download/copy/sync.
type s3TransferOptions struct {
	Recursive       bool
	Concurrency     int
	PartSizeRaw     string
	PartSize        uint64
	StorageClass    string
	ContentType     string
	ContentEncoding string
	CacheControl    string
	MetadataRaw     []string
	Metadata        map[string]string
	ExpiresRaw      string
	Expires         time.Time
	NoClobber       bool
	IfSizeDiffer    bool
	Include         []string
	Exclude         []string
	DryRun          bool
	UseV1           bool

	// sync
	Delete          bool
	ExactTimestamps bool

	// mv
	Move bool
}

func (o *s3TransferOptions) prepare() error {
	if o.Concurrency < 1 {
		o.Concurrency = 1
	}
	if o.PartSizeRaw != "" {
		b, err := humanize.ParseBytes(o.PartSizeRaw)
		if err != nil {
			return fmt.Errorf("invalid --part-size %q: %w", o.PartSizeRaw, err)
		}
		o.PartSize = b
	}
	if o.ExpiresRaw != "" {
		t, err := time.Parse(time.RFC3339, o.ExpiresRaw)
		if err != nil {
			return fmt.Errorf("invalid --expires (use RFC3339): %w", err)
		}
		o.Expires = t
	}
	if len(o.MetadataRaw) > 0 {
		o.Metadata = map[string]string{}
		for _, kv := range o.MetadataRaw {
			i := strings.IndexByte(kv, '=')
			if i < 1 {
				return fmt.Errorf("invalid --metadata %q (expected key=value)", kv)
			}
			o.Metadata[kv[:i]] = kv[i+1:]
		}
	}
	return nil
}

func (o *s3TransferOptions) putOptions(key string) minio.PutObjectOptions {
	opts := minio.PutObjectOptions{
		StorageClass:    o.StorageClass,
		ContentType:     o.ContentType,
		ContentEncoding: o.ContentEncoding,
		CacheControl:    o.CacheControl,
	}
	if opts.ContentType == "" {
		if ct := mime.TypeByExtension(path.Ext(key)); ct != "" {
			opts.ContentType = ct
		}
	}
	if o.PartSize > 0 {
		opts.PartSize = o.PartSize
	}
	if o.Concurrency > 1 {
		opts.NumThreads = uint(o.Concurrency)
	}
	if len(o.Metadata) > 0 {
		opts.UserMetadata = o.Metadata
	}
	if !o.Expires.IsZero() {
		opts.Expires = o.Expires
	}
	return opts
}

func s3TransferFlags(cmd *cobra.Command, o *s3TransferOptions) {
	cmd.Flags().IntVarP(&o.Concurrency, "concurrency", "c", 5, "Number of parallel transfers")
	cmd.Flags().StringVar(&o.PartSizeRaw, "part-size", "", "Multipart part size (e.g. 8MB)")
	cmd.Flags().StringVar(&o.StorageClass, "storage-class", "", "Storage class for uploaded objects")
	cmd.Flags().StringVar(&o.ContentType, "content-type", "", "Content-Type override")
	cmd.Flags().StringVar(&o.ContentEncoding, "content-encoding", "", "Content-Encoding header")
	cmd.Flags().StringVar(&o.CacheControl, "cache-control", "", "Cache-Control header")
	cmd.Flags().StringArrayVar(&o.MetadataRaw, "metadata", nil, "User metadata key=value (repeatable)")
	cmd.Flags().StringVar(&o.ExpiresRaw, "expires", "", "Expiration date (RFC3339)")
	cmd.Flags().BoolVar(&o.NoClobber, "no-clobber", false, "Do not overwrite existing objects/files")
	cmd.Flags().BoolVar(&o.IfSizeDiffer, "if-size-differ", false, "Transfer only when sizes differ")
	cmd.Flags().StringArrayVar(&o.Include, "include", nil, "Only transfer keys matching the glob (repeatable)")
	cmd.Flags().StringArrayVar(&o.Exclude, "exclude", nil, "Skip keys matching the glob (repeatable)")
	cmd.Flags().BoolVar(&o.DryRun, "dry-run", false, "Show what would be done without transferring")
	cmd.Flags().BoolVar(&o.UseV1, "use-list-objects-v1", true, "Use ListObjectsV1 (required by some S3 services)")
}

// s3Parallel runs jobs with a bounded worker pool, returning the first error.
func s3Parallel(concurrency int, jobs []func() error) error {
	if concurrency < 1 {
		concurrency = 1
	}
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error

	for _, job := range jobs {
		mu.Lock()
		failed := firstErr != nil
		mu.Unlock()
		if failed {
			break
		}
		wg.Add(1)
		sem <- struct{}{}
		j := job
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			if err := j(); err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	return firstErr
}

// ---------------------------------------------------------------------------
// cp / mv
// ---------------------------------------------------------------------------

func newS3CpCmd() *cobra.Command { return s3CopyMoveCmd(false) }
func newS3MvCmd() *cobra.Command { return s3CopyMoveCmd(true) }

func s3CopyMoveCmd(move bool) *cobra.Command {
	o := &s3TransferOptions{Move: move}
	use, short := "cp", "Copy files/objects"
	if move {
		use, short = "mv", "Move files/objects"
	}
	cmd := &cobra.Command{
		Use:   use + " <source> <destination>",
		Short: short,
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.prepare(); err != nil {
				return err
			}
			cl, err := s3ClientFromConfig()
			if err != nil {
				return err
			}
			return s3Transfer(context.Background(), cl, args[0], args[1], o)
		},
	}
	cmd.Flags().BoolVarP(&o.Recursive, "recursive", "r", false, "Recurse into directories/prefixes")
	s3TransferFlags(cmd, o)
	return cmd
}

func s3Transfer(ctx context.Context, cl *minio.Client, src, dst string, o *s3TransferOptions) error {
	su := parseS3Path(src)
	du := parseS3Path(dst)

	switch {
	case !su.isS3 && du.isS3:
		return s3Upload(ctx, cl, src, du, o)
	case su.isS3 && !du.isS3:
		return s3Download(ctx, cl, su, dst, o)
	case su.isS3 && du.isS3:
		return s3CopyRemote(ctx, cl, su, du, o)
	default:
		return fmt.Errorf("at least one argument must be an s3:// path")
	}
}

func s3Upload(ctx context.Context, cl *minio.Client, local string, dst s3Path, o *s3TransferOptions) error {
	if dst.bucket == "" {
		return fmt.Errorf("invalid destination: bucket required")
	}

	if strings.ContainsAny(local, "*?[") {
		matches, err := filepath.Glob(local)
		if err != nil {
			return err
		}
		var jobs []func() error
		for _, m := range matches {
			fi, err := os.Stat(m)
			if err != nil {
				return err
			}
			if fi.IsDir() {
				continue
			}
			key := s3ObjectKeyFor(dst.key, filepath.Base(m))
			if !s3MatchFilters(key, o.Include, o.Exclude) {
				continue
			}
			jobs = append(jobs, func() error { return s3UploadFile(ctx, cl, m, dst.bucket, key, o) })
		}
		if len(jobs) == 0 {
			return fmt.Errorf("no files matched: %s", local)
		}
		return s3Parallel(o.Concurrency, jobs)
	}

	fi, err := os.Stat(local)
	if err != nil {
		return err
	}

	if fi.IsDir() {
		if !o.Recursive {
			return fmt.Errorf("%s is a directory (use -r)", local)
		}
		base := filepath.Clean(local)
		var jobs []func() error
		err := filepath.Walk(base, func(p string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			rel, err := filepath.Rel(base, p)
			if err != nil {
				return err
			}
			key := s3JoinKey(dst.key, filepath.ToSlash(rel))
			if !s3MatchFilters(key, o.Include, o.Exclude) {
				return nil
			}
			jobs = append(jobs, func() error { return s3UploadFile(ctx, cl, p, dst.bucket, key, o) })
			return nil
		})
		if err != nil {
			return err
		}
		if err := s3Parallel(o.Concurrency, jobs); err != nil {
			return err
		}
		if o.Move && !o.DryRun {
			os.RemoveAll(base)
		}
		return nil
	}

	key := s3ObjectKeyFor(dst.key, filepath.Base(local))
	if !s3MatchFilters(key, o.Include, o.Exclude) {
		return nil
	}
	if err := s3UploadFile(ctx, cl, local, dst.bucket, key, o); err != nil {
		return err
	}
	if o.Move && !o.DryRun {
		os.Remove(local)
	}
	return nil
}

func s3UploadFile(ctx context.Context, cl *minio.Client, localPath, bucket, key string, o *s3TransferOptions) error {
	if o.DryRun {
		fmt.Printf("upload %s -> s3://%s/%s (dry-run)\n", localPath, bucket, key)
		return nil
	}
	fi, err := os.Stat(localPath)
	if err != nil {
		return err
	}
	if o.NoClobber {
		if _, err := cl.StatObject(ctx, bucket, key, minio.StatObjectOptions{}); err == nil {
			fmt.Printf("skip s3://%s/%s (exists)\n", bucket, key)
			return nil
		}
	}
	if o.IfSizeDiffer {
		if info, err := cl.StatObject(ctx, bucket, key, minio.StatObjectOptions{}); err == nil && info.Size == fi.Size() {
			return nil
		}
	}
	if _, err := cl.FPutObject(ctx, bucket, key, localPath, o.putOptions(key)); err != nil {
		return err
	}
	fmt.Printf("upload %s -> s3://%s/%s\n", localPath, bucket, key)
	return nil
}

func s3Download(ctx context.Context, cl *minio.Client, src s3Path, dst string, o *s3TransferOptions) error {
	if src.bucket == "" {
		return fmt.Errorf("invalid source: bucket required")
	}
	prefix, pattern := splitWildcard(src.key)
	hasPattern := pattern != ""

	if !hasPattern && !o.Recursive {
		if _, err := cl.StatObject(ctx, src.bucket, src.key, minio.StatObjectOptions{}); err == nil {
			return s3DownloadFile(ctx, cl, src.bucket, src.key, localTarget(dst, path.Base(src.key)), o)
		}
	}

	objs, err := s3ListAll(ctx, cl, src.bucket, prefix, true, o.UseV1)
	if err != nil {
		return err
	}
	var jobs []func() error
	for _, ob := range objs {
		if strings.HasSuffix(ob.Key, "/") {
			continue
		}
		if hasPattern {
			if ok, _ := path.Match(pattern, path.Base(ob.Key)); !ok {
				continue
			}
		}
		if !s3MatchFilters(ob.Key, o.Include, o.Exclude) {
			continue
		}
		rel := strings.TrimPrefix(ob.Key, prefix)
		target := filepath.Join(dst, filepath.FromSlash(rel))
		ob := ob
		target2 := target
		jobs = append(jobs, func() error {
			return s3DownloadFile(ctx, cl, src.bucket, ob.Key, target2, o)
		})
	}
	if len(jobs) == 0 {
		return fmt.Errorf("no objects found: s3://%s/%s", src.bucket, src.key)
	}
	return s3Parallel(o.Concurrency, jobs)
}

func s3DownloadFile(ctx context.Context, cl *minio.Client, bucket, key, localPath string, o *s3TransferOptions) error {
	if o.DryRun {
		fmt.Printf("download s3://%s/%s -> %s (dry-run)\n", bucket, key, localPath)
		return nil
	}
	if o.NoClobber {
		if _, err := os.Stat(localPath); err == nil {
			fmt.Printf("skip %s (exists)\n", localPath)
			return nil
		}
	}
	if o.IfSizeDiffer {
		if info, err := cl.StatObject(ctx, bucket, key, minio.StatObjectOptions{}); err == nil {
			if fi, err := os.Stat(localPath); err == nil && fi.Size() == info.Size {
				return nil
			}
		}
	}
	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return err
	}
	if err := cl.FGetObject(ctx, bucket, key, localPath, minio.GetObjectOptions{}); err != nil {
		return err
	}
	if info, err := cl.StatObject(ctx, bucket, key, minio.StatObjectOptions{}); err == nil && !info.LastModified.IsZero() {
		os.Chtimes(localPath, info.LastModified, info.LastModified)
	}
	fmt.Printf("download s3://%s/%s -> %s\n", bucket, key, localPath)
	return nil
}

func s3CopyRemote(ctx context.Context, cl *minio.Client, src, dst s3Path, o *s3TransferOptions) error {
	if src.bucket == "" || dst.bucket == "" {
		return fmt.Errorf("source and destination buckets are required")
	}
	prefix, pattern := splitWildcard(src.key)
	hasPattern := pattern != ""

	if !hasPattern && !o.Recursive {
		if _, err := cl.StatObject(ctx, src.bucket, src.key, minio.StatObjectOptions{}); err == nil {
			key := s3ObjectKeyFor(dst.key, path.Base(src.key))
			if err := s3CopyObject(ctx, cl, src.bucket, src.key, dst.bucket, key, o); err != nil {
				return err
			}
			if o.Move && !o.DryRun {
				return cl.RemoveObject(ctx, src.bucket, src.key, minio.RemoveObjectOptions{})
			}
			return nil
		}
	}

	objs, err := s3ListAll(ctx, cl, src.bucket, prefix, true, o.UseV1)
	if err != nil {
		return err
	}
	var jobs []func() error
	for _, ob := range objs {
		if strings.HasSuffix(ob.Key, "/") {
			continue
		}
		if hasPattern {
			if ok, _ := path.Match(pattern, path.Base(ob.Key)); !ok {
				continue
			}
		}
		if !s3MatchFilters(ob.Key, o.Include, o.Exclude) {
			continue
		}
		rel := strings.TrimPrefix(ob.Key, prefix)
		key := s3JoinKey(dst.key, rel)
		jobs = append(jobs, func() error {
			if err := s3CopyObject(ctx, cl, src.bucket, ob.Key, dst.bucket, key, o); err != nil {
				return err
			}
			if o.Move && !o.DryRun {
				return cl.RemoveObject(ctx, src.bucket, ob.Key, minio.RemoveObjectOptions{})
			}
			return nil
		})
	}
	if len(jobs) == 0 {
		return fmt.Errorf("no objects found: s3://%s/%s", src.bucket, src.key)
	}
	return s3Parallel(o.Concurrency, jobs)
}

func s3CopyObject(ctx context.Context, cl *minio.Client, srcBucket, srcKey, dstBucket, dstKey string, o *s3TransferOptions) error {
	if o.DryRun {
		fmt.Printf("copy s3://%s/%s -> s3://%s/%s (dry-run)\n", srcBucket, srcKey, dstBucket, dstKey)
		return nil
	}
	if o.NoClobber {
		if _, err := cl.StatObject(ctx, dstBucket, dstKey, minio.StatObjectOptions{}); err == nil {
			fmt.Printf("skip s3://%s/%s (exists)\n", dstBucket, dstKey)
			return nil
		}
	}
	if o.IfSizeDiffer {
		srcInfo, err1 := cl.StatObject(ctx, srcBucket, srcKey, minio.StatObjectOptions{})
		dstInfo, err2 := cl.StatObject(ctx, dstBucket, dstKey, minio.StatObjectOptions{})
		if err1 == nil && err2 == nil && srcInfo.Size == dstInfo.Size {
			return nil
		}
	}
	_, err := cl.CopyObject(ctx,
		minio.CopyDestOptions{Bucket: dstBucket, Object: dstKey},
		minio.CopySrcOptions{Bucket: srcBucket, Object: srcKey},
	)
	if err != nil {
		return err
	}
	fmt.Printf("copy s3://%s/%s -> s3://%s/%s\n", srcBucket, srcKey, dstBucket, dstKey)
	return nil
}

// ---------------------------------------------------------------------------
// pipe
// ---------------------------------------------------------------------------

func newS3PipeCmd() *cobra.Command {
	o := &s3TransferOptions{}
	cmd := &cobra.Command{
		Use:   "pipe s3://bucket/key",
		Short: "Upload stdin to an object",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.prepare(); err != nil {
				return err
			}
			cl, err := s3ClientFromConfig()
			if err != nil {
				return err
			}
			u := parseS3Path(args[0])
			if !u.isS3 || u.bucket == "" || u.key == "" {
				return fmt.Errorf("expected an s3://bucket/key argument")
			}
			tmp, err := os.CreateTemp("", "unicd-pipe-*")
			if err != nil {
				return err
			}
			defer os.Remove(tmp.Name())
			if _, err := io.Copy(tmp, os.Stdin); err != nil {
				tmp.Close()
				return err
			}
			if err := tmp.Close(); err != nil {
				return err
			}
			o.Move = false
			return s3UploadFile(context.Background(), cl, tmp.Name(), u.bucket, u.key, o)
		},
	}
	s3TransferFlags(cmd, o)
	return cmd
}

// ---------------------------------------------------------------------------
// sync
// ---------------------------------------------------------------------------

func newS3SyncCmd() *cobra.Command {
	o := &s3TransferOptions{}
	cmd := &cobra.Command{
		Use:   "sync <source> <destination>",
		Short: "Synchronize a directory with a bucket/prefix (one-way)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.prepare(); err != nil {
				return err
			}
			cl, err := s3ClientFromConfig()
			if err != nil {
				return err
			}
			return s3Sync(context.Background(), cl, args[0], args[1], o)
		},
	}
	cmd.Flags().BoolVar(&o.Delete, "delete", false, "Delete destination files missing at the source")
	cmd.Flags().BoolVar(&o.ExactTimestamps, "exact-timestamps", false, "Also compare modification times")
	s3TransferFlags(cmd, o)
	return cmd
}

func s3Sync(ctx context.Context, cl *minio.Client, src, dst string, o *s3TransferOptions) error {
	su := parseS3Path(src)
	du := parseS3Path(dst)

	switch {
	case !su.isS3 && du.isS3:
		return s3SyncUpload(ctx, cl, src, du, o)
	case su.isS3 && !du.isS3:
		return s3SyncDownload(ctx, cl, su, dst, o)
	case su.isS3 && du.isS3:
		return s3SyncRemote(ctx, cl, su, du, o)
	default:
		return fmt.Errorf("at least one argument must be an s3:// path")
	}
}

func s3SyncUpload(ctx context.Context, cl *minio.Client, local string, dst s3Path, o *s3TransferOptions) error {
	base := filepath.Clean(local)
	if !isExistingDir(base) {
		return fmt.Errorf("%s is not a directory", local)
	}

	remote := map[string]minio.ObjectInfo{}
	if objs, err := s3ListAll(ctx, cl, dst.bucket, dst.key, true, o.UseV1); err == nil {
		for _, ob := range objs {
			remote[ob.Key] = ob
		}
	}

	localSeen := map[string]bool{}
	var jobs []func() error
	err := filepath.Walk(base, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		rel, err := filepath.Rel(base, p)
		if err != nil {
			return err
		}
		key := s3JoinKey(dst.key, filepath.ToSlash(rel))
		localSeen[key] = true
		if !s3MatchFilters(key, o.Include, o.Exclude) {
			return nil
		}
		if ob, ok := remote[key]; ok {
			if ob.Size == info.Size() && (!o.ExactTimestamps || !ob.LastModified.After(info.ModTime().Add(time.Second))) {
				return nil
			}
		}
		jobs = append(jobs, func() error { return s3UploadFile(ctx, cl, p, dst.bucket, key, o) })
		return nil
	})
	if err != nil {
		return err
	}
	if err := s3Parallel(o.Concurrency, jobs); err != nil {
		return err
	}

	if o.Delete {
		for key := range remote {
			if localSeen[key] {
				continue
			}
			if !s3MatchFilters(key, o.Include, o.Exclude) {
				continue
			}
			if o.DryRun {
				fmt.Printf("delete s3://%s/%s (dry-run)\n", dst.bucket, key)
				continue
			}
			if err := cl.RemoveObject(ctx, dst.bucket, key, minio.RemoveObjectOptions{}); err != nil {
				return err
			}
			fmt.Printf("delete s3://%s/%s\n", dst.bucket, key)
		}
	}
	return nil
}

func s3SyncDownload(ctx context.Context, cl *minio.Client, src s3Path, local string, o *s3TransferOptions) error {
	if !o.DryRun {
		if err := os.MkdirAll(local, 0755); err != nil {
			return err
		}
	}
	objs, err := s3ListAll(ctx, cl, src.bucket, src.key, true, o.UseV1)
	if err != nil {
		return err
	}

	remoteSeen := map[string]bool{}
	var jobs []func() error
	for _, ob := range objs {
		if strings.HasSuffix(ob.Key, "/") {
			continue
		}
		rel := strings.TrimPrefix(ob.Key, src.key)
		target := filepath.Join(local, filepath.FromSlash(rel))
		remoteSeen[target] = true
		if !s3MatchFilters(ob.Key, o.Include, o.Exclude) {
			continue
		}
		if fi, err := os.Stat(target); err == nil {
			if fi.Size() == ob.Size && (!o.ExactTimestamps || !ob.LastModified.After(fi.ModTime().Add(time.Second))) {
				continue
			}
		}
		jobs = append(jobs, func() error {
			return s3DownloadFile(ctx, cl, src.bucket, ob.Key, target, o)
		})
	}
	if err := s3Parallel(o.Concurrency, jobs); err != nil {
		return err
	}

	if o.Delete {
		return filepath.Walk(local, func(p string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return err
			}
			if remoteSeen[p] || !s3MatchFilters(filepath.ToSlash(p), o.Include, o.Exclude) {
				return nil
			}
			if o.DryRun {
				fmt.Printf("delete %s (dry-run)\n", p)
				return nil
			}
			if err := os.Remove(p); err != nil {
				return err
			}
			fmt.Printf("delete %s\n", p)
			return nil
		})
	}
	return nil
}

func s3SyncRemote(ctx context.Context, cl *minio.Client, src, dst s3Path, o *s3TransferOptions) error {
	srcObjs, err := s3ListAll(ctx, cl, src.bucket, src.key, true, o.UseV1)
	if err != nil {
		return err
	}
	dstObjs, err := s3ListAll(ctx, cl, dst.bucket, dst.key, true, o.UseV1)
	if err != nil {
		return err
	}
	dstMap := map[string]minio.ObjectInfo{}
	for _, ob := range dstObjs {
		dstMap[ob.Key] = ob
	}

	srcSeen := map[string]bool{}
	var jobs []func() error
	for _, ob := range srcObjs {
		if strings.HasSuffix(ob.Key, "/") {
			continue
		}
		rel := strings.TrimPrefix(ob.Key, src.key)
		key := s3JoinKey(dst.key, rel)
		srcSeen[key] = true
		if !s3MatchFilters(ob.Key, o.Include, o.Exclude) {
			continue
		}
		if d, ok := dstMap[key]; ok {
			if d.Size == ob.Size && (!o.ExactTimestamps || !ob.LastModified.After(d.LastModified.Add(time.Second))) {
				continue
			}
		}
		jobs = append(jobs, func() error {
			return s3CopyObject(ctx, cl, src.bucket, ob.Key, dst.bucket, key, o)
		})
	}
	if err := s3Parallel(o.Concurrency, jobs); err != nil {
		return err
	}

	if o.Delete {
		for key := range dstMap {
			if srcSeen[key] || !s3MatchFilters(key, o.Include, o.Exclude) {
				continue
			}
			if o.DryRun {
				fmt.Printf("delete s3://%s/%s (dry-run)\n", dst.bucket, key)
				continue
			}
			if err := cl.RemoveObject(ctx, dst.bucket, key, minio.RemoveObjectOptions{}); err != nil {
				return err
			}
			fmt.Printf("delete s3://%s/%s\n", dst.bucket, key)
		}
	}
	return nil
}
