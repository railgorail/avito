package stats

import (
	"context"
	"railgorail/avito/internal/entity"
	"railgorail/avito/internal/service"
	"railgorail/avito/internal/transport/http/dto"
)

//go:generate go run github.com/vektra/mockery/v2@v2.53.5 --name=StatsProvider
type StatsProvider interface {
	GetAssignmentsCountStats(ctx context.Context, sort string) ([]*entity.UserStatistics, error)
	GetPrStats(ctx context.Context) (*entity.PrStatistics, error)
}

type StatsService struct {
	statsProvider StatsProvider
	trm           service.TransactionManager
}

func NewStatsService(trm service.TransactionManager, statsProvider StatsProvider) *StatsService {
	return &StatsService{
		trm:           trm,
		statsProvider: statsProvider,
	}
}

func (s *StatsService) GetStatistics(ctx context.Context, sort string) (*dto.StatsResponse, error) {

	resp := &dto.StatsResponse{
		User: []dto.UserStats{},
	}

	err := s.trm.Do(ctx, func(ctx context.Context) error {
		userStats, err := s.statsProvider.GetAssignmentsCountStats(ctx, sort)
		if err != nil {
			return err
		}
		prStats, err := s.statsProvider.GetPrStats(ctx)
		if err != nil {
			return err
		}

		for _, u := range userStats {
			stat := dto.UserStats(*u)
			resp.User = append(resp.User, stat)
		}
		resp.Pr = dto.PrStats(*prStats)

		return nil
	})
	if err != nil {
		return nil, err
	}

	return resp, nil
}
