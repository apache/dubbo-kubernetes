package storage

import (
	"bytes"
	"context"
	"io"

	"github.com/apache/dubbo-kubernetes/pkg/bufman/dal"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/model"
)

type DBStorageHelperImpl struct {
}

func NewDBStorageHelper() *DBStorageHelperImpl {
	return &DBStorageHelperImpl{}
}

func (helper *DBStorageHelperImpl) StoreBlob(ctx context.Context, file *model.CommitFile) error {
	return dal.FileBlob.Create(&model.FileBlob{
		Digest:  file.Digest,
		Content: file.Content,
	})
}

func (helper *DBStorageHelperImpl) StoreManifest(ctx context.Context, manifest *model.CommitFile) error {
	return dal.FileBlob.Create(&model.FileBlob{
		Digest:  manifest.Digest,
		Content: manifest.Content,
	})
}

func (helper *DBStorageHelperImpl) StoreDocumentation(ctx context.Context, file *model.CommitFile) error {
	return dal.FileBlob.Create(&model.FileBlob{
		Digest:  file.Digest,
		Content: file.Content,
	})
}

func (helper *DBStorageHelperImpl) ReadBlobToReader(ctx context.Context, digest string) (io.Reader, error) {
	content, err := helper.ReadManifest(ctx, digest)
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(content), nil
}

func (helper *DBStorageHelperImpl) ReadBlob(ctx context.Context, digest string) ([]byte, error) {
	blob, err := dal.FileBlob.Where(dal.FileBlob.Digest.Eq(digest)).First()
	if err != nil {
		return nil, err
	}

	return blob.Content, nil
}

func (helper *DBStorageHelperImpl) ReadManifestToReader(ctx context.Context, digest string) (io.Reader, error) {
	content, err := helper.ReadManifest(ctx, digest)
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(content), nil
}

func (helper *DBStorageHelperImpl) ReadManifest(ctx context.Context, digest string) ([]byte, error) {
	blob, err := dal.FileBlob.Where(dal.FileBlob.Digest.Eq(digest)).First()
	if err != nil {
		return nil, err
	}

	return blob.Content, nil
}
