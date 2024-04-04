package downloader

import (
	"errors"
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
		return 0, errors.New("Content-Range header is not in a valid format")
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
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

type Storage interface {
	Save(path string, data []byte) error
	Read(path string) ([]byte, error)
	Delete(path string) error
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
	chunkSize := calculateChunkSize(size)

	var wg sync.WaitGroup
	errChan := make(chan error, 1)
	for i := 0; i < numChunks(size, chunkSize); i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			start := i * chunkSize
			end := (i+1)*chunkSize - 1
			if i == numChunks(size, chunkSize)-1 {
				end = size - 1
			}
			chunk, err := s.repo.DownloadChunk(url, start, end)
			if err != nil {
				select {
				case errChan <- err:
				default:
				}
				return
			}
			chunkPath := fmt.Sprintf("%s.part%d", path, i)
			err = s.storage.Save(chunkPath, chunk)
			if err != nil {
				select {
				case errChan <- err:
				default:
				}
				return
			}
		}(i)
	}

	go func() {
		wg.Wait()
		close(errChan)
	}()

	if err, ok := <-errChan; ok {
		return err
	}

	for i := 0; i < numChunks(size, chunkSize); i++ {
		chunkPath := fmt.Sprintf("%s.part%d", path, i)
		chunkData, err := s.storage.Read(chunkPath)
		if err != nil {
			return err
		}
		err = s.storage.Save(path, chunkData)
		if err != nil {
			return err
		}
		err = s.storage.Delete(chunkPath)
		if err != nil {
			return err
		}
	}

	return nil
}

func calculateChunkSize(size int) int {
	const minChunkSize = 1024 * 1024
	const idealChunks = 10

	chunkSize := size / idealChunks
	if chunkSize < minChunkSize {
		chunkSize = minChunkSize
	}

	return chunkSize
}

func numChunks(size, chunkSize int) int {
	chunks := size / chunkSize
	if size%chunkSize != 0 {
		chunks++
	}
	return chunks
}
