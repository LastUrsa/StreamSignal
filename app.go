package main

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"

	appsvc "StreamSignal/internal/app"
	"StreamSignal/internal/domain"
	"StreamSignal/internal/platforms/bluesky"
	"StreamSignal/internal/platforms/discord"
	"StreamSignal/internal/platforms/mastodon"
	"StreamSignal/internal/ports"
	"StreamSignal/internal/secrets/wincred"
	"StreamSignal/internal/storage/secure"
	"StreamSignal/internal/storage/sqlite"
)

// App exposes Wails bindings for the frontend.
type App struct {
	ctx             context.Context
	db              *sql.DB
	settingsService *appsvc.SettingsService
	destinationSvc  *appsvc.DestinationService
	groupSvc        *appsvc.DestinationGroupService
	credentialSvc   *appsvc.CredentialSetupService
	previewService  *appsvc.PreviewService
	logService      *appsvc.LogService
	diagnosticsSvc  *appsvc.DiagnosticsService
	executionSvc    *appsvc.ExecutionService
	recoverySvc     *appsvc.LiveNowRecoveryService
}

// NewApp creates a new application shell.
func NewApp() *App {
	dataDir, err := appsvc.EnsureDataDir()
	if err != nil {
		panic(err)
	}

	return newAppWithDependencies(filepath.Join(dataDir, "streamsignal.db"), wincred.NewStore())
}

func newAppWithDatabasePath(databasePath string) *App {
	return newAppWithDependencies(databasePath, wincred.NewStore())
}

func newAppWithDependencies(databasePath string, secretStore ports.SecretStore) *App {
	db, err := sqlite.Open(databasePath)
	if err != nil {
		panic(err)
	}

	settingsRepo := secure.NewSettingsRepository(sqlite.NewSettingsRepository(db), secretStore)
	destinationRepo := secure.NewDestinationRepository(sqlite.NewDestinationRepository(db), secretStore)
	groupRepo := sqlite.NewDestinationGroupRepository(db)
	logRepo := sqlite.NewLogRepository(db)
	historyRepo := sqlite.NewPostHistoryRepository(db)
	liveNowSessionRepo := secure.NewLiveNowSessionRepository(sqlite.NewLiveNowSessionRepository(db), secretStore)
	discordPublisher := discord.NewPublisher(nil)
	blueskyPublisher := bluesky.NewPublisher("", nil)
	mastodonPublisher := mastodon.NewPublisher(nil)

	return &App{
		db:              db,
		settingsService: appsvc.NewSettingsService(settingsRepo),
		destinationSvc:  appsvc.NewDestinationService(destinationRepo),
		groupSvc:        appsvc.NewDestinationGroupService(groupRepo),
		credentialSvc:   appsvc.NewCredentialSetupService(discordPublisher, blueskyPublisher, mastodonPublisher),
		previewService: func() *appsvc.PreviewService {
			service := appsvc.NewPreviewService(destinationRepo, settingsRepo, liveNowSessionRepo)
			service.UseDestinationGroups(groupRepo)
			return service
		}(),
		logService:     appsvc.NewLogService(logRepo),
		diagnosticsSvc: appsvc.NewDiagnosticsService(settingsRepo, destinationRepo, logRepo, liveNowSessionRepo),
		executionSvc: func() *appsvc.ExecutionService {
			service := appsvc.NewExecutionService(
				destinationRepo,
				settingsRepo,
				historyRepo,
				liveNowSessionRepo,
				discordPublisher,
				blueskyPublisher,
				mastodonPublisher,
			)
			service.UseDestinationGroups(groupRepo)
			return service
		}(),
		recoverySvc: appsvc.NewLiveNowRecoveryService(destinationRepo, liveNowSessionRepo, blueskyPublisher),
	}
}

// startup is called when the app starts.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// shutdown closes open resources when the app exits.
func (a *App) shutdown(context.Context) {
	if a.db != nil {
		_ = a.db.Close()
	}
}

