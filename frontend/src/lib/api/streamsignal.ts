import {
    ClearPendingLiveNowSession,
    DeleteDestination,
    DeleteDestinationGroup,
    DuplicateDestinationGroup,
    EndStream,
    ForceGoLive,
    GeneratePreview,
    GetDestinationGroup,
    GetDestinationGroupIDs,
    GetDiagnostics,
    GetGroupDestinations,
    GetLogs,
    GoLive,
    ListDestinationGroups,
    GetSettings,
    ListPendingLiveNowSessions,
    ListDestinations,
    SaveDestination,
    SaveDestinationGroup,
    SaveSettings,
    SetDestinationGroups,
    SetGroupDestinations,
    TestDestinationConnection,
} from '../../../wailsjs/go/main/App';
import { domain } from '../../../wailsjs/go/models';
import type { CredentialCheckResult } from '../../types/credential-check';
import type { DestinationGroupInput } from '../../types/destination-group';
import type { DestinationInput, DestinationPlatform } from '../../types/destination';
import type { ExecutionResult, ExecutionSummary } from '../../types/execution';
import type { ActiveLiveNowSession, LiveNowRecoveryResult } from '../../types/live-now-recovery';
import type { LogEntry } from '../../types/log-entry';
import type { AnnouncementInput, PreviewItem } from '../../types/preview';
import type { AppSettings } from '../../types/settings';

function toAnnouncementModel(announcement: AnnouncementInput, destinationGroupID = '', destinationIDs: string[] = []): domain.Announcement {
    return {
        ...domain.Announcement.createFrom(announcement),
        destinationGroupID,
        destinationIDs,
    } as unknown as domain.Announcement;
}

function asArray<T>(value: T[] | null | undefined): T[] {
    return Array.isArray(value) ? value : [];
}

function toDestinationModel(destination: DestinationInput): domain.Destination {
    const { createdAt, updatedAt, ...rest } = destination;
    return domain.Destination.createFrom({
        ...rest,
        ...(createdAt ? { createdAt } : {}),
        ...(updatedAt ? { updatedAt } : {}),
    });
}

export function errorMessage(err: unknown, fallback: string): string {
    if (err instanceof Error) {
        return err.message;
    }
    if (typeof err === 'string') {
        return err;
    }
    return fallback;
}

function toExecutionResult(result: domain.ExecutionResult): ExecutionResult {
    return {
        destinationID: result.destinationID,
        destinationName: result.destinationName,
        platform: result.platform,
        state: result.state as ExecutionResult['state'],
        message: result.message,
        content: result.content,
        receipt: result.receipt
            ? {
                  provider: result.receipt.provider,
                  id: result.receipt.id,
                  uri: result.receipt.uri,
                  cid: result.receipt.cid,
              }
            : undefined,
    };
}

function toExecutionSummary(summary: domain.ExecutionSummary): ExecutionSummary {
    return {
        mode: summary.mode as ExecutionSummary['mode'],
        status: summary.status as ExecutionSummary['status'],
        testModeActive: summary.testModeActive,
        results: asArray(summary.results).map(toExecutionResult),
        totalCount: summary.totalCount,
        successCount: summary.successCount,
        failedCount: summary.failedCount,
        skippedCount: summary.skippedCount,
        validationErrorCount: summary.validationErrorCount,
        requiresDuplicateConfirmation: summary.requiresDuplicateConfirmation,
        duplicateWarningMessage: summary.duplicateWarningMessage,
    };
}

function toDestinationInput(destination: domain.Destination): DestinationInput {
    return {
        id: destination.id,
        platform: destination.platform as DestinationPlatform,
        name: destination.name,
        enabled: destination.enabled,
        template: destination.template,
        configJSON: destination.configJSON,
        createdAt: destination.createdAt ? String(destination.createdAt) : '',
        updatedAt: destination.updatedAt ? String(destination.updatedAt) : '',
    };
}

function toDestinationGroupModel(group: DestinationGroupInput): domain.DestinationGroup {
    const { createdAt, updatedAt, ...rest } = group;
    return domain.DestinationGroup.createFrom({
        ...rest,
        ...(createdAt ? { createdAt } : {}),
        ...(updatedAt ? { updatedAt } : {}),
    });
}

function toDestinationGroupInput(group: domain.DestinationGroup): DestinationGroupInput {
    return {
        id: group.id,
        name: group.name,
        description: group.description,
        createdAt: group.createdAt ? String(group.createdAt) : '',
        updatedAt: group.updatedAt ? String(group.updatedAt) : '',
    };
}

function toCredentialCheckResult(result: domain.CredentialCheckResult): CredentialCheckResult {
    return {
        platform: result.platform as CredentialCheckResult['platform'],
        state: result.state as CredentialCheckResult['state'],
        message: result.message,
    };
}

function toActiveLiveNowSession(session: domain.ActiveLiveNowSession): ActiveLiveNowSession {
    return {
        destinationID: session.destinationID,
        destinationName: session.destinationName,
        platform: session.platform,
        accountIdentifier: session.accountIdentifier,
        credentialKey: session.credentialKey,
        streamURL: session.streamURL,
        streamTitle: session.streamTitle,
        startedAt: session.startedAt ? String(session.startedAt) : '',
    };
}

function toLogEntry(entry: domain.LogEntry): LogEntry {
    return {
        timestamp: entry.timestamp ? String(entry.timestamp) : '',
        destination: entry.destination,
        action: entry.action,
        status: entry.status,
        message: entry.message,
    };
}

