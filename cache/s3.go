package cache

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// S3Backend S3 기반 캐시 백엔드
type S3Backend struct {
	client *s3.Client
	bucket string
	prefix string
	mu     sync.RWMutex
}

// S3Config S3 백엔드 설정
type S3Config struct {
	Bucket   string
	Prefix   string
	Region   string
	Endpoint string // MinIO 등 S3 호환 서비스용
}

// NewS3Backend 새 S3 백엔드 생성
func NewS3Backend(ctx context.Context, cfg S3Config) (*S3Backend, error) {
	// AWS SDK 설정
	awsCfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(cfg.Region),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// S3 클라이언트 옵션
	opts := func(o *s3.Options) {
		if cfg.Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
			o.UsePathStyle = true
		}
	}

	// S3 클라이언트 생성
	client := s3.NewFromConfig(awsCfg, opts)

	// 버킷 존재 확인
	_, err = client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(cfg.Bucket),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to access bucket %s: %w", cfg.Bucket, err)
	}

	return &S3Backend{
		client: client,
		bucket: cfg.Bucket,
		prefix: strings.TrimSuffix(cfg.Prefix, "/"),
	}, nil
}

// Get 캐시에서 데이터 읽기
func (s *S3Backend) Get(key string) (io.ReadCloser, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ctx := context.Background()
	objectKey := s.getObjectKey(key)

	// 메타데이터 먼저 확인
	meta, err := s.getMetadata(ctx, key)
	if err != nil {
		return nil, err
	}

	// TTL 확인
	if meta.TTL > 0 && time.Since(meta.CreatedAt) > meta.TTL {
		// 만료된 캐시 삭제
		go func() {
			if err := s.Delete(key); err != nil {
				// 만료된 캐시 삭제 실패 시 로그 (백그라운드 작업이므로 에러 무시)
			}
		}()
		return nil, fmt.Errorf("cache expired")
	}

	// 객체 가져오기
	result, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(objectKey),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get object: %w", err)
	}

	// 접근 시간 업데이트
	go s.updateAccessTime(key)

	return result.Body, nil
}

// Put 캐시에 데이터 저장
func (s *S3Backend) Put(key string, data io.Reader, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := context.Background()
	objectKey := s.getObjectKey(key)

	// 데이터를 메모리에 읽기 (크기 계산을 위해)
	buf := new(bytes.Buffer)
	size, err := io.Copy(buf, data)
	if err != nil {
		return fmt.Errorf("failed to read data: %w", err)
	}

	// S3에 업로드
	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(objectKey),
		Body:        bytes.NewReader(buf.Bytes()),
		ContentType: aws.String("application/octet-stream"),
	})
	if err != nil {
		return fmt.Errorf("failed to put object: %w", err)
	}

	// 메타데이터 저장
	metadata := &CacheMetadata{
		Key:        key,
		Size:       size,
		CreatedAt:  time.Now(),
		AccessedAt: time.Now(),
		TTL:        ttl,
	}

	return s.putMetadata(ctx, key, metadata)
}

// Exists 캐시 키 존재 여부 확인
func (s *S3Backend) Exists(key string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ctx := context.Background()
	objectKey := s.getObjectKey(key)

	// 객체 존재 확인
	_, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(objectKey),
	})
	if err != nil {
		return false
	}

	// 메타데이터 확인 및 TTL 체크
	meta, err := s.getMetadata(ctx, key)
	if err != nil {
		return false
	}

	// TTL 확인
	if meta.TTL > 0 && time.Since(meta.CreatedAt) > meta.TTL {
		return false
	}

	return true
}

// Delete 캐시 항목 삭제
func (s *S3Backend) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := context.Background()
	objectKey := s.getObjectKey(key)
	metaKey := s.getMetadataKey(key)

	// 객체 삭제
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(objectKey),
	})
	if err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}

	// 메타데이터 삭제
	_, err = s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(metaKey),
	})

	return err
}

// GetMetadata 캐시 메타데이터 조회
func (s *S3Backend) GetMetadata(key string) (*CacheMetadata, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ctx := context.Background()
	return s.getMetadata(ctx, key)
}

// Clear 전체 캐시 삭제
func (s *S3Backend) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := context.Background()

	// 프리픽스로 객체 목록 조회
	prefix := s.prefix
	if prefix != "" {
		prefix += "/"
	}

	paginator := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
		Prefix: aws.String(prefix),
	})

	// 모든 객체 삭제
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("failed to list objects: %w", err)
		}

		if len(page.Contents) == 0 {
			continue
		}

		// 삭제할 객체 목록 생성
		var objects []types.ObjectIdentifier
		for _, obj := range page.Contents {
			objects = append(objects, types.ObjectIdentifier{
				Key: obj.Key,
			})
		}

		// 배치 삭제
		_, err = s.client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
			Bucket: aws.String(s.bucket),
			Delete: &types.Delete{
				Objects: objects,
			},
		})
		if err != nil {
			return fmt.Errorf("failed to delete objects: %w", err)
		}
	}

	return nil
}

// Size 캐시 크기 조회
func (s *S3Backend) Size() (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ctx := context.Background()
	var totalSize int64

	// 프리픽스로 객체 목록 조회
	prefix := s.prefix
	if prefix != "" {
		prefix += "/"
	}

	paginator := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
		Prefix: aws.String(prefix),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return 0, fmt.Errorf("failed to list objects: %w", err)
		}

		for _, obj := range page.Contents {
			// 메타데이터 파일 제외
			if !strings.HasSuffix(*obj.Key, ".meta") {
				totalSize += *obj.Size
			}
		}
	}

	return totalSize, nil
}

// getObjectKey 캐시 객체 키 생성
func (s *S3Backend) getObjectKey(key string) string {
	if s.prefix == "" {
		return key
	}
	return fmt.Sprintf("%s/%s", s.prefix, key)
}

// getMetadataKey 메타데이터 객체 키 생성
func (s *S3Backend) getMetadataKey(key string) string {
	return s.getObjectKey(key) + ".meta"
}

// getMetadata 메타데이터 조회 (내부 사용)
func (s *S3Backend) getMetadata(ctx context.Context, key string) (*CacheMetadata, error) {
	metaKey := s.getMetadataKey(key)

	result, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(metaKey),
	})
	if err != nil {
		return nil, err
	}
	defer result.Body.Close()

	var meta CacheMetadata
	if err := json.NewDecoder(result.Body).Decode(&meta); err != nil {
		return nil, err
	}

	return &meta, nil
}

// putMetadata 메타데이터 저장 (내부 사용)
func (s *S3Backend) putMetadata(ctx context.Context, key string, meta *CacheMetadata) error {
	metaKey := s.getMetadataKey(key)

	data, err := json.Marshal(meta)
	if err != nil {
		return err
	}

	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(metaKey),
		Body:        bytes.NewReader(data),
		ContentType: aws.String("application/json"),
	})

	return err
}

// updateAccessTime 접근 시간 업데이트
func (s *S3Backend) updateAccessTime(key string) {
	ctx := context.Background()

	meta, err := s.getMetadata(ctx, key)
	if err != nil {
		return
	}

	meta.AccessedAt = time.Now()
	if err := s.putMetadata(ctx, key, meta); err != nil {
		// 메타데이터 업데이트 실패 시 에러 무시 (액세스 시간 업데이트는 선택적)
	}
}
