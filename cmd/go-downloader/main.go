package main

import (
	"fmt"
	"github.com/PauloSm/go-downloader/pkg/downloader"
	"github.com/PauloSm/go-downloader/pkg/storage"
	"os"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Missing params: go-downloader <URL> <File Path>")
		return
	}

	url := os.Args[1]
	savePath := os.Args[2]

	httpDownloader := downloader.NewHttpDownloader()

	localStorage := storage.NewLocalStorage()

	downloadService := downloader.NewService(httpDownloader, localStorage)

	err := downloadService.DownloadFile(url, savePath)
	if err != nil {
		fmt.Printf("Error when downloading the file: %v\n", err)
		return
	}

	fmt.Println("The download has finished", savePath)
}

