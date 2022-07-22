package dbpg

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"github.com/lib/pq"
	"github.com/vulpemventures/neutrino-elements/pkg/repository"
)

const (
	uniqueViolation = "23505"
)

type filterRepositoryImpl struct {
	db *DbService
}

func NewFilterRepositoryImpl(db *DbService) (repository.FilterRepository, error) {
	return filterRepositoryImpl{
		db: db,
	}, nil
}

type Filter struct {
	Key   string `db:"filter_key"`
	Value []byte `db:"filter_value"`
}

func (f filterRepositoryImpl) PutFilter(
	ctx context.Context,
	entry *repository.FilterEntry,
) error {
	tx, err := f.db.Db.Beginx()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	filter := Filter{
		Key:   entry.Key.String(),
		Value: entry.NBytes,
	}

	_, err = tx.NamedExec(
		"INSERT INTO filter (filter_key, filter_value) VALUES "+
			"(:filter_key, :filter_value)",
		&filter,
	)
	if err != nil {
		if pqErr := err.(*pq.Error); pqErr != nil {
			if pqErr.Code == uniqueViolation {
				f, err := f.GetFilter(ctx, entry.Key)
				if err != nil {
					return fmt.Errorf("PutFilter -> failed to get filter: %w", err)
				}

				if !bytes.Equal(f.NBytes, entry.NBytes) {
					return fmt.Errorf("PutFilter -> filter already exists but with different value")
				}

				return nil
			}
		}
		return err
	}

	return tx.Commit()
}

func (f filterRepositoryImpl) GetFilter(
	ctx context.Context,
	key repository.FilterKey,
) (*repository.FilterEntry, error) {
	query := `select * from filter where filter_key=$1;`

	filter := &Filter{}
	err := f.db.Db.Get(filter, query, key.String())
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, repository.ErrFilterNotFound
		}

		return nil, err
	}

	return &repository.FilterEntry{
		Key:    key,
		NBytes: filter.Value,
	}, nil
}
