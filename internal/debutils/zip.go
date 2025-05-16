package debutils

import (
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
)

func Decompress(inFile string) ([]string, error) {

	gzFile, err := os.Open(inFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open gz file: %v", err)
	}
	defer gzFile.Close()

	decompressedFile := "Packages"
	outDecompressed, err := os.Create(decompressedFile)
	if err != nil {
		return nil, fmt.Errorf("failed to create decompressed file: %v", err)
	}
	defer outDecompressed.Close()

	gzReader, err := gzip.NewReader(gzFile)
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %v", err)
	}
	defer gzReader.Close()

	_, err = io.Copy(outDecompressed, gzReader)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress file: %v", err)
	}

	return []string{decompressedFile}, nil
}

func Download(repoURL string) ([]string, error) {

	resp, err := http.Get(repoURL)
	if err != nil {
		return nil, fmt.Errorf("failed to download repo config: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected HTTP status: %s", resp.Status)
	}

	// Store the .gz file locally
	outFile := "Packages.gz"
	out, err := os.Create(outFile)
	if err != nil {
		return nil, fmt.Errorf("failed to create file: %v", err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to write to file: %v", err)
	}

	return []string{outFile}, nil
}
