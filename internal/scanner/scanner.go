package scanner

import (
	"os"
	"path/filepath"
	"strings"
)

type FileType int

const (
	FileTypeMarkdown FileType = iota
	FileTypeHTML
)

type FileInfo struct {
	FilePath string
	FileType FileType
}

func ScanDirectory(dir string) ([]string, []string, error) {
	var markdownFiles []string
	var htmlFiles []string
	
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}
		
		if strings.HasSuffix(d.Name(), ".md") || strings.HasSuffix(d.Name(), ".markdown") {
			markdownFiles = append(markdownFiles, path)
		} else if strings.HasSuffix(d.Name(), ".html") || strings.HasSuffix(d.Name(), ".htm") {
			htmlFiles = append(htmlFiles, path)
		}

		return nil
	})

	if err != nil {
		return nil, nil, err
	}

	return markdownFiles, htmlFiles, nil
}

func IsRelevantFile(filename string) bool {
	return strings.HasSuffix(filename, ".md") ||
		strings.HasSuffix(filename, ".markdown") ||
		strings.HasSuffix(filename, ".html") ||
		strings.HasSuffix(filename, ".htm")
}

func GetFileType(filename string) FileType {
	if strings.HasSuffix(filename, ".md") || strings.HasSuffix(filename, ".markdown") {
		return FileTypeMarkdown
	}
	return FileTypeHTML
}