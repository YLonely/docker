package image // import "github.com/docker/docker/image"

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	"github.com/docker/docker/pkg/ioutils"
	"github.com/opencontainers/go-digest"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// DigestWalkFunc is function called by StoreBackend.Walk
type DigestWalkFunc func(id digest.Digest) error

// StoreBackend provides interface for image.Store persistence
type StoreBackend interface {
	Walk(f DigestWalkFunc) error
	Get(id digest.Digest, options *StoreBackendOptions) ([]byte, error)
	Set(data []byte) (digest.Digest, error)
	Delete(id digest.Digest) error
	SetMetadata(id digest.Digest, key string, data []byte) error
	GetMetadata(id digest.Digest, key string) ([]byte, error)
	DeleteMetadata(id digest.Digest, key string) error
}

// StoreBackendOptions holds params for metadata getting
type StoreBackendOptions struct {
	UseExtraStorage bool
}

// fs implements StoreBackend using the filesystem.
type fs struct {
	sync.RWMutex
	root             string
	extraStoragePath string
}

const (
	contentDirName  = "content"
	metadataDirName = "metadata"
)

// NewFSStoreBackend returns new filesystem based backend for image.Store
func NewFSStoreBackend(root, extraStoragePath string) (StoreBackend, error) {
	return newFSStore(root, extraStoragePath)
}

func newFSStore(root, extraStoragePath string) (*fs, error) {
	s := &fs{
		root:             root,
		extraStoragePath: extraStoragePath,
	}
	if err := os.MkdirAll(filepath.Join(root, contentDirName, string(digest.Canonical)), 0700); err != nil {
		return nil, errors.Wrap(err, "failed to create storage backend")
	}
	if err := os.MkdirAll(filepath.Join(root, metadataDirName, string(digest.Canonical)), 0700); err != nil {
		return nil, errors.Wrap(err, "failed to create storage backend")
	}
	return s, nil
}

func (s *fs) contentFile(dgst digest.Digest, useExtraStorageAsRoot bool) string {
	rootPath := s.root
	if useExtraStorageAsRoot {
		rootPath = s.extraStoragePath
	}
	return filepath.Join(rootPath, contentDirName, string(dgst.Algorithm()), dgst.Hex())
}

func (s *fs) metadataDir(dgst digest.Digest, useExtraStorageAsRoot bool) string {
	rootPath := s.root
	if useExtraStorageAsRoot {
		rootPath = s.extraStoragePath
	}
	return filepath.Join(rootPath, metadataDirName, string(dgst.Algorithm()), dgst.Hex())
}

// Walk calls the supplied callback for each image ID in the storage backend.
func (s *fs) Walk(f DigestWalkFunc) error {
	// Only Canonical digest (sha256) is currently supported
	s.RLock()
	dir, err := ioutil.ReadDir(filepath.Join(s.root, contentDirName, string(digest.Canonical)))
	s.RUnlock()
	if err != nil {
		return err
	}
	for _, v := range dir {
		dgst := digest.NewDigestFromHex(string(digest.Canonical), v.Name())
		if err := dgst.Validate(); err != nil {
			logrus.Debugf("skipping invalid digest %s: %s", dgst, err)
			continue
		}
		if err := f(dgst); err != nil {
			return err
		}
	}
	return nil
}

// Get returns the content stored under a given digest.
func (s *fs) Get(dgst digest.Digest, options *StoreBackendOptions) ([]byte, error) {
	s.RLock()
	defer s.RUnlock()

	return s.get(dgst, options)
}

func (s *fs) get(dgst digest.Digest, opt *StoreBackendOptions) ([]byte, error) {
	var content []byte
	var err error
	readOk := false
	if opt.UseExtraStorage {
		content, err = ioutil.ReadFile(s.contentFile(dgst, true))
		readOk = true
		if err == nil {
			// if use extra storage and we get what we want,
			// just make a symbol link from the extra dir
			// Or we should make a symbol link for metadata dir?
			extraPath := s.contentFile(dgst, true)
			symPath := s.contentFile(dgst, false)
			// do not create a symbol link if dir already exists.
			_, err = os.Stat(symPath)
			if os.IsNotExist(err) {
				if err = os.Symlink(extraPath, symPath); err != nil {
					return nil, errors.Wrapf(err, "cannot make symbol link for file %s", symPath)
				}
			} else if err != nil {
				return nil, errors.Wrapf(err, "stat file failed %s", symPath)
			}
		}
		// fallback to local dir
	}
	err = nil
	if !readOk {
		content, err = ioutil.ReadFile(s.contentFile(dgst, false))
	}
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get digest %s", dgst)
	}

	// todo: maybe optional
	if digest.FromBytes(content) != dgst {
		return nil, fmt.Errorf("failed to verify: %v", dgst)
	}

	return content, nil
}

// Set stores content by checksum.
// TODO: maybe we want to set content in the extra dir
func (s *fs) Set(data []byte) (digest.Digest, error) {
	s.Lock()
	defer s.Unlock()

	if len(data) == 0 {
		return "", fmt.Errorf("invalid empty data")
	}

	dgst := digest.FromBytes(data)
	if err := ioutils.AtomicWriteFile(s.contentFile(dgst, false), data, 0600); err != nil {
		return "", errors.Wrap(err, "failed to write digest data")
	}

	return dgst, nil
}

// Delete removes content and metadata files associated with the digest.
// Just delete the symbol link if the file is a link
func (s *fs) Delete(dgst digest.Digest) error {
	s.Lock()
	defer s.Unlock()

	if err := os.RemoveAll(s.metadataDir(dgst, false)); err != nil {
		return err
	}
	return os.Remove(s.contentFile(dgst, false))
}

// SetMetadata sets metadata for a given ID. It fails if there's no base file.
func (s *fs) SetMetadata(dgst digest.Digest, key string, data []byte) error {
	s.Lock()
	defer s.Unlock()
	if _, err := s.get(dgst, &StoreBackendOptions{}); err != nil {
		return err
	}

	baseDir := filepath.Join(s.metadataDir(dgst, false))
	if err := os.MkdirAll(baseDir, 0700); err != nil {
		return err
	}
	return ioutils.AtomicWriteFile(filepath.Join(s.metadataDir(dgst, false), key), data, 0600)
}

// GetMetadata returns metadata for a given digest.
// Maybe leave GetMetadata unchanged?
func (s *fs) GetMetadata(dgst digest.Digest, key string) ([]byte, error) {
	s.RLock()
	defer s.RUnlock()

	if _, err := s.get(dgst, &StoreBackendOptions{}); err != nil {
		return nil, err
	}
	bytes, err := ioutil.ReadFile(filepath.Join(s.metadataDir(dgst, false), key))
	if err != nil {
		return nil, errors.Wrap(err, "failed to read metadata")
	}
	return bytes, nil
}

// DeleteMetadata removes the metadata associated with a digest.
func (s *fs) DeleteMetadata(dgst digest.Digest, key string) error {
	s.Lock()
	defer s.Unlock()

	return os.RemoveAll(filepath.Join(s.metadataDir(dgst, false), key))
}
