package service

import (
	"context"
	"slices"

	"bito/internal/economy"
	"bito/internal/model"

	"github.com/google/uuid"
)

type ShopRepository interface {
	Buy(ctx context.Context, userID uuid.UUID, item string, price int64) (int64, error)
	Owned(ctx context.Context, userID uuid.UUID) ([]string, error)
	Look(ctx context.Context, userID uuid.UUID) (model.Look, bool, error)
	SaveLook(ctx context.Context, userID uuid.UUID, look model.Look) error
	Avatar(ctx context.Context, userID uuid.UUID) (string, bool, error)
	SaveAvatar(ctx context.Context, userID uuid.UUID, avatar string) error
	Avatars(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]string, error)
}

type ShopState struct {
	Owned     []string
	Look      model.Look
	LookSaved bool
	Avatar    string
}

type ShopService struct {
	repo ShopRepository
}

func NewShopService(repo ShopRepository) *ShopService {
	return &ShopService{repo: repo}
}

func defaultLook() model.Look {
	return model.Look{
		Theme: economy.DefaultLook[economy.KindTheme],
		Felt:  economy.DefaultLook[economy.KindFelt],
		Back:  economy.DefaultLook[economy.KindBack],
	}
}

func (s *ShopService) State(ctx context.Context, me uuid.UUID) (ShopState, error) {
	owned, err := s.repo.Owned(ctx, me)
	if err != nil {
		return ShopState{}, err
	}
	look, lookFound, err := s.repo.Look(ctx, me)
	if err != nil {
		return ShopState{}, err
	}
	if !lookFound {
		look = defaultLook()
	}
	avatar, found, err := s.repo.Avatar(ctx, me)
	if err != nil {
		return ShopState{}, err
	}
	if !found {
		avatar = economy.DefaultLook[economy.KindAvatar]
	}
	return ShopState{Owned: owned, Look: look, LookSaved: lookFound, Avatar: avatar}, nil
}

func (s *ShopService) Buy(ctx context.Context, me uuid.UUID, key string) (int64, error) {
	item, ok := economy.ItemByKey(key)
	if !ok {
		return 0, model.ErrUnknownItem
	}
	if item.Price == 0 {
		return 0, model.ErrAlreadyOwned
	}
	return s.repo.Buy(ctx, me, item.Key(), item.Price)
}

func (s *ShopService) Wear(ctx context.Context, me uuid.UUID, look model.Look) error {
	owned, err := s.repo.Owned(ctx, me)
	if err != nil {
		return err
	}
	has := make(map[string]bool, len(owned))
	for _, key := range owned {
		has[key] = true
	}

	for kind, id := range map[string]string{economy.KindTheme: look.Theme, economy.KindFelt: look.Felt, economy.KindBack: look.Back} {
		item, ok := economy.ItemByKey(kind + ":" + id)
		if !ok {
			return model.ErrUnknownItem
		}
		if item.Price > 0 && !has[item.Key()] {
			return model.ErrNotOwned
		}
	}
	return s.repo.SaveLook(ctx, me, look)
}

const maxAvatarLookup = 50

func (s *ShopService) SetAvatar(ctx context.Context, me uuid.UUID, id string) error {
	item, ok := economy.ItemByKey(economy.KindAvatar + ":" + id)
	if !ok {
		return model.ErrUnknownItem
	}
	if item.Price > 0 {
		owned, err := s.repo.Owned(ctx, me)
		if err != nil {
			return err
		}
		if !slices.Contains(owned, item.Key()) {
			return model.ErrNotOwned
		}
	}
	return s.repo.SaveAvatar(ctx, me, id)
}

func (s *ShopService) Avatars(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]string, error) {
	if len(ids) > maxAvatarLookup {
		ids = ids[:maxAvatarLookup]
	}
	saved, err := s.repo.Avatars(ctx, ids)
	if err != nil {
		return nil, err
	}
	avatars := make(map[uuid.UUID]string, len(ids))
	for _, id := range ids {
		avatar, ok := saved[id]
		if !ok {
			avatar = economy.DefaultLook[economy.KindAvatar]
		}
		avatars[id] = avatar
	}
	return avatars, nil
}
