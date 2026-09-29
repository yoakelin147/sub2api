package handler

import (
	"slices"

	"github.com/Wei-Shaw/sub2api/internal/pkg/antigravity"
	"github.com/Wei-Shaw/sub2api/internal/pkg/claude"
	"github.com/Wei-Shaw/sub2api/internal/pkg/geminicli"
	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/Wei-Shaw/sub2api/internal/pkg/xai"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type supplierAccountTestModel struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	DisplayName string `json:"display_name"`
	CreatedAt   string `json:"created_at"`
}

func supplierAccountTestModels(account *service.Account) []supplierAccountTestModel {
	models := make([]supplierAccountTestModel, 0)
	add := func(id, name, createdAt string) {
		if name == "" {
			name = id
		}
		models = append(models, supplierAccountTestModel{ID: id, Type: "model", DisplayName: name, CreatedAt: createdAt})
	}
	switch account.Platform {
	case service.PlatformOpenAI:
		for _, model := range openai.DefaultModels {
			add(model.ID, model.DisplayName, "")
		}
	case service.PlatformGemini:
		choices := geminicli.DefaultModels
		if account.IsOAuth() && account.IsGeminiGoogleOne() {
			choices = geminicli.GoogleOneModels
		}
		for _, model := range choices {
			add(model.ID, model.DisplayName, model.CreatedAt)
		}
	case service.PlatformAntigravity:
		for _, model := range antigravity.DefaultModels() {
			add(model.ID, model.DisplayName, model.CreatedAt)
		}
	case service.PlatformGrok:
		for _, model := range xai.DefaultModels() {
			add(model.ID, model.DisplayName, "")
		}
	case service.PlatformOpenCodeGo:
		add(service.DefaultOpenCodeGoTestModel, service.DefaultOpenCodeGoTestModel, "")
	default:
		for _, model := range claude.DefaultModels {
			add(model.ID, model.DisplayName, model.CreatedAt)
		}
	}
	if account.IsOAuth() && (account.IsGemini() || account.IsAnthropic()) {
		return models
	}
	var hasMapping bool
	switch mapping := account.Credentials["model_mapping"].(type) {
	case map[string]any:
		hasMapping = len(mapping) > 0
	case map[string]string:
		hasMapping = len(mapping) > 0
	}
	if !hasMapping || account.IsOpenAI() && account.IsOpenAIPassthroughEnabled() {
		return models
	}
	byID := make(map[string]supplierAccountTestModel, len(models))
	for _, model := range models {
		byID[model.ID] = model
	}
	ids := make([]string, 0)
	for id := range account.GetModelMapping() {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	selected := make([]supplierAccountTestModel, 0, len(ids))
	for _, id := range ids {
		if model, ok := byID[id]; ok {
			selected = append(selected, model)
		} else {
			selected = append(selected, supplierAccountTestModel{ID: id, Type: "model", DisplayName: id})
		}
	}
	return selected
}
