# AGENTS.md

## Settings Field Checklist

When adding or renaming any system setting field, do not stop after updating just one API.

For every new settings field, verify the full propagation path end-to-end:

1. Backend setting key/constants
   - `backend/internal/service/domain_constants.go`

2. Backend service models and parsing
   - `backend/internal/service/settings_view.go`
   - `backend/internal/service/setting_service.go`
   - Add default value, parsing, clamping, and persistence updates as needed.

3. Public settings response chain
   - `backend/internal/service/setting_service.go`
     - `PublicSettings`
     - `PublicSettingsInjectionPayload`
     - `GetPublicSettings`
     - `GetPublicSettingsForInjection`
   - `backend/internal/handler/dto/settings.go`
     - `dto.PublicSettings`
   - `backend/internal/handler/setting_handler.go`
     - `GetPublicSettings`

4. Admin settings response/update chain
   - `backend/internal/handler/dto/settings.go`
     - `dto.SystemSettings`
   - `backend/internal/handler/admin/setting_handler.go`
     - admin GET payload
     - update request struct
     - merge/update logic
     - response payload after update
     - diff/audit list

5. Frontend settings consumers
   - `frontend/src/types/index.ts`
   - `frontend/src/api/admin/settings.ts`
   - `frontend/src/stores/app.ts`
   - Any user/admin view that reads the field

6. Tests that must be updated or added
   - `backend/internal/service/setting_service_public_test.go`
   - `backend/internal/handler/setting_handler_public_test.go`
   - `backend/internal/handler/dto/public_settings_injection_schema_test.go`
   - Any admin settings schema drift test

## Failure Pattern To Avoid

This repo has repeatedly hit the same regression: a new field is added to admin settings and service parsing, but one of these locations is missed:

- public handler response mapping
- SSR public settings injection payload
- frontend public settings type/defaults
- admin update/response payload

If admin `/api/v1/admin/settings` and public `/api/v1/settings/public` disagree for the same field, treat it as an incomplete implementation, not a follow-up cleanup.

## Definition Of Done For Settings Fields

A settings-field change is not done until:

1. Admin GET returns the field
2. Admin update persists the field
3. Public GET returns the field
4. SSR injection exposes the field when applicable
5. Frontend types/defaults are updated
6. Drift tests cover the field
