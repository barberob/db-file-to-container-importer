package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
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

func downloadS3File(client *s3.Client, bucket, key string) (string, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
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

	// Copy content
	_, err = tempFile.ReadFrom(output.Body)
	if err != nil {
		os.Remove(tempFile.Name())
		return "", err
	}

	return tempFile.Name(), nil
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

	fmt.Printf("📥 Téléchargement de s3://%s/%s...\n", s3Cfg.Bucket, key)
	tempFile, err := downloadS3File(client, s3Cfg.Bucket, key)
	if err != nil {
		return "", nil, err
	}

	cleanup := func() {
		os.Remove(tempFile)
	}

	return tempFile, cleanup, nil
}
