package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

var (
	ErrPlaygroundGroupRequired = infraerrors.BadRequest("PLAYGROUND_GROUP_REQUIRED", "group_id is required")
	ErrPlaygroundGroupInvalid  = infraerrors.BadRequest("PLAYGROUND_GROUP_INVALID", "invalid group_id")
	ErrPlaygroundGroupDenied   = infraerrors.Forbidden("PLAYGROUND_GROUP_DENIED", "user cannot access this group")
)

const playgroundRuntimeAPIKeyPrefix = "pg-user-"

type PlaygroundGroup struct {
	ID                  int64    `json:"id"`
	Name                string   `json:"name"`
	Description         string   `json:"description"`
	Platform            string   `json:"platform"`
	RateMultiplier      float64  `json:"rate_multiplier"`
	EffectiveMultiplier float64  `json:"effective_multiplier"`
	SubscriptionType    string   `json:"subscription_type"`
	SupportedScopes     []string `json:"supported_model_scopes,omitempty"`
}

type PlaygroundExecutionContext struct {
	User         *User
	Group        *Group
	RuntimeAPIKey *APIKey
	Subscription *UserSubscription
}

type PlaygroundService struct {
	userRepo            UserRepository
	groupRepo           GroupRepository
	apiKeyService       *APIKeyService
	subscriptionService *SubscriptionService
	capabilityService   *AccountModelCapabilityService
	gatewayService      *GatewayService
}

func NewPlaygroundService(
	userRepo UserRepository,
	groupRepo GroupRepository,
	apiKeyService *APIKeyService,
	subscriptionService *SubscriptionService,
	capabilityService *AccountModelCapabilityService,
	gatewayService *GatewayService,
) *PlaygroundService {
	return &PlaygroundService{
		userRepo:            userRepo,
		groupRepo:           groupRepo,
		apiKeyService:       apiKeyService,
		subscriptionService: subscriptionService,
		capabilityService:   capabilityService,
		gatewayService:      gatewayService,
	}
}

func (s *PlaygroundService) ListGroups(ctx context.Context, userID int64) ([]PlaygroundGroup, error) {
	groups, err := s.apiKeyService.GetAvailableGroups(ctx, userID)
	if err != nil {
		return nil, err
	}
	rates, err := s.apiKeyService.GetUserGroupRates(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]PlaygroundGroup, 0, len(groups))
	for i := range groups {
		group := groups[i]
		effectiveMultiplier := group.RateMultiplier
		if rate, ok := rates[group.ID]; ok {
			effectiveMultiplier = rate
		}
		out = append(out, PlaygroundGroup{
			ID:                  group.ID,
			Name:                group.Name,
			Description:         group.Description,
			Platform:            group.Platform,
			RateMultiplier:      group.RateMultiplier,
			EffectiveMultiplier: effectiveMultiplier,
			SubscriptionType:    group.SubscriptionType,
			SupportedScopes:     append([]string(nil), group.SupportedModelScopes...),
		})
	}
	return out, nil
}

func (s *PlaygroundService) ResolveExecutionContext(ctx context.Context, userID int64, groupID int64) (*PlaygroundExecutionContext, error) {
	if groupID <= 0 {
		return nil, ErrPlaygroundGroupInvalid
	}
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	group, err := s.groupRepo.GetByID(ctx, groupID)
	if err != nil {
		return nil, fmt.Errorf("get group: %w", err)
	}
	if group == nil || !group.IsActive() {
		return nil, ErrGroupNotFound
	}
	availableGroups, err := s.apiKeyService.GetAvailableGroups(ctx, userID)
	if err != nil {
		return nil, err
	}
	allowed := false
	for i := range availableGroups {
		if availableGroups[i].ID == groupID {
			allowed = true
			break
		}
	}
	if !allowed {
		return nil, ErrPlaygroundGroupDenied
	}
	var subscription *UserSubscription
	if group.IsSubscriptionType() && s.subscriptionService != nil {
		subscription, err = s.subscriptionService.GetActiveSubscription(ctx, userID, groupID)
		if err != nil {
			return nil, err
		}
	}
	runtimeAPIKey := buildPlaygroundRuntimeAPIKey(user, group)
	return &PlaygroundExecutionContext{
		User:          user,
		Group:         group,
		RuntimeAPIKey: runtimeAPIKey,
		Subscription:  subscription,
	}, nil
	}

func (s *PlaygroundService) ListModels(ctx context.Context, userID int64, groupID *int64, search string, limit int) ([]GroupAvailableModel, error) {
	availableGroups, err := s.apiKeyService.GetAvailableGroups(ctx, userID)
	if err != nil {
		return nil, err
	}
	groupIDs := make([]int64, 0, len(availableGroups))
	allowed := map[int64]struct{}{}
	for i := range availableGroups {
		groupIDs = append(groupIDs, availableGroups[i].ID)
		allowed[availableGroups[i].ID] = struct{}{}
	}
	if groupID != nil {
		if _, ok := allowed[*groupID]; !ok {
			return nil, ErrPlaygroundGroupDenied
		}
		groupIDs = []int64{*groupID}
	}
	if limit <= 0 {
		limit = 100
	}
	if s.capabilityService == nil {
		return s.listModelsFallback(ctx, groupIDs), nil
	}
	items, err := s.capabilityService.ListGroupModels(ctx, groupIDs, strings.TrimSpace(search), limit)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return s.listModelsFallback(ctx, groupIDs), nil
	}
	return items, nil
}

func (s *PlaygroundService) listModelsFallback(ctx context.Context, groupIDs []int64) []GroupAvailableModel {
	if s.gatewayService == nil || len(groupIDs) == 0 {
		return []GroupAvailableModel{}
	}
	seen := make(map[string]struct{})
	out := make([]GroupAvailableModel, 0)
	for _, groupID := range groupIDs {
		groupCopy := groupID
		models := s.gatewayService.GetAvailableModels(ctx, &groupCopy, "")
		for _, model := range models {
			model = strings.TrimSpace(model)
			if model == "" {
				continue
			}
			if _, exists := seen[model]; exists {
				continue
			}
			seen[model] = struct{}{}
			out = append(out, GroupAvailableModel{ModelName: model})
		}
	}
	return out
}

func buildPlaygroundRuntimeAPIKey(user *User, group *Group) *APIKey {
	now := time.Now()
	groupID := group.ID
	apiKey := &APIKey{
		ID:        -user.ID*1000 - group.ID,
		UserID:    user.ID,
		Key:       playgroundRuntimeAPIKeyPrefix + fmt.Sprintf("%d-%d", user.ID, group.ID),
		Name:      "Playground Runtime",
		GroupID:   &groupID,
		Status:    StatusAPIKeyActive,
		CreatedAt: now,
		UpdatedAt: now,
		User:      user,
		Group:     group,
	}
	return apiKey
}
