package service

import (
	"context"
	"fmt"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

var (
	ErrImages2Disabled           = infraerrors.Forbidden("IMAGES2_DISABLED", "Images 2 feature is disabled")
	ErrImages2InsufficientFunds  = infraerrors.Forbidden("IMAGES2_INSUFFICIENT_BALANCE", "Insufficient balance for image generation")
	ErrImages2GroupNotConfigured = infraerrors.NotFound("IMAGES2_GROUP_NOT_FOUND", "Images 2 target group is not configured")
)

type Images2Service struct {
	settings *SettingService
	apiKeys  *APIKeyService
	users    *UserService
	groups   *GroupService
}

type Images2GenerateRequest struct {
	Prompt   string `json:"prompt"`
	ImageURL string `json:"image_url,omitempty"`
	Size     string `json:"size,omitempty"`
}

type Images2GenerateResponse struct {
	AppliedPrice float64 `json:"applied_price"`
	Balance      float64 `json:"balance"`
}

type Images2PreparedRequest struct {
	Settings     *PublicSettings
	User         *User
	Group        *Group
	APIKey       *APIKey
	Subscription *UserSubscription
	Prompt       string
	ModelName    string
	ImageURL     string
	Size         string
}

func NewImages2Service(settings *SettingService, apiKeys *APIKeyService, users *UserService, groups *GroupService) *Images2Service {
	return &Images2Service{
		settings: settings,
		apiKeys:  apiKeys,
		users:    users,
		groups:   groups,
	}
}

func (s *Images2Service) Prepare(ctx context.Context, userID int64, req Images2GenerateRequest) (*Images2PreparedRequest, error) {
	if strings.TrimSpace(req.Prompt) == "" {
		return nil, infraerrors.BadRequest("IMAGES2_PROMPT_REQUIRED", "prompt is required")
	}
	size, err := normalizeImages2Size(req.Size)
	if err != nil {
		return nil, err
	}

	settings, err := s.settings.GetPublicSettings(ctx)
	if err != nil {
		return nil, err
	}
	if !settings.Images2Enabled {
		return nil, ErrImages2Disabled
	}

	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	if user.Balance < settings.Images2PricePerImage {
		return nil, ErrImages2InsufficientFunds
	}

	group, err := s.resolveGroupByName(ctx, settings.Images2TargetGroupName)
	if err != nil {
		return nil, err
	}

	apiKey, err := s.findOrCreateKey(ctx, userID, group.ID)
	if err != nil {
		return nil, err
	}

	updatedUser, err := s.users.GetByID(ctx, userID)
	if err != nil {
		updatedUser = user
	}

	var subscription *UserSubscription
	if group.IsSubscriptionType() && s.apiKeys.userSubRepo != nil {
		subscription, _ = s.apiKeys.userSubRepo.GetActiveByUserIDAndGroupID(ctx, userID, group.ID)
	}
	apiKey.User = updatedUser
	apiKey.Group = group

	return &Images2PreparedRequest{
		Settings:     settings,
		User:         updatedUser,
		Group:        group,
		APIKey:       apiKey,
		Subscription: subscription,
		Prompt:       strings.TrimSpace(req.Prompt),
		ModelName:    settings.Images2ModelName,
		ImageURL:     strings.TrimSpace(req.ImageURL),
		Size:         size,
	}, nil
}

func normalizeImages2Size(size string) (string, error) {
	normalized := strings.TrimSpace(size)
	if normalized == "" {
		return "1024x1024", nil
	}
	switch normalized {
	case "1024x1024", "1536x1024", "1024x1536":
		return normalized, nil
	default:
		return "", infraerrors.BadRequest("IMAGES2_SIZE_INVALID", "size must be one of 1024x1024, 1536x1024, 1024x1536")
	}
}

func (s *Images2Service) resolveGroupByName(ctx context.Context, name string) (*Group, error) {
	groups, err := s.groups.ListActive(ctx)
	if err != nil {
		return nil, err
	}
	needle := strings.TrimSpace(name)
	for i := range groups {
		if strings.EqualFold(strings.TrimSpace(groups[i].Name), needle) {
			return &groups[i], nil
		}
	}
	return nil, ErrImages2GroupNotConfigured
}

func (s *Images2Service) findOrCreateKey(ctx context.Context, userID, groupID int64) (*APIKey, error) {
	filters := APIKeyListFilters{GroupID: &groupID}
	keys, _, err := s.apiKeys.List(ctx, userID, pagination.PaginationParams{Page: 1, PageSize: 100, SortBy: "id", SortOrder: "desc"}, filters)
	if err == nil {
		for i := range keys {
			if keys[i].Status == StatusActive && strings.TrimSpace(keys[i].Key) != "" {
				return &keys[i], nil
			}
		}
	}
	name := "ChatGPT Images 2 Auto Key"
	return s.apiKeys.Create(ctx, userID, CreateAPIKeyRequest{Name: name, GroupID: &groupID})
}
