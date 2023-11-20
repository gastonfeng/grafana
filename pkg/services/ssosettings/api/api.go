package api

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/grafana/grafana/pkg/api/response"
	"github.com/grafana/grafana/pkg/api/routing"
	"github.com/grafana/grafana/pkg/infra/log"
	ac "github.com/grafana/grafana/pkg/services/accesscontrol"
	contextmodel "github.com/grafana/grafana/pkg/services/contexthandler/model"
	"github.com/grafana/grafana/pkg/services/featuremgmt"
	"github.com/grafana/grafana/pkg/services/ssosettings"
	"github.com/grafana/grafana/pkg/services/ssosettings/models"
	"github.com/grafana/grafana/pkg/web"
)

type Api struct {
	Log                log.Logger
	RouteRegister      routing.RouteRegister
	AccessControl      ac.AccessControl
	Features           *featuremgmt.FeatureManager
	SSOSettingsService ssosettings.Service
}

func ProvideApi(
	ssoSettingsSvc ssosettings.Service,
	routeRegister routing.RouteRegister,
	ac ac.AccessControl,
) *Api {
	api := &Api{
		SSOSettingsService: ssoSettingsSvc,
		RouteRegister:      routeRegister,
		AccessControl:      ac,
		Log:                log.New("ssosettings.api"),
	}

	return api
}

// RegisterAPIEndpoints Registers Endpoints on Grafana Router
func (api *Api) RegisterAPIEndpoints() {
	api.RouteRegister.Group("/api/v1/sso-settings", func(router routing.RouteRegister) {
		auth := ac.Middleware(api.AccessControl)

		scopeKey := ac.Parameter(":key")
		settingsScope := ac.Scope("settings", "auth."+scopeKey, "*")

		reqWriteAccess := auth(ac.EvalAny(
			ac.EvalPermission(ac.ActionSettingsWrite, ac.ScopeSettingsAuth),
			ac.EvalPermission(ac.ActionSettingsWrite, settingsScope)))

		router.Get("/", auth(ac.EvalPermission(ac.ActionSettingsRead, ac.ScopeSettingsAuth)), routing.Wrap(api.listAllProvidersSettings))
		router.Get("/:key", auth(ac.EvalPermission(ac.ActionSettingsRead, settingsScope)), routing.Wrap(api.getProviderSettings))
		router.Put("/:key", reqWriteAccess, routing.Wrap(api.updateProviderSettings))
		router.Delete("/:key", reqWriteAccess, routing.Wrap(api.removeProviderSettings))
	})
}

func (api *Api) listAllProvidersSettings(c *contextmodel.ReqContext) response.Response {
	providers, err := api.SSOSettingsService.List(c.Req.Context(), c.SignedInUser)
	if err != nil {
		return response.Error(500, "Failed to get providers", err)
	}

	return response.JSON(200, providers)
}

func (api *Api) getProviderSettings(c *contextmodel.ReqContext) response.Response {
	key, ok := web.Params(c.Req)[":key"]
	if !ok {
		return response.Error(400, "Missing key", nil)
	}

	settings, err := api.SSOSettingsService.GetForProvider(c.Req.Context(), key)
	if err != nil {
		return response.Error(404, "The provider was not found", err)
	}

	if c.QueryBool("includeDefaults") {
		return response.JSON(200, settings)
	}

	// colinTODO: remove when defaults for each provider are implemented
	defaults := map[string]interface{}{
		"enabled":                    false,
		"role_attribute_strict":      false,
		"allow_sign_up":              true,
		"name":                       "default name",
		"tls_skip_verify_insecure":   false,
		"use_pkce":                   true,
		"use_refresh_token":          false,
		"allow_assign_grafana_admin": false,
		"auto_login":                 false,
	}

	for key, defaultValue := range defaults {
		if value, exists := settings.Settings[key]; exists && value == defaultValue {
			delete(settings.Settings, key)
		}
	}

	if _, exists := settings.Settings["client_secret"]; exists {
		settings.Settings["client_secret"] = "*********"
	}

	etag := generateSHA1ETag(settings.Settings)

	return response.JSON(200, settings).SetHeader("ETag", etag)
}

func generateSHA1ETag(settings map[string]interface{}) string {
	hasher := sha1.New()
	data, _ := json.Marshal(settings)
	hasher.Write(data)
	return fmt.Sprintf("%x", hasher.Sum(nil))
}

func (api *Api) updateProviderSettings(c *contextmodel.ReqContext) response.Response {
	key, ok := web.Params(c.Req)[":key"]
	if !ok {
		return response.Error(400, "Missing key", nil)
	}

	var newSettings models.SSOSetting
	if err := web.Bind(c.Req, &newSettings); err != nil {
		return response.Error(400, "Failed to parse request body", err)
	}

	err := api.SSOSettingsService.Upsert(c.Req.Context(), key, newSettings.Settings)
	// TODO: first check whether the error is referring to validation errors

	// other error
	if err != nil {
		return response.Error(500, "Failed to update provider settings", err)
	}

	return response.JSON(204, nil)
}

func (api *Api) removeProviderSettings(c *contextmodel.ReqContext) response.Response {
	key, ok := web.Params(c.Req)[":key"]
	if !ok {
		return response.Error(400, "Missing key", nil)
	}

	err := api.SSOSettingsService.Delete(c.Req.Context(), key)
	if err != nil {
		return response.Error(500, "Failed to delete provider settings", err)
	}

	return response.JSON(204, nil)
}
