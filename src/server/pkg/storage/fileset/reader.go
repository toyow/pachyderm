package fileset

import (
	"context"
	"io"

	"github.com/gogo/protobuf/proto"
	"github.com/pachyderm/pachyderm/src/server/pkg/storage/chunk"
	"github.com/pachyderm/pachyderm/src/server/pkg/storage/fileset/index"
)

// Reader is an abstraction for reading a fileset.
type Reader struct {
	store              Store
	chunks             *chunk.Storage
	path               string
	indexOpts          []index.Option
	additive, deletive *index.Reader
}

func newReader(store Store, chunks *chunk.Storage, p string, opts ...index.Option) *Reader {
	r := &Reader{
		store:     store,
		chunks:    chunks,
		path:      p,
		indexOpts: opts,
	}
	return r
}

func (r *Reader) setup(ctx context.Context) error {
	if r.additive == nil {
		md, err := r.store.Get(ctx, r.path)
		if err != nil {
			return err
		}
		r.additive = index.NewReader(r.chunks, md.Additive, r.indexOpts...)
		r.deletive = index.NewReader(r.chunks, md.Deletive, r.indexOpts...)
	}
	return nil
}

// Iterate iterates over the files in the file set.
func (r *Reader) Iterate(ctx context.Context, cb func(File) error, deletive ...bool) error {
	if err := r.setup(ctx); err != nil {
		return err
	}
	if len(deletive) > 0 && deletive[0] {
		return r.deletive.Iterate(ctx, func(idx *index.Index) error {
			return cb(newFileReader(ctx, r.chunks, idx))
		})
	}
	return r.additive.Iterate(ctx, func(idx *index.Index) error {
		return cb(newFileReader(ctx, r.chunks, idx))
	})
}

// FileReader is an abstraction for reading a file.
type FileReader struct {
	ctx    context.Context
	chunks *chunk.Storage
	idx    *index.Index
}

func newFileReader(ctx context.Context, chunks *chunk.Storage, idx *index.Index) *FileReader {
	return &FileReader{
		ctx:    ctx,
		chunks: chunks,
		idx:    proto.Clone(idx).(*index.Index),
	}
}

// Index returns the index for the file.
func (fr *FileReader) Index() *index.Index {
	return proto.Clone(fr.idx).(*index.Index)
}

// Content writes the content of the file.
func (fr *FileReader) Content(w io.Writer) error {
	dataRefs := getDataRefs(fr.idx.File.Parts)
	r := fr.chunks.NewReader(fr.ctx, dataRefs)
	return r.Get(w)
}
