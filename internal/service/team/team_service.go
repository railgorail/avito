package team

import (
	"context"

	"railgorail/avito/internal/entity"
	"railgorail/avito/internal/service"
	"railgorail/avito/internal/transport/http/dto"
)

type TeamProvider interface {
	Create(ctx context.Context, teamName string) (int, error)
	GetByTeamName(ctx context.Context, teamName string) (*entity.Team, error)
}

//go:generate go run github.com/vektra/mockery/v2@v2.53.5 --name=UserProvider
type UserProvider interface {
	Save(ctx context.Context, user *entity.User) (string, error)
	GetUsersInTeam(ctx context.Context, teamName string) ([]*entity.User, error)
	GetById(ctx context.Context, userID string) (*entity.User, error)
	SetIsActive(ctx context.Context, userID string, isActive bool) error
	GetActiveUsersIDInTeam(ctx context.Context, teamID int) ([]string, error)
}

//go:generate go run github.com/vektra/mockery/v2@v2.53.5 --name=PrProvider
type PrProvider interface {
	GetUserReviews(ctx context.Context, userID string) ([]*entity.PullRequest, error)
	GetPrReviewers(ctx context.Context, prID string) ([]string, error)
	ReassignReviewer(ctx context.Context, prID, oldUserID, newUserID string) error
	DeleteReviewer(ctx context.Context, prID, userID string) error
}

type TeamService struct {
	teamProvider TeamProvider
	userProvider UserProvider
	prProvider   PrProvider
	trm          service.TransactionManager
}

func NewTeamService(
	trm service.TransactionManager,
	teamProvider TeamProvider,
	userProvider UserProvider,
	prProvider PrProvider,
) *TeamService {
	return &TeamService{
		teamProvider: teamProvider,
		userProvider: userProvider,
		prProvider:   prProvider,
		trm:          trm,
	}
}

func (s *TeamService) Add(ctx context.Context, teamName string, users []dto.TeamMember) (*dto.TeamSchema, error) {
	resp := &dto.TeamSchema{}
	members := make([]dto.TeamMember, 0, len(users))

	err := s.trm.Do(ctx, func(ctx context.Context) error {
		teamID, err := s.teamProvider.Create(ctx, teamName)
		if err != nil {
			return err
		}

		for _, u := range users {
			user := &entity.User{
				ID:       u.UserID,
				Name:     u.Username,
				TeamID:   teamID,
				IsActive: u.IsActive,
			}

			_, err := s.userProvider.Save(ctx, user)
			if err != nil {
				return err
			}

			member := dto.TeamMember{
				UserID:   user.ID,
				Username: user.Name,
				IsActive: user.IsActive,
			}

			members = append(members, member)
		}

		resp.TeamName = teamName
		resp.Members = members

		return nil
	})
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (s *TeamService) Get(ctx context.Context, teamName string) (*dto.TeamSchema, error) {
	resp := &dto.TeamSchema{}

	_, err := s.teamProvider.GetByTeamName(ctx, teamName)
	if err != nil {
		return nil, err
	}

	users, err := s.userProvider.GetUsersInTeam(ctx, teamName)
	if err != nil {
		return nil, err
	}

	members := make([]dto.TeamMember, 0, len(users))
	for _, u := range users {
		member := dto.TeamMember{
			UserID:   u.ID,
			Username: u.Name,
			IsActive: u.IsActive,
		}

		members = append(members, member)
	}

	resp.TeamName = teamName
	resp.Members = members

	return resp, nil
}
