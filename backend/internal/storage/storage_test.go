package storage

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Exercise the real adapters, including the SDK uploader, without cloud credentials
// or external services. The HTTP fixture implements only the object operations used
// here; it is not an S3/MinIO server compatibility test.
type storageFixture struct {
	providers [2]StorageProvider
	seed      func(string, []byte)
}

func newStorageFixture(t *testing.T, provider string) storageFixture {
	t.Helper()
	if provider == "local" {
		dir := t.TempDir()
		var f storageFixture
		for i := range f.providers {
			p, err := NewLocalStorage(dir)
			if err != nil {
				t.Fatal(err)
			}
			f.providers[i] = p
		}
		f.seed = func(key string, data []byte) {
			if err := os.WriteFile(filepath.Join(dir, key), data, 0600); err != nil {
				t.Fatal(err)
			}
		}
		return f
	}
	objects := &objectFixture{objects: map[string][]byte{}, uploads: map[string]map[int][]byte{}}
	server := httptest.NewServer(http.HandlerFunc(objects.serveHTTP))
	t.Cleanup(server.Close)
	var f storageFixture
	for i := range f.providers {
		client := s3.New(s3.Options{
			Region: "us-east-1", Credentials: aws.AnonymousCredentials{},
			BaseEndpoint: aws.String(server.URL), UsePathStyle: true,
			HTTPClient: server.Client(), RetryMaxAttempts: 1,
		})
		if provider == "s3" {
			f.providers[i] = NewS3Storage(client, "test-bucket")
		} else {
			// Skip bucket provisioning: R2 concerns object operations, not startup.
			f.providers[i] = &MinIOStorage{Client: client, BucketName: "test-bucket"}
		}
	}
	f.seed = func(key string, data []byte) {
		objects.mu.Lock()
		defer objects.mu.Unlock()
		objects.objects["/test-bucket/"+key] = bytes.Clone(data)
	}
	return f
}

type objectFixture struct {
	mu         sync.Mutex
	objects    map[string][]byte
	uploads    map[string]map[int][]byte
	nextUpload int
}

func (s *objectFixture) serveHTTP(w http.ResponseWriter, r *http.Request) {
	data, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "body read failed", 400)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	query := r.URL.Query()
	uploadID := query.Get("uploadId")
	if r.Method == http.MethodPost && query.Has("uploads") {
		s.nextUpload++
		id := strconv.Itoa(s.nextUpload)
		s.uploads[r.URL.Path+"/"+id] = map[int][]byte{}
		w.Header().Set("Content-Type", "application/xml")
		fmt.Fprintf(w, `<InitiateMultipartUploadResult><UploadId>%s</UploadId></InitiateMultipartUploadResult>`, id)
		return
	}
	if uploadID != "" {
		key := r.URL.Path + "/" + uploadID
		parts, ok := s.uploads[key]
		if !ok {
			http.NotFound(w, r)
			return
		}
		switch r.Method {
		case http.MethodPut:
			number, err := strconv.Atoi(query.Get("partNumber"))
			if err != nil {
				http.Error(w, "invalid part", 400)
				return
			}
			parts[number] = bytes.Clone(data)
			w.Header().Set("ETag", `"fixture-part"`)
		case http.MethodPost:
			var completed struct {
				Parts []struct {
					Number int `xml:"PartNumber"`
				} `xml:"Part"`
			}
			if err := xml.Unmarshal(data, &completed); err != nil {
				http.Error(w, "invalid completion", 400)
				return
			}
			var object []byte
			for _, part := range completed.Parts {
				value, ok := parts[part.Number]
				if !ok {
					http.Error(w, "missing part", 400)
					return
				}
				object = append(object, value...)
			}
			s.objects[r.URL.Path] = object
			delete(s.uploads, key)
			w.Header().Set("Content-Type", "application/xml")
			io.WriteString(w, `<CompleteMultipartUploadResult><ETag>"fixture-object"</ETag></CompleteMultipartUploadResult>`)
		case http.MethodDelete:
			delete(s.uploads, key)
			w.WriteHeader(http.StatusNoContent)
		default:
			http.Error(w, "unexpected multipart operation", 400)
		}
		return
	}
	switch r.Method {
	case http.MethodPut:
		s.objects[r.URL.Path] = bytes.Clone(data)
		w.Header().Set("ETag", `"fixture-object"`)
	case http.MethodGet:
		value, ok := s.objects[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Length", strconv.Itoa(len(value)))
		w.Write(value)
	case http.MethodDelete:
		delete(s.objects, r.URL.Path)
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "unexpected object operation", 400)
	}
}

