package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"railgorail/avito/internal/entity"
	"railgorail/avito/internal/lib"

	"github.com/jmoiron/sqlx"
)

type StatisticsRepo struct {
	db *sqlx.DB
}

func NewStatisticsRepo(db *sqlx.DB) *StatisticsRepo {
	return &StatisticsRepo{
		db: db,
	}
}

func (r *StatisticsRepo) GetAssignmentsCountStats(ctx context.Context, sort string) ([]*entity.UserStatistics, error) {
	const op = "pull_request_repo.GetAssignmentsCountStats"

	query := fmt.Sprintf(`
		SELECT u.id as user_id, u.name as username, COUNT(pr.pull_request_id) as assignment_count
		FROM users u
		LEFT JOIN pr_reviewers pr ON u.id = pr.user_id
		GROUP BY u.id, u.name
		ORDER BY assignment_count %s, u.name ASC
	`, sort)

	var stats []*entity.UserStatistics
	err := r.db.SelectContext(ctx, &stats, query)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []*entity.UserStatistics{}, nil
		}
		return nil, lib.Err(op, err)
	}

	return stats, nil
}

func (r *StatisticsRepo) GetPrStats(ctx context.Context) (*entity.PrStatistics, error) {
	const op = "pull_request_repo.GetPrStats"

	query := `
		SELECT
		COUNT(*) as pr_count,
		COUNT(CASE WHEN status = 'OPEN' THEN 1 END) as open_pr_count,
		COUNT(CASE WHEN status = 'MERGED' THEN 1 END) as merged_pr_count
		FROM pull_requests
	`

	var res entity.PrStatistics
	err := r.db.GetContext(ctx, &res, query)
	if err != nil {
		return nil, lib.Err(op, err)
	}
	return &res, nil
}