func (a *App) GetSettings() (domain.AppSettings, error) {
	return a.settingsService.Load(a.ctx)
}

func (a *App) SaveSettings(settings domain.AppSettings) (domain.AppSettings, error) {
	saved, err := a.settingsService.Save(a.ctx, settings)
	if err == nil {
		_ = a.logService.Append(a.ctx, "settings", "save_settings", "SUCCESS", "Saved application settings.")
	}
	return saved, err
}

func (a *App) ListDestinations() ([]domain.Destination, error) {
	return a.destinationSvc.List(a.ctx)
}

func (a *App) SaveDestination(destination domain.Destination) (domain.Destination, error) {
	saved, err := a.destinationSvc.Save(a.ctx, destination)
	if err == nil {
		_ = a.groupSvc.AddDestination(a.ctx, domain.DefaultDestinationGroupID, saved.ID)
		_ = a.logService.Append(a.ctx, saved.Name, "save_destination", "SUCCESS", "Saved destination configuration.")
	}
	return saved, err
}

func (a *App) DeleteDestination(id string) error {
	err := a.destinationSvc.Delete(a.ctx, id)
	if err == nil {
		_ = a.logService.Append(a.ctx, id, "delete_destination", "SUCCESS", "Deleted destination configuration.")
	}
	return err
}

func (a *App) ListDestinationGroups() ([]domain.DestinationGroup, error) {
	return a.groupSvc.List(a.ctx)
}

func (a *App) GetDestinationGroup(id string) (domain.DestinationGroup, error) {
	return a.groupSvc.Get(a.ctx, id)
}

func (a *App) SaveDestinationGroup(group domain.DestinationGroup) (domain.DestinationGroup, error) {
	saved, err := a.groupSvc.Save(a.ctx, group)
	if err == nil {
		_ = a.logService.Append(a.ctx, saved.Name, "save_destination_group", "SUCCESS", "Saved destination group.")
	}
	return saved, err
}

func (a *App) DuplicateDestinationGroup(id string) (domain.DestinationGroup, error) {
	saved, err := a.groupSvc.Duplicate(a.ctx, id)
	if err == nil {
		_ = a.logService.Append(a.ctx, saved.Name, "duplicate_destination_group", "SUCCESS", "Duplicated destination group.")
	}
	return saved, err
}

func (a *App) DeleteDestinationGroup(id string) error {
	err := a.groupSvc.Delete(a.ctx, id)
	if err == nil {
		_ = a.logService.Append(a.ctx, id, "delete_destination_group", "SUCCESS", "Deleted destination group.")
	}
	return err
}

func (a *App) GetGroupDestinations(groupID string) ([]domain.Destination, error) {
	return a.groupSvc.GetDestinations(a.ctx, groupID)
}

func (a *App) SetGroupDestinations(groupID string, destinationIDs []string) error {
	return a.groupSvc.SetDestinations(a.ctx, groupID, destinationIDs)
}

func (a *App) GetDestinationGroupIDs(destinationID string) ([]string, error) {
	return a.groupSvc.GetGroupIDsForDestination(a.ctx, destinationID)
}

func (a *App) SetDestinationGroups(destinationID string, groupIDs []string) error {
	return a.groupSvc.SetGroupsForDestination(a.ctx, destinationID, groupIDs)
}

func (a *App) AddDestinationToGroup(groupID string, destinationID string) error {
	return a.groupSvc.AddDestination(a.ctx, groupID, destinationID)
}

func (a *App) RemoveDestinationFromGroup(groupID string, destinationID string) error {
	return a.groupSvc.RemoveDestination(a.ctx, groupID, destinationID)
}

func (a *App) TestDestinationConnection(destination domain.Destination) domain.CredentialCheckResult {
	result := a.credentialSvc.TestDestination(a.ctx, destination)
	status := result.State
	if status == "" {
		status = "FAILED"
	}
	_ = a.logService.Append(a.ctx, destination.Name, "test_destination_connection", status, result.Message)
	return result
}