func TestStorageConcurrentUploads(t *testing.T) {
	for _, provider := range []string{"local", "s3", "minio"} {
		t.Run(provider, func(t *testing.T) {
			f := newStorageFixture(t, provider)
			testConcurrentObjects(t, f, 32, func(i int) []byte { return []byte(fmt.Sprintf("print(%d)\n", i)) })
		})
	}
}

func TestStorageMultipartUploads(t *testing.T) {
	for _, provider := range []string{"s3", "minio"} {
		t.Run(provider, func(t *testing.T) {
			f := newStorageFixture(t, provider)
			// The SDK's default 5 MiB part size is below the application's 10 MiB
			// attachment limit. Distinct bytes in every part detect interleaving.
			testConcurrentObjects(t, f, 2, func(i int) []byte { return bytes.Repeat([]byte{byte('A' + i)}, 6*1024*1024+17) })
		})
	}
}

func testConcurrentObjects(t *testing.T, f storageFixture, count int, payload func(int) []byte) {
	t.Helper()
	type result struct {
		key  string
		data []byte
		err  error
	}
	results := make(chan result, count)
	start := make(chan struct{})
	for i := range count {
		go func(i int) {
			data := payload(i)
			<-start
			key, err := f.providers[i%len(f.providers)].Upload(context.Background(), data, "answer.py")
			results <- result{key, data, err}
		}(i)
	}
	close(start)
	var uploaded []result
	seen := map[string]bool{}
	for range count {
		r := <-results
		if r.err != nil {
			t.Errorf("upload failed: %v", r.err)
			continue
		}
		if r.key == "" || seen[r.key] {
			t.Errorf("empty or reused key %q", r.key)
		}
		seen[r.key] = true
		uploaded = append(uploaded, r)
	}
	if t.Failed() {
		return
	}
	for _, r := range uploaded {
		data, err := f.providers[1].Download(context.Background(), r.key)
		if err != nil || !bytes.Equal(data, r.data) {
			t.Fatalf("download changed another upload: %v", err)
		}
		text, err := f.providers[0].ReadText(context.Background(), r.key)
		if err != nil || text != string(r.data) {
			t.Fatalf("read changed another upload: %v", err)
		}
	}
	if err := f.providers[1].Delete(context.Background(), uploaded[0].key); err != nil {
		t.Fatal(err)
	}
	if _, err := f.providers[0].Download(context.Background(), uploaded[0].key); err == nil {
		t.Error("deleted object still exists")
	}
	for _, r := range uploaded[1:] {
		data, err := f.providers[0].Download(context.Background(), r.key)
		if err != nil || !bytes.Equal(data, r.data) {
			t.Fatalf("deleting one object changed another: %v", err)
		}
	}
}

func TestStorageFilenameIndependence(t *testing.T) {
	for _, provider := range []string{"local", "s3", "minio"} {
		t.Run(provider, func(t *testing.T) {
			f := newStorageFixture(t, provider)
			for _, name := range []string{"", "answer..py", "../answer.py", `..\..\answer.py`, "คำตอบ.py", strings.Repeat("long", 100)} {
				key, err := f.providers[0].Upload(context.Background(), []byte("answer"), name)
				if err != nil {
					t.Errorf("upload with filename %q: %v", name, err)
					continue
				}
				if strings.ContainsAny(key, `/\`) || strings.Contains(key, "..") {
					t.Errorf("filename affected object path: %q", key)
					continue
				}
				text, err := f.providers[1].ReadText(context.Background(), key)
				if err != nil || text != "answer" {
					t.Errorf("cannot read returned key %q: %v", key, err)
				}
				if err := f.providers[1].Delete(context.Background(), key); err != nil {
					t.Error(err)
				}
			}
			key, err := f.providers[0].Upload(context.Background(), nil, "empty.py")
			if err != nil {
				t.Fatal(err)
			}
			data, err := f.providers[1].Download(context.Background(), key)
			if err != nil || len(data) != 0 {
				t.Errorf("empty object changed: %v", err)
			}
		})
	}
}

func TestStorageLegacyKeys(t *testing.T) {
	for _, provider := range []string{"local", "s3", "minio"} {
		t.Run(provider, func(t *testing.T) {
			f := newStorageFixture(t, provider)
			key := "1729737600-original.py"
			f.seed(key, []byte("legacy answer"))
			data, err := f.providers[0].Download(context.Background(), key)
			if err != nil || string(data) != "legacy answer" {
				t.Fatalf("legacy download failed: %v", err)
			}
			text, err := f.providers[1].ReadText(context.Background(), key)
			if err != nil || text != "legacy answer" {
				t.Fatalf("legacy read failed: %v", err)
			}
			if err := f.providers[1].Delete(context.Background(), key); err != nil {
				t.Fatal(err)
			}
			if _, err := f.providers[0].Download(context.Background(), key); err == nil {
				t.Error("legacy deletion failed")
			}
		})
	}
}