function toPreviewItem(item: domain.PreviewItem): PreviewItem {
    return {
        destinationID: item.destinationID,
        destinationName: item.destinationName,
        platform: item.platform,
        previewLabel: item.previewLabel,
        content: item.content,
        characterCount: item.characterCount,
        validationState: item.validationState,
        validationNotes: asArray(item.validationNotes),
        linkPreview: item.linkPreview
            ? {
                  url: item.linkPreview.url,
                  title: item.linkPreview.title,
                  description: item.linkPreview.description,
                  imageURL: item.linkPreview.imageURL,
                  siteName: item.linkPreview.siteName,
              }
            : null,
    };
}

function toSettingsModel(settings: AppSettings): domain.AppSettings {
    return domain.AppSettings.createFrom(settings);
}

export function getLogs(): Promise<LogEntry[]> {
    return GetLogs().then((entries: domain.LogEntry[] | null) => asArray(entries).map(toLogEntry));
}

export function getDiagnostics(): Promise<string> {
    return GetDiagnostics();
}

export function generatePreview(announcement: AnnouncementInput, destinationGroupID = '', destinationIDs: string[] = []): Promise<PreviewItem[]> {
    return GeneratePreview(toAnnouncementModel(announcement, destinationGroupID, destinationIDs)).then((items: domain.PreviewItem[] | null) => asArray(items).map(toPreviewItem));
}

export function goLive(announcement: AnnouncementInput, destinationGroupID = '', destinationIDs: string[] = []): Promise<ExecutionSummary> {
    return GoLive(toAnnouncementModel(announcement, destinationGroupID, destinationIDs)).then(toExecutionSummary);
}

export function forceGoLive(announcement: AnnouncementInput, destinationGroupID = '', destinationIDs: string[] = []): Promise<ExecutionSummary> {
    return ForceGoLive(toAnnouncementModel(announcement, destinationGroupID, destinationIDs)).then(toExecutionSummary);
}

export function endStream(announcement: AnnouncementInput, destinationGroupID = '', destinationIDs: string[] = []): Promise<ExecutionSummary> {
    return EndStream(toAnnouncementModel(announcement, destinationGroupID, destinationIDs)).then(toExecutionSummary);
}

export function listPendingLiveNowSessions(): Promise<ActiveLiveNowSession[]> {
    return ListPendingLiveNowSessions().then((sessions: domain.ActiveLiveNowSession[] | null) => asArray(sessions).map(toActiveLiveNowSession));
}

export function clearPendingLiveNowSession(destinationID: string): Promise<LiveNowRecoveryResult> {
    return ClearPendingLiveNowSession(destinationID).then(toExecutionResult);
}

export function listDestinations(): Promise<DestinationInput[]> {
    return ListDestinations().then((destinations: domain.Destination[] | null) => asArray(destinations).map(toDestinationInput));
}

export function saveDestination(destination: DestinationInput): Promise<DestinationInput> {
    return SaveDestination(toDestinationModel(destination)).then(toDestinationInput);
}

export function testDestinationConnection(destination: DestinationInput): Promise<CredentialCheckResult> {
    return TestDestinationConnection(toDestinationModel(destination)).then(toCredentialCheckResult);
}

export function deleteDestination(id: string): Promise<void> {
    return DeleteDestination(id);
}

export function listDestinationGroups(): Promise<DestinationGroupInput[]> {
    return ListDestinationGroups().then((groups: domain.DestinationGroup[] | null) => asArray(groups).map(toDestinationGroupInput));
}

export function getDestinationGroup(id: string): Promise<DestinationGroupInput> {
    return GetDestinationGroup(id).then(toDestinationGroupInput);
}

export function saveDestinationGroup(group: DestinationGroupInput): Promise<DestinationGroupInput> {
    return SaveDestinationGroup(toDestinationGroupModel(group)).then(toDestinationGroupInput);
}

export function duplicateDestinationGroup(id: string): Promise<DestinationGroupInput> {
    return DuplicateDestinationGroup(id).then(toDestinationGroupInput);
}

export function deleteDestinationGroup(id: string): Promise<void> {
    return DeleteDestinationGroup(id);
}

export function getGroupDestinations(groupID: string): Promise<DestinationInput[]> {
    return GetGroupDestinations(groupID).then((destinations: domain.Destination[] | null) => asArray(destinations).map(toDestinationInput));
}

export function setGroupDestinations(groupID: string, destinationIDs: string[]): Promise<void> {
    return SetGroupDestinations(groupID, destinationIDs);
}

export function getDestinationGroupIDs(destinationID: string): Promise<string[]> {
    return GetDestinationGroupIDs(destinationID).then((groupIDs: string[] | null) => asArray(groupIDs));
}

export function setDestinationGroups(destinationID: string, groupIDs: string[]): Promise<void> {
    return SetDestinationGroups(destinationID, groupIDs);
}

export function getSettings(): Promise<AppSettings> {
    return GetSettings().then((settings: domain.AppSettings) => ({
        defaultStreamURL: settings.defaultStreamURL,
        defaultHashtags: settings.defaultHashtags,
        duplicateProtectionEnabled: settings.duplicateProtectionEnabled,
        duplicateWindowMinutes: settings.duplicateWindowMinutes,
        selectedDestinationGroupID: settings.selectedDestinationGroupID,
    }));
}

export function saveSettings(settings: AppSettings): Promise<AppSettings> {
    return SaveSettings(toSettingsModel(settings)).then((saved: domain.AppSettings) => ({
        defaultStreamURL: saved.defaultStreamURL,
        defaultHashtags: saved.defaultHashtags,
        duplicateProtectionEnabled: saved.duplicateProtectionEnabled,
        duplicateWindowMinutes: saved.duplicateWindowMinutes,
        selectedDestinationGroupID: saved.selectedDestinationGroupID,
    }));
}