func (a *App) GeneratePreview(announcement domain.Announcement) ([]domain.PreviewItem, error) {
	items, err := a.previewService.Generate(a.ctx, announcement)
	if err == nil {
		_ = a.logService.Append(a.ctx, "preview", "generate_preview", "SUCCESS", fmt.Sprintf("Generated %d preview items.", len(items)))
	}
	return items, err
}

func (a *App) GetLogs() ([]domain.LogEntry, error) {
	return a.logService.ListRecent(a.ctx, 100)
}

func (a *App) GetDiagnostics() (string, error) {
	return a.diagnosticsSvc.Build(a.ctx)
}

func (a *App) GoLive(announcement domain.Announcement) (domain.ExecutionSummary, error) {
	summary, err := a.executionSvc.GoLive(a.ctx, announcement)
	if err == nil {
		a.appendExecutionLogs("Go Live", "go_live", "go_live_result", summary)
	}
	return summary, err
}

func (a *App) ForceGoLive(announcement domain.Announcement) (domain.ExecutionSummary, error) {
	summary, err := a.executionSvc.ForceGoLive(a.ctx, announcement)
	if err == nil {
		a.appendExecutionLogs("Confirmed Go Live", "force_go_live", "force_go_live_result", summary)
	}
	return summary, err
}

func (a *App) EndStream(announcement domain.Announcement) (domain.ExecutionSummary, error) {
	summary, err := a.executionSvc.EndStream(a.ctx, announcement)
	if err == nil {
		a.appendExecutionLogs("End Stream", "end_stream", "end_stream_result", summary)
	}
	return summary, err
}

func (a *App) appendExecutionLogs(actionLabel, summaryAction, resultAction string, summary domain.ExecutionSummary) {
	_ = a.logService.Append(a.ctx, summaryAction, summaryAction, executionLogStatus(summary), executionLogMessage(actionLabel, summary))
	for _, result := range summary.Results {
		destination := result.DestinationName
		if destination == "" {
			destination = result.DestinationID
		}
		_ = a.logService.Append(a.ctx, destination, resultAction, string(result.State), executionResultLogMessage(result))
	}
}

func (a *App) ListPendingLiveNowSessions() ([]domain.ActiveLiveNowSession, error) {
	return a.recoverySvc.ListPending(a.ctx)
}

func (a *App) ClearPendingLiveNowSession(destinationID string) (domain.ExecutionResult, error) {
	result, err := a.recoverySvc.ClearPending(a.ctx, destinationID)
	if err != nil {
		_ = a.logService.Append(
			a.ctx,
			destinationID,
			"clear_pending_live_now",
			"FAILED",
			fmt.Sprintf("Live Now recovery failed: %s", err.Error()),
		)
		return result, err
	}
	destinationName := result.DestinationName
	if destinationName == "" {
		destinationName = destinationID
	}
	_ = a.logService.Append(a.ctx, destinationName, "clear_pending_live_now", string(result.State), result.Message)
	return result, err
}

func executionLogStatus(summary domain.ExecutionSummary) string {
	if summary.Status == "" {
		return string(domain.ExecutionSummaryStatusSuccess)
	}
	return string(summary.Status)
}

func executionLogMessage(action string, summary domain.ExecutionSummary) string {
	message := fmt.Sprintf(
		"%s returned %d results: %d success, %d failed, %d skipped, %d validation.",
		action,
		summary.TotalCount,
		summary.SuccessCount,
		summary.FailedCount,
		summary.SkippedCount,
		summary.ValidationErrorCount,
	)
	if summary.RequiresDuplicateConfirmation {
		return fmt.Sprintf("%s Duplicate confirmation required.", message)
	}
	return message
}

func executionResultLogMessage(result domain.ExecutionResult) string {
	platform := string(result.Platform)
	if platform == "" {
		platform = "unknown"
	}
	if result.Message == "" {
		return fmt.Sprintf("%s result for %s.", result.State, platform)
	}
	return fmt.Sprintf("%s result for %s: %s", result.State, platform, result.Message)
}
