package downloader

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

type FileRepository interface {
	GetFileSize(url string) (int, error)
	DownloadChunk(url string, start, end int) ([]byte, error)
}

type HttpDownloader struct{}

func NewHttpDownloader() *HttpDownloader {
	return &HttpDownloader{}
}

func (h *HttpDownloader) GetFileSize(url string) (int, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Add("Range", "bytes=0-0")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	contentRange := resp.Header.Get("Content-Range")
	parts := strings.Split(contentRange, "/")
	if len(parts) < 2 {
		return 0, fmt.Errorf("Content-Range header is not in a valid format")
	}
	fileSize, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, err
	}

	return fileSize, nil
}

func (h *HttpDownloader) DownloadChunk(url string, start, end int) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", start, end))
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

type Storage interface {
	Save(path string, data []byte) error
}

type Service struct {
	repo    FileRepository
	storage Storage
}

func NewService(r FileRepository, s Storage) *Service {
	return &Service{repo: r, storage: s}
}

func (s *Service) DownloadFile(url, path string) error {
	size, err := s.repo.GetFileSize(url)
	if err != nil {
		return err
	}
	chunkSize := size / 10

	var wg sync.WaitGroup
	chunks := make([][]byte, 10)
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			start := i * chunkSize
			end := start + chunkSize - 1
			if i == 9 {
				end = size
			}
			chunk, _ := s.repo.DownloadChunk(url, start, end)
			chunks[i] = chunk
		}(i)
	}
	wg.Wait()

	var data []byte
	for _, chunk := range chunks {
		data = append(data, chunk...)
	}

	return s.storage.Save(path, data)
}
