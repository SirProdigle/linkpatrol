package walker

import "context"

type Walker interface {
	Walk(ctx context.Context, uri string) error
}

type PathType int

const (
	PathTypeUrl PathType = iota
	PathTypeFile
	PathTypeEmail
	PathTypeTel
	PathTypeAnchor
	PathTypeRoot
	PathTypeFtp
	PathTypeGit
	PathTypeUnknown
	PathTypeRelativeFile
	PathTypeRelativeUrl
)

type WalkerResult struct {
	BasePath string
	Path     string
	Type     PathType
}
