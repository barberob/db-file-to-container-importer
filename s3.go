package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/charmbracelet/huh"
)

type S3Config struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	Region    string
}

func loadS3Config() (*S3Config, error) {
	configDir := getConfigDir()
	data, err := os.ReadFile(filepath.Join(configDir, "s3config"))
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(data), "\n")
	if len(lines) < 5 {
		return nil, fmt.Errorf("incomplete S3 config")
	}

	return &S3Config{
		Endpoint:  lines[0],
		AccessKey: lines[1],
		SecretKey: lines[2],
		Bucket:    lines[3],
		Region:    lines[4],
	}, nil
}

func saveS3Config(cfg *S3Config) error {
	configDir := getConfigDir()
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}
	content := fmt.Sprintf("%s\n%s\n%s\n%s\n%s", cfg.Endpoint, cfg.AccessKey, cfg.SecretKey, cfg.Bucket, cfg.Region)
	return os.WriteFile(filepath.Join(configDir, "s3config"), []byte(content), 0600)
}

func createS3Client(cfg *S3Config) *s3.Client {
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL:           cfg.Endpoint,
			SigningRegion: cfg.Region,
		}, nil
	})

	awsCfg, _ := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(cfg.Region),
		config.WithEndpointResolverWithOptions(customResolver),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, "")),
	)

	return s3.NewFromConfig(awsCfg)
}

func promptS3Config() (*S3Config, error) {
	var cfg S3Config

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("S3 Endpoint URL").
				Placeholder("https://s3.amazonaws.com").
				Value(&cfg.Endpoint),
			huh.NewInput().
				Title("Bucket name").
				Value(&cfg.Bucket),
			huh.NewInput().
				Title("Region").
				Placeholder("fr-par").
				Value(&cfg.Region),
			huh.NewInput().
				Title("Access Key ID").
				Value(&cfg.AccessKey),
			huh.NewInput().
				Title("Secret Access Key").
				Password(true).
				Value(&cfg.SecretKey),
		),
	)

	if err := form.Run(); err != nil {
		return nil, err
	}

	// Apply defaults
	if cfg.Region == "" {
		cfg.Region = "fr-par"
	}
	if cfg.Endpoint == "" {
		cfg.Endpoint = "https://s3.amazonaws.com"
	}

	return &cfg, nil
}

func browseS3Files(client *s3.Client, bucket string) (string, error) {
	prefix := ""

	for {
		input := &s3.ListObjectsV2Input{
			Bucket:    aws.String(bucket),
			Prefix:    aws.String(prefix),
			Delimiter: aws.String("/"),
		}

		output, err := client.ListObjectsV2(context.Background(), input)
		if err != nil {
			return "", fmt.Errorf("failed to list S3 objects: %w", err)
		}

		var options []huh.Option[string]

		// Option to go up
		if prefix != "" {
			parentPrefix := ""
			if idx := strings.LastIndex(strings.TrimSuffix(prefix, "/"), "/"); idx >= 0 {
				parentPrefix = prefix[:idx+1]
			}
			options = append(options, huh.NewOption("📁 .. (dossier parent)", "__parent__"+parentPrefix))
		}

		// Common prefixes (folders)
		for _, commonPrefix := range output.CommonPrefixes {
			folderName := strings.TrimSuffix(strings.TrimPrefix(*commonPrefix.Prefix, prefix), "/")
			options = append(options, huh.NewOption("📁 "+folderName, "dir:"+*commonPrefix.Prefix))
		}

		// Files - sorted by modification time
		validExts := []string{".sql", ".sql.gz", ".dump"}
		type s3FileInfo struct {
			name     string
			fileName string
			modTime  int64
			size     int64
		}
		var files []s3FileInfo

		for _, obj := range output.Contents {
			name := *obj.Key
			// Skip the folder itself
			if name == prefix {
				continue
			}
			fileName := strings.TrimPrefix(name, prefix)
			for _, ext := range validExts {
				if strings.HasSuffix(fileName, ext) {
					modTime := int64(0)
					if obj.LastModified != nil {
						modTime = obj.LastModified.Unix()
					}
					size := int64(0)
					if obj.Size != nil {
						size = *obj.Size
					}
					files = append(files, s3FileInfo{name: name, fileName: fileName, modTime: modTime, size: size})
					break
				}
			}
		}

		// Sort by modification time (most recent first)
		for i := 0; i < len(files)-1; i++ {
			for j := i + 1; j < len(files); j++ {
				if files[i].modTime < files[j].modTime {
					files[i], files[j] = files[j], files[i]
				}
			}
		}

		for _, f := range files {
			display := fmt.Sprintf("📄 %s (%d bytes)", f.fileName, f.size)
			options = append(options, huh.NewOption(display, "file:"+f.name))
		}

		if len(options) == 0 {
			return "", fmt.Errorf("aucun fichier SQL trouvé dans s3://%s/%s", bucket, prefix)
		}

		var selected string
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title(fmt.Sprintf("☁️  s3://%s/%s", bucket, prefix)).
					Options(options...).
					Value(&selected).
					Height(20).
					Filtering(true),
			),
		)

		if err := form.Run(); err != nil {
			return "", err
		}

		switch {
		case strings.HasPrefix(selected, "__parent__"):
			prefix = strings.TrimPrefix(selected, "__parent__")
		case strings.HasPrefix(selected, "dir:"):
			prefix = strings.TrimPrefix(selected, "dir:")
		case strings.HasPrefix(selected, "file:"):
			return strings.TrimPrefix(selected, "file:"), nil
		}
	}
}

// progressReader wraps an io.Reader and reports progress
type progressReader struct {
	reader   io.Reader
	total    int64
	current  int64
	onUpdate func(percent float64)
}

func (pr *progressReader) Read(p []byte) (n int, err error) {
	n, err = pr.reader.Read(p)
	pr.current += int64(n)
	if pr.total > 0 {
		percent := float64(pr.current) / float64(pr.total)
		pr.onUpdate(percent)
	}
	return n, err
}

func downloadS3File(client *s3.Client, bucket, key string) (string, error) {
	// First, get file metadata to know the size
	headInput := &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	headOutput, err := client.HeadObject(context.Background(), headInput)
	if err != nil {
		return "", fmt.Errorf("failed to get file info from S3: %w", err)
	}

	fileSize := int64(0)
	if headOutput.ContentLength != nil {
		fileSize = *headOutput.ContentLength
	}

	// Now download the file with checksum validation
	input := &s3.GetObjectInput{
		Bucket:       aws.String(bucket),
		Key:          aws.String(key),
		ChecksumMode: types.ChecksumModeEnabled,
	}

	output, err := client.GetObject(context.Background(), input)
	if err != nil {
		return "", fmt.Errorf("failed to download from S3: %w", err)
	}
	defer output.Body.Close()

	// Create temp file with proper extension
	pattern := "dbimport-*.sql"
	if strings.HasSuffix(key, ".gz") {
		pattern = "dbimport-*.sql.gz"
	} else if strings.HasSuffix(key, ".dump") {
		pattern = "dbimport-*.dump"
	}

	tempFile, err := os.CreateTemp("", pattern)
	if err != nil {
		return "", err
	}
	defer tempFile.Close()

	// Create progress bar
	barWidth := 30
	lastPercent := -1

	// Print initial state at 0%
	emptyBar := strings.Repeat("░", barWidth)
	fmt.Printf("📥 Téléchargement: %s / %s [%s] 0%%", formatBytes(0), formatBytes(fileSize), emptyBar)

	// Create progress-tracking reader with throttled updates
	pr := &progressReader{
		reader: output.Body,
		total:  fileSize,
		onUpdate: func(percent float64) {
			// Only update every 1% to avoid flickering
			currentPercent := int(percent * 100)
			if currentPercent != lastPercent {
				lastPercent = currentPercent
				currentBytes := int64(float64(fileSize) * percent)

				// Build progress bar
				filled := int(percent * float64(barWidth))
				if filled > barWidth {
					filled = barWidth
				}
				bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

				fmt.Printf("\r📥 Téléchargement: %s / %s [%s] %d%%",
					formatBytes(currentBytes),
					formatBytes(fileSize),
					bar,
					currentPercent)
			}
		},
	}

	// Copy content
	_, err = io.Copy(tempFile, pr)
	if err != nil {
		os.Remove(tempFile.Name())
		return "", err
	}

	// Final update - show 100%
	bar := strings.Repeat("█", barWidth)
	fmt.Printf("\r📥 Téléchargement: %s / %s [%s] 100%% ✓\n",
		formatBytes(fileSize),
		formatBytes(fileSize),
		bar)

	return tempFile.Name(), nil
}

// formatBytes converts bytes to human-readable format
func formatBytes(bytes int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

func getFileFromS3() (string, func(), error) {
	s3Cfg, err := loadS3Config()
	if err != nil {
		s3Cfg, err = promptS3Config()
		if err != nil {
			return "", nil, err
		}
		if err := saveS3Config(s3Cfg); err != nil {
			return "", nil, fmt.Errorf("failed to save S3 config: %w", err)
		}
	}

	client := createS3Client(s3Cfg)
	key, err := browseS3Files(client, s3Cfg.Bucket)
	if err != nil {
		return "", nil, err
	}

	tempFile, err := downloadS3File(client, s3Cfg.Bucket, key)
	if err != nil {
		return "", nil, err
	}

	cleanup := func() {
		os.Remove(tempFile)
	}

	return tempFile, cleanup, nil
}
