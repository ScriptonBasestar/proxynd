package cache

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFileSystemBackend(t *testing.T) {
	// 임시 디렉토리 생성
	tempDir, err := os.MkdirTemp("", "cache_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)
	
	// 파일 시스템 백엔드 생성
	backend, err := NewFileSystemBackend(tempDir)
	if err != nil {
		t.Fatal(err)
	}
	
	t.Run("Put and Get", func(t *testing.T) {
		key := "test/file.txt"
		data := []byte("Hello, Cache!")
		
		// 데이터 저장
		err := backend.Put(key, bytes.NewReader(data), time.Hour)
		if err != nil {
			t.Fatal(err)
		}
		
		// 데이터 읽기
		reader, err := backend.Get(key)
		if err != nil {
			t.Fatal(err)
		}
		defer reader.Close()
		
		// 내용 확인
		result, err := io.ReadAll(reader)
		if err != nil {
			t.Fatal(err)
		}
		
		if !bytes.Equal(result, data) {
			t.Errorf("Expected %s, got %s", data, result)
		}
	})
	
	t.Run("Exists", func(t *testing.T) {
		key := "test/exists.txt"
		
		// 존재하지 않는 키
		if backend.Exists(key) {
			t.Error("Key should not exist")
		}
		
		// 키 저장
		backend.Put(key, bytes.NewReader([]byte("test")), time.Hour)
		
		// 존재하는 키
		if !backend.Exists(key) {
			t.Error("Key should exist")
		}
	})
	
	t.Run("Delete", func(t *testing.T) {
		key := "test/delete.txt"
		
		// 데이터 저장
		backend.Put(key, bytes.NewReader([]byte("delete me")), time.Hour)
		
		// 삭제
		err := backend.Delete(key)
		if err != nil {
			t.Fatal(err)
		}
		
		// 존재 확인
		if backend.Exists(key) {
			t.Error("Key should not exist after deletion")
		}
	})
	
	t.Run("TTL Expiration", func(t *testing.T) {
		key := "test/ttl.txt"
		
		// 짧은 TTL로 저장
		backend.Put(key, bytes.NewReader([]byte("expire soon")), 100*time.Millisecond)
		
		// 즉시 읽기 - 성공해야 함
		if !backend.Exists(key) {
			t.Error("Key should exist immediately after creation")
		}
		
		// TTL 만료 대기
		time.Sleep(200 * time.Millisecond)
		
		// 만료 후 읽기 - 실패해야 함
		if backend.Exists(key) {
			t.Error("Key should not exist after TTL expiration")
		}
	})
	
	t.Run("Metadata", func(t *testing.T) {
		key := "test/metadata.txt"
		data := []byte("metadata test")
		
		// 데이터 저장
		backend.Put(key, bytes.NewReader(data), time.Hour)
		
		// 메타데이터 조회
		meta, err := backend.GetMetadata(key)
		if err != nil {
			t.Fatal(err)
		}
		
		if meta.Key != key {
			t.Errorf("Expected key %s, got %s", key, meta.Key)
		}
		
		if meta.Size != int64(len(data)) {
			t.Errorf("Expected size %d, got %d", len(data), meta.Size)
		}
		
		if meta.TTL != time.Hour {
			t.Errorf("Expected TTL %v, got %v", time.Hour, meta.TTL)
		}
	})
	
	t.Run("Clear", func(t *testing.T) {
		// 여러 파일 저장
		for i := 0; i < 5; i++ {
			key := filepath.Join("test", "clear", string(rune('a'+i))+".txt")
			backend.Put(key, bytes.NewReader([]byte("data")), time.Hour)
		}
		
		// 전체 삭제
		err := backend.Clear()
		if err != nil {
			t.Fatal(err)
		}
		
		// 크기 확인
		size, err := backend.Size()
		if err != nil {
			t.Fatal(err)
		}
		
		if size != 0 {
			t.Errorf("Expected size 0 after clear, got %d", size)
		}
	})
}