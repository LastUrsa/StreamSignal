import { FormEvent, type ComponentType, type SVGProps, useEffect, useRef, useState } from 'react';
import {
    Activity,
    ClipboardList,
    Copy,
    ExternalLink,
    FilePlus,
    FileText,
    Home,
    Info,
    Layers,
    ListChecks,
    Play,
    Radio,
    RefreshCw,
    RotateCcw,
    Save,
    Send,
    Settings,
    ShieldAlert,
    Square,
    Trash2,
    TriangleAlert,
    Wrench,
    X,
    Zap,
} from 'lucide-react';
import './App.css';
import streamSignalIcon from './assets/images/streamsignal-icon.png';
import blueskyLogo from './assets/platforms/bluesky.svg';
import discordLogo from './assets/platforms/discord.svg';
import mastodonLogo from './assets/platforms/mastodon.svg';
import {
    clearPendingLiveNowSession,
    deleteAnnouncementProfile,
    deleteDestination,
    deleteDestinationGroup,
    duplicateAnnouncementProfile,
    duplicateDestinationGroup,
    endStream,
    errorMessage,
    forceGoLive,
    generatePreview,
    getDestination,
    getDestinationGroupIDs,
    getDiagnostics,
    getGroupDestinations,
    getLogs,
    goLive,
    listAnnouncementProfiles,
    listDestinationGroups,
    listPendingLiveNowSessions,
    getSettings,
    listDestinations,
    markAnnouncementProfileUsed,
    saveDestination,
    saveAnnouncementProfile,
    saveDestinationGroup,
    saveSettings,
    setDestinationGroups as saveDestinationGroupsForDestination,
    setGroupDestinations,
    testDestinationConnection,
} from './lib/api/streamsignal';
import {
    createEmptyAnnouncementProfileForm,
    toAnnouncementProfileFormState,
    toAnnouncementProfileInput,
    type AnnouncementProfileFormState,
    type AnnouncementProfileInput,
} from './types/announcement-profile';
import type { CredentialCheckResult } from './types/credential-check';
import {
    createEmptyDestinationForm,
    defaultTemplateForPlatform,
    TEMPLATE_VARIABLES,
    toDestinationFormState,
    toDestinationInput,
    type DestinationFormState,
    type DestinationInput,
} from './types/destination';
import {
    createEmptyDestinationGroupForm,
    DEFAULT_DESTINATION_GROUP_ID,
    DEFAULT_DESTINATION_GROUP_NAME,
    toDestinationGroupFormState,
    type DestinationGroupFormState,
    type DestinationGroupInput,
} from './types/destination-group';
import type { ExecutionSummary } from './types/execution';
import type { ActiveLiveNowSession } from './types/live-now-recovery';
import type { AnnouncementInput, PreviewItem } from './types/preview';
import type { LogEntry } from './types/log-entry';
import type { AppSettings } from './types/settings';

type TabKey = 'home' | 'destinations' | 'profiles' | 'settings' | 'logs';

const STORED_SECRET_TOKEN = '[stored securely]';
const SHOW_DEBUG_WORKFLOW_UI = import.meta.env.DEV || import.meta.env.MODE === 'debug';
const APP_VERSION = '0.4.0';
const TAB_KEYS: TabKey[] = ['home', 'destinations', 'profiles', 'settings', 'logs'];
const PRODUCTION_TAB_KEYS: TabKey[] = ['home', 'destinations', 'profiles', 'settings'];

const initialAnnouncement: AnnouncementInput = {
    streamTitle: '',
    streamURL: '',
    category: '',
    message: '',
    hashtags: '',
    announcementImageURL: '',
    announcementImageDataURL: '',
    blueskyCardThumbnailURL: '',
    blueskyCardThumbnailDataURL: '',
};

const initialDestination = createEmptyDestinationForm();

const defaultSettings: AppSettings = {
    defaultStreamURL: '',
    defaultHashtags: '',
    duplicateProtectionEnabled: true,
    duplicateWindowMinutes: 10,
    selectedDestinationGroupID: DEFAULT_DESTINATION_GROUP_ID,
};

type DestinationSecretCache = Pick<DestinationFormState, 'discordWebhookKey' | 'blueskyCredentialKey' | 'mastodonCredentialKey'>;

type GuidedSetupStep = {
    title: string;
    detail: string;
};

type WorkflowToast = {
    id: number;
    message: string;
    kind: 'success' | 'warning';
};

type LogCategoryFilter = 'all' | 'execution' | 'destinations' | 'settings' | 'preview';
type LogStatusFilter = 'all' | 'failed' | 'validation' | 'warning' | 'success';
type HomeTargetMode = 'destinations' | 'group';

type SidebarItem = {
    key: TabKey;
    label: string;
    icon: ComponentType<SVGProps<SVGSVGElement>>;
    badge?: number;
};

type StatusTone = 'success' | 'warning' | 'info' | 'error' | 'disabled' | 'offline' | 'active' | 'running' | 'idle';

const LOG_CATEGORY_FILTERS: Array<{ value: LogCategoryFilter; label: string }> = [
    { value: 'all', label: 'All' },
    { value: 'execution', label: 'Execution' },
    { value: 'destinations', label: 'Destinations' },
    { value: 'settings', label: 'Settings' },
    { value: 'preview', label: 'Preview' },
];

const LOG_STATUS_FILTERS: Array<{ value: LogStatusFilter; label: string }> = [
    { value: 'all', label: 'All' },
    { value: 'failed', label: 'Failed' },
    { value: 'validation', label: 'Validation' },
    { value: 'warning', label: 'Warning' },
    { value: 'success', label: 'Success' },
];

const LEGAL_NOTICE =
    'StreamSignal is an independent tool and is not affiliated with, endorsed by, or sponsored by Discord, Bluesky, Mastodon, or any Mastodon server operator. Platform names and logos are used only to identify supported integrations.';

// Keep platform logos unmodified and limited to integration-identification UI.
const PLATFORM_LOGOS: Record<string, string> = {
    discord: discordLogo,
    bluesky: blueskyLogo,
    mastodon: mastodonLogo,
};

function platformLabel(platform: string) {
    switch (platform.toLowerCase()) {
        case 'discord':
            return 'Discord';
        case 'bluesky':
            return 'Bluesky';
        case 'mastodon':
            return 'Mastodon';
        default:
            return platform || 'Destination';
    }
}

function PlatformIcon({ platform, className = '' }: { platform: string; className?: string }) {
    const logo = PLATFORM_LOGOS[platform.toLowerCase()];
    if (logo) {
        return <img src={logo} alt="" className={`platform-icon ${className}`} aria-hidden="true" />;
    }
    return <Radio className={`platform-icon platform-icon-fallback ${className}`} aria-hidden="true" />;
}

function statusTone(status: string | null | undefined): StatusTone {
    const normalized = String(status ?? '').toLowerCase();
    if (normalized === 'success' || normalized === 'valid') {
        return 'success';
    }
    if (normalized === 'running') {
        return 'running';
    }
    if (normalized === 'active') {
        return 'active';
    }
    if (normalized === 'connected') {
        return 'success';
    }
    if (normalized === 'warning' || normalized === 'partial' || normalized === 'skipped' || normalized === 'pending' || normalized === 'paused') {
        return 'warning';
    }
    if (normalized === 'validation' || normalized === 'validation_error' || normalized === 'invalid') {
        return 'info';
    }
    if (normalized === 'failed' || normalized === 'error') {
        return 'error';
    }
    if (normalized === 'disabled') {
        return 'disabled';
    }
    if (normalized === 'offline') {
        return 'offline';
    }
    return 'idle';
}

function StatusBadge({ status, label }: { status: string | null | undefined; label?: string }) {
    const tone = statusTone(status);
    return <span className={`status-badge status-${tone}`}>{label ?? status ?? 'Idle'}</span>;
}

function inferLogPlatform(entry: LogEntry, destinations: DestinationInput[]) {
    const destination = destinations.find((item) => item.name === entry.destination);
    return destination?.platform ?? '';
}

function newestFirstLogs(entries: LogEntry[]) {
    return [...entries].sort((left, right) => Date.parse(right.timestamp || '0') - Date.parse(left.timestamp || '0'));
}

function executionModeLabel(mode: ExecutionSummary['mode']) {
    switch (mode) {
        case 'go_live':
            return 'Go Live';
        case 'end_stream':
            return 'End Stream';
        default:
            return 'Execution';
    }
}

function executionResultKind(mode: ExecutionSummary['mode'], message: string) {
    if (mode === 'end_stream') {
        if (message.includes('Live Now cleared') || message.includes('Recovered and cleared pending Live Now session')) {
            return 'Live Now Clear';
        }
        if (message.includes('End stream post')) {
            return 'End Stream Post';
        }
        return 'End Stream Action';
    }
    if (mode === 'go_live' && message.includes('Live Now')) {
        return 'Go Live + Live Now';
    }
    return executionModeLabel(mode);
}

function logCategoryForAction(action: string): LogCategoryFilter {
    if (action.includes('go_live') || action.includes('end_stream') || action === 'clear_pending_live_now') {
        return 'execution';
    }
    if (action.includes('destination') || action === 'test_destination_connection') {
        return 'destinations';
    }
    if (action.includes('settings')) {
        return 'settings';
    }
    if (action.includes('preview')) {
        return 'preview';
    }
    return 'all';
}

function logMatchesStatusFilter(entry: LogEntry, filter: LogStatusFilter) {
    const status = entry.status.toUpperCase();
    switch (filter) {
        case 'failed':
            return status === 'FAILED' || status === 'PARTIAL';
        case 'validation':
            return status === 'VALIDATION_ERROR';
        case 'warning':
            return status === 'WARNING' || status === 'SKIPPED';
        case 'success':
            return status === 'SUCCESS';
        default:
            return true;
    }
}

function logActionLabel(action: string) {
    switch (action) {
        case 'generate_preview':
            return 'Preview';
        case 'go_live':
            return 'Go Live';
        case 'go_live_result':
            return 'Go Live Result';
        case 'force_go_live':
            return 'Confirmed Go Live';
        case 'force_go_live_result':
            return 'Confirmed Go Live Result';
        case 'end_stream':
            return 'End Stream';
        case 'end_stream_result':
            return 'End Stream Result';
        case 'clear_pending_live_now':
            return 'Live Now Recovery';
        case 'save_destination':
            return 'Save Destination';
        case 'delete_destination':
            return 'Delete Destination';
        case 'save_settings':
            return 'Save Settings';
        default:
            return action;
    }
}

function executionStatusMessage(summary: ExecutionSummary | null) {
    if (!summary) {
        return null;
    }

    if (summary.status === 'PARTIAL') {
        if (summary.failedCount > 0 && summary.validationErrorCount > 0) {
            return 'Execution finished with issues. Some destinations failed after execution began, and others still need validation before retrying.';
        }
        if (summary.failedCount > 0) {
            return 'Execution finished with issues. Some destinations failed after execution began.';
        }
        if (summary.validationErrorCount > 0) {
            return 'Execution needs attention. Some destinations still have validation issues before they can run.';
        }
    }

    if (summary.status === 'WARNING' && summary.skippedCount > 0) {
        return 'Execution completed with warnings. One or more destinations were skipped and did not publish.';
    }

    return null;
}

function publishStatusMessage(summary: ExecutionSummary | null) {
    if (!summary) {
        return 'Run Go Live or End Stream to see publish status.';
    }

    const action = executionModeLabel(summary.mode);
    if (summary.status === 'SUCCESS') {
        return `${action} completed successfully${summary.successCount > 0 ? ` for ${summary.successCount} destination${summary.successCount === 1 ? '' : 's'}` : ''}.`;
    }
    return executionStatusMessage(summary) ?? `${action} needs attention.`;
}

function shouldOpenExecutionDetails(summary: ExecutionSummary | null) {
    if (!summary) {
        return false;
    }
    return summary.status !== 'SUCCESS' || summary.failedCount > 0 || summary.validationErrorCount > 0 || summary.skippedCount > 0;
}

function shouldShowExecutionPanel(summary: ExecutionSummary | null, executionError: string | null) {
    if (executionError) {
        return true;
    }
    return shouldOpenExecutionDetails(summary);
}

function hasSecretValue(value: string) {
    return value.trim() !== '';
}

function maskSecretValue(value: string) {
    return hasSecretValue(value) ? STORED_SECRET_TOKEN : '';
}

function resolveSecretInput(value: string, storedValue: string) {
    if (value === STORED_SECRET_TOKEN) {
        return storedValue;
    }
    return value;
}

function destinationSecretCacheFrom(form: DestinationFormState): DestinationSecretCache {
    return {
        discordWebhookKey: form.discordWebhookKey,
        blueskyCredentialKey: form.blueskyCredentialKey,
        mastodonCredentialKey: form.mastodonCredentialKey,
    };
}

function toMaskedDestinationForm(form: DestinationFormState): DestinationFormState {
    return {
        ...form,
        discordWebhookKey: maskSecretValue(form.discordWebhookKey),
        blueskyCredentialKey: maskSecretValue(form.blueskyCredentialKey),
        mastodonCredentialKey: maskSecretValue(form.mastodonCredentialKey),
    };
}

function applyAnnouncementDefaults(announcement: AnnouncementInput, settings: AppSettings, prefillStreamURL = false, prefillHashtags = false): AnnouncementInput {
    return {
        ...announcement,
        streamURL: prefillStreamURL && announcement.streamURL.trim() === '' ? settings.defaultStreamURL : announcement.streamURL,
        hashtags: prefillHashtags && announcement.hashtags.trim() === '' ? settings.defaultHashtags : announcement.hashtags,
    };
}

function asArray<T>(value: T[] | null | undefined): T[] {
    return Array.isArray(value) ? value : [];
}

const BLUESKY_CARD_THUMBNAIL_MAX_BYTES = 1_000_000;
const BLUESKY_ADDITIONAL_IMAGE_MAX_BYTES = 1_000_000;
const DISCORD_CARD_THUMBNAIL_MAX_BYTES = 8_000_000;
const MASTODON_ADDITIONAL_IMAGE_MAX_BYTES = 8_000_000;
const BLUESKY_CARD_THUMBNAIL_MAX_EDGE = 1200;

function fileToDataURL(file: File): Promise<string> {
    return new Promise((resolve, reject) => {
        const reader = new FileReader();
        reader.onload = () => resolve(String(reader.result ?? ''));
        reader.onerror = () => reject(reader.error ?? new Error('Unable to read selected image.'));
        reader.readAsDataURL(file);
    });
}

function loadImage(dataURL: string): Promise<HTMLImageElement> {
    return new Promise((resolve, reject) => {
        const image = new Image();
        image.onload = () => resolve(image);
        image.onerror = () => reject(new Error('Unable to load selected image.'));
        image.src = dataURL;
    });
}

function canvasToBlob(canvas: HTMLCanvasElement, type: string, quality: number): Promise<Blob> {
    return new Promise((resolve, reject) => {
        canvas.toBlob((blob) => {
            if (blob) {
                resolve(blob);
                return;
            }
            reject(new Error('Unable to compress selected image.'));
        }, type, quality);
    });
}

function blobToDataURL(blob: Blob): Promise<string> {
    return new Promise((resolve, reject) => {
        const reader = new FileReader();
        reader.onload = () => resolve(String(reader.result ?? ''));
        reader.onerror = () => reject(reader.error ?? new Error('Unable to read compressed image.'));
        reader.readAsDataURL(blob);
    });
}

async function prepareCardThumbnail(file: File, maxBytes: number): Promise<{ dataURL: string; compressed: boolean }> {
    const original = await fileToDataURL(file);
    if (file.size <= maxBytes) {
        return { dataURL: original, compressed: false };
    }

    const image = await loadImage(original);
    const scale = Math.min(1, BLUESKY_CARD_THUMBNAIL_MAX_EDGE / Math.max(image.naturalWidth, image.naturalHeight));
    const width = Math.max(1, Math.round(image.naturalWidth * scale));
    const height = Math.max(1, Math.round(image.naturalHeight * scale));
    const canvas = document.createElement('canvas');
    canvas.width = width;
    canvas.height = height;

    const context = canvas.getContext('2d');
    if (!context) {
        throw new Error('Unable to prepare selected image.');
    }
    context.fillStyle = '#ffffff';
    context.fillRect(0, 0, width, height);
    context.drawImage(image, 0, 0, width, height);

    for (const quality of [0.9, 0.82, 0.74, 0.66, 0.58, 0.5]) {
        const blob = await canvasToBlob(canvas, 'image/jpeg', quality);
        if (blob.size <= maxBytes) {
            return { dataURL: await blobToDataURL(blob), compressed: true };
        }
    }

    throw new Error(`Card thumbnail is too large to compress below ${Math.round(maxBytes / 1_000_000)} MB.`);
}

function credentialSetupLabel(platform: DestinationInput['platform']) {
    switch (platform) {
        case 'discord':
            return 'Discord webhook';
        case 'bluesky':
            return 'Bluesky app password';
        case 'mastodon':
            return 'Mastodon access token';
        default:
            return 'credential';
    }
}

function guidedSetupSteps(platform: DestinationInput['platform']): GuidedSetupStep[] {
    switch (platform) {
        case 'discord':
            return [
                {
                    title: 'Create and name a channel webhook',
                    detail: 'In Discord, open Server Settings > Integrations > Webhooks, create a webhook for your go-live channel, and name it StreamSignal or the messenger name you want shown in chat.',
                },
                {
                    title: 'Paste the webhook URL below',
                    detail: 'Copy the full webhook URL into StreamSignal. Messages sent through this destination will appear in Discord as the webhook app/bot you named.',
                },
                {
                    title: 'Test before saving',
                    detail: 'Use Test Connection to confirm the webhook responds before you save this destination.',
                },
            ];
        case 'bluesky':
            return [
                {
                    title: 'Create an app password',
                    detail: 'In Bluesky, open Settings > Privacy and Security > App Passwords, create one for StreamSignal, and copy it once.',
                },
                {
                    title: 'Enter your handle and app password',
                    detail: 'Use the Bluesky handle or account identifier for the account that should post and set Live Now.',
                },
                {
                    title: 'Test login before saving',
                    detail: 'Test Connection signs in with the app password so you know the account can be reached before you depend on it.',
                },
            ];
        case 'mastodon':
            return [
                {
                    title: 'Open your Mastodon instance',
                    detail: 'Sign in to the account StreamSignal should post from, then open Preferences > Development. On some instances this is under Settings > Development.',
                },
                {
                    title: 'Create a new application',
                    detail: 'Name it StreamSignal. If Mastodon asks for a website, you can leave it blank or use your stream/profile URL.',
                },
                {
                    title: 'Set the redirect URI',
                    detail: 'Use urn:ietf:wg:oauth:2.0:oob if the field is required. StreamSignal only needs a manually copied access token and does not run an OAuth redirect server.',
                },
                {
                    title: 'Choose scopes',
                    detail: 'Enable write:statuses so StreamSignal can publish posts. If your instance requires account verification for token testing, also enable read:accounts.',
                },
                {
                    title: 'Copy the access token',
                    detail: 'After saving the application, copy the generated access token. Paste the full instance URL, like https://mastodon.social, and the access token into StreamSignal.',
                },
                {
                    title: 'Test before saving',
                    detail: 'Test Connection checks the token against your Mastodon account so invalid, expired, or under-scoped tokens fail early.',
                },
            ];
        default:
            return [];
    }
}

function destinationTemplateLabel(platform: DestinationInput['platform']) {
    return platform === 'bluesky' ? 'Post Template' : 'Template';
}

function destinationSendToMetadata(destination: DestinationInput) {
    const form = toDestinationFormState(destination);
    const parts = [destination.platform, 'Go Live ON'];
    if (destination.platform === 'bluesky') {
        parts.push(form.blueskyLiveStatusEnabled ? 'Live Status ON' : 'Live Status OFF');
    }
    parts.push(form.endStreamEnabled ? 'Post End Stream ON' : 'Post End Stream OFF');
    return parts.join(' · ');
}

function destinationManageMetadata(destination: DestinationInput) {
    return destination.platform;
}

function endStreamActionHint(postCount: number, liveNowClearCount: number) {
    const postLabel = `${postCount} destination${postCount === 1 ? '' : 's'}`;
    if (liveNowClearCount > 0) {
        const liveNowLabel = `${liveNowClearCount} Live Now status${liveNowClearCount === 1 ? '' : 'es'}`;
        return `End Stream will post to ${postLabel} and clear ${liveNowLabel}.`;
    }
    return `End Stream will post to ${postLabel}.`;
}

function destinationCountLabel(count: number) {
    return `${count} destination${count === 1 ? '' : 's'}`;
}

function destinationGroupSummary(groupName: string, destinationCount: number) {
    return `${groupName} · ${destinationCountLabel(destinationCount)}`;
}

function destinationGroupDisplayLabel(groupNames: string[]) {
    if (groupNames.length <= 2) {
        return groupNames.join(', ');
    }
    return `${groupNames[0]} +${groupNames.length - 1}`;
}

function App() {
    const [selectedTab, setSelectedTab] = useState<TabKey>(() => {
        const requestedView = new URLSearchParams(window.location.search).get('view');
        const availableTabs = SHOW_DEBUG_WORKFLOW_UI ? TAB_KEYS : PRODUCTION_TAB_KEYS;
        return availableTabs.includes(requestedView as TabKey) ? (requestedView as TabKey) : 'home';
    });

    const [announcement, setAnnouncement] = useState<AnnouncementInput>(initialAnnouncement);
    const [homeTargetMode, setHomeTargetMode] = useState<HomeTargetMode>('destinations');
    const [selectedDestinationIDs, setSelectedDestinationIDs] = useState<string[]>([]);
    const [selectedDestinationGroupID, setSelectedDestinationGroupID] = useState(DEFAULT_DESTINATION_GROUP_ID);
    const [previewItems, setPreviewItems] = useState<PreviewItem[]>([]);
    const [previewError, setPreviewError] = useState<string | null>(null);
    const [previewLoading, setPreviewLoading] = useState(false);
    const [executionSummary, setExecutionSummary] = useState<ExecutionSummary | null>(null);
    const [executionError, setExecutionError] = useState<string | null>(null);
    const [executionLoading, setExecutionLoading] = useState<'go_live' | 'end_stream' | null>(null);
    const [pendingLiveNowSessions, setPendingLiveNowSessions] = useState<ActiveLiveNowSession[]>([]);
    const [recoveryStatus, setRecoveryStatus] = useState<string | null>(null);
    const [recoveryError, setRecoveryError] = useState<string | null>(null);
    const [recoveryLoadingId, setRecoveryLoadingId] = useState<string | null>(null);

    const [destinations, setDestinations] = useState<DestinationInput[]>([]);
    const [destinationGroups, setDestinationGroups] = useState<DestinationGroupInput[]>([]);
    const [groupDestinations, setGroupDestinationsState] = useState<DestinationInput[]>([]);
    const [destinationGroupIDsByDestination, setDestinationGroupIDsByDestination] = useState<Record<string, string[]>>({});
    const [destinationWorkspaceTab, setDestinationWorkspaceTab] = useState<'destinations' | 'groups'>('destinations');
    const [destinationForm, setDestinationForm] = useState<DestinationFormState>(initialDestination);
    const [destinationGroupForm, setDestinationGroupForm] = useState<DestinationGroupFormState>(createEmptyDestinationGroupForm());
    const [destinationGroupMembershipIDs, setDestinationGroupMembershipIDs] = useState<string[]>([DEFAULT_DESTINATION_GROUP_ID]);
    const [destinationIDsForGroupForm, setDestinationIDsForGroupForm] = useState<string[]>([]);
    const [destinationSecrets, setDestinationSecrets] = useState<DestinationSecretCache>({
        discordWebhookKey: '',
        blueskyCredentialKey: '',
        mastodonCredentialKey: '',
    });
    const [destinationStatus, setDestinationStatus] = useState<string | null>(null);
    const [destinationError, setDestinationError] = useState<string | null>(null);
    const [destinationConnectionResult, setDestinationConnectionResult] = useState<CredentialCheckResult | null>(null);
    const [destinationConnectionLoading, setDestinationConnectionLoading] = useState(false);
    const [destinationHelperStatus, setDestinationHelperStatus] = useState<string | null>(null);
    const [destinationSetupModalOpen, setDestinationSetupModalOpen] = useState(false);

    const [announcementProfiles, setAnnouncementProfiles] = useState<AnnouncementProfileInput[]>([]);
    const [selectedAnnouncementProfileID, setSelectedAnnouncementProfileID] = useState('');
    const [profileForm, setProfileForm] = useState<AnnouncementProfileFormState>(createEmptyAnnouncementProfileForm());
    const [profileStatus, setProfileStatus] = useState<string | null>(null);
    const [profileError, setProfileError] = useState<string | null>(null);
    const [pendingProfileID, setPendingProfileID] = useState<string | null>(null);
    const [profileDirty, setProfileDirty] = useState(false);
    const [saveProfileModalOpen, setSaveProfileModalOpen] = useState(false);
    const [homeProfileName, setHomeProfileName] = useState('');
    const [homeProfileDescription, setHomeProfileDescription] = useState('');
    const [homeProfileError, setHomeProfileError] = useState<string | null>(null);
    const [homeProfileSaving, setHomeProfileSaving] = useState(false);

    const [settings, setSettings] = useState<AppSettings>(defaultSettings);
    const [settingsStatus, setSettingsStatus] = useState<string | null>(null);
    const [settingsError, setSettingsError] = useState<string | null>(null);
    const [logs, setLogs] = useState<LogEntry[]>([]);
    const [logsError, setLogsError] = useState<string | null>(null);
    const [logCategoryFilter, setLogCategoryFilter] = useState<LogCategoryFilter>('all');
    const [logStatusFilter, setLogStatusFilter] = useState<LogStatusFilter>('all');
    const [diagnosticsStatus, setDiagnosticsStatus] = useState<string | null>(null);
    const [workflowToast, setWorkflowToast] = useState<WorkflowToast | null>(null);
    const initializedSelection = useRef(false);
    const streamURLTouched = useRef(false);
    const hashtagsTouched = useRef(false);
    const workflowToastTimeout = useRef<number | null>(null);
    const announcementImageInputRef = useRef<HTMLInputElement | null>(null);
    const announcementProfileImageInputRef = useRef<HTMLInputElement | null>(null);
    const announcementProfileBlueskyThumbnailInputRef = useRef<HTMLInputElement | null>(null);
    const homeBlueskyThumbnailInputRef = useRef<HTMLInputElement | null>(null);
    const discordThumbnailInputRef = useRef<HTMLInputElement | null>(null);
    const blueskyThumbnailInputRef = useRef<HTMLInputElement | null>(null);
    const blueskyAdditionalImageInputRef = useRef<HTMLInputElement | null>(null);
    const mastodonAdditionalImageInputRef = useRef<HTMLInputElement | null>(null);
    const templateInputRef = useRef<HTMLTextAreaElement | null>(null);
    const endStreamTemplateInputRef = useRef<HTMLTextAreaElement | null>(null);
    const activeTemplateInputRef = useRef<HTMLTextAreaElement | null>(null);

    useEffect(() => {
        void refreshDestinations();
        void refreshDestinationGroups();
        void refreshAnnouncementProfiles();
        void refreshSettings();
        if (SHOW_DEBUG_WORKFLOW_UI) {
            void refreshLogs();
        }
        void refreshPendingLiveNowSessions();
    }, []);

    useEffect(() => {
        return () => {
            if (workflowToastTimeout.current !== null) {
                window.clearTimeout(workflowToastTimeout.current);
            }
        };
    }, []);

    async function refreshDestinations() {
        try {
            const items = await listDestinations();
            setDestinations(asArray(items));
            const entries = await Promise.all(asArray(items).map(async (destination) => [destination.id, await getDestinationGroupIDs(destination.id)] as const));
            setDestinationGroupIDsByDestination(Object.fromEntries(entries));
        } catch (err: unknown) {
            const message = errorMessage(err, 'Unable to load destinations.');
            setDestinationError(message);
        }
    }

    async function refreshDestinationGroups() {
        try {
            const groups = asArray(await listDestinationGroups());
            setDestinationGroups(groups);
            if (groups.length > 0 && !groups.some((group) => group.id === selectedDestinationGroupID)) {
                setSelectedDestinationGroupID(DEFAULT_DESTINATION_GROUP_ID);
            }
        } catch (err: unknown) {
            const message = errorMessage(err, 'Unable to load destination groups.');
            setDestinationError(message);
        }
    }

    async function refreshAnnouncementProfiles() {
        try {
            const profiles = asArray(await listAnnouncementProfiles());
            setAnnouncementProfiles(profiles);
            if (selectedAnnouncementProfileID && !profiles.some((profile) => profile.id === selectedAnnouncementProfileID)) {
                setSelectedAnnouncementProfileID('');
                setProfileDirty(false);
            }
        } catch (err: unknown) {
            setProfileError(errorMessage(err, 'Unable to load announcement profiles.'));
        }
    }

    async function refreshSettings() {
        try {
            const current = await getSettings();
            setSettings(current);
            setSelectedDestinationGroupID(current.selectedDestinationGroupID || DEFAULT_DESTINATION_GROUP_ID);
            setAnnouncement((existing) => applyAnnouncementDefaults(existing, current, !streamURLTouched.current, !hashtagsTouched.current));
        } catch (err: unknown) {
            const message = errorMessage(err, 'Unable to load settings.');
            setSettingsError(message);
        }
    }

    useEffect(() => {
        void refreshSelectedGroupDestinations(selectedDestinationGroupID);
    }, [selectedDestinationGroupID, destinations.length]);

    useEffect(() => {
        const availableIDs = destinations.filter((destination) => destination.enabled).map((destination) => destination.id);
        if (availableIDs.length > 0 && !initializedSelection.current) {
            initializedSelection.current = true;
            setSelectedDestinationIDs(availableIDs);
            return;
        }
        setSelectedDestinationIDs((current) => current.filter((id) => availableIDs.includes(id)));
    }, [destinations]);

    async function refreshSelectedGroupDestinations(groupID = selectedDestinationGroupID) {
        try {
            const items = await getGroupDestinations(groupID || DEFAULT_DESTINATION_GROUP_ID);
            setGroupDestinationsState(asArray(items).filter((destination) => destination.enabled));
        } catch (err: unknown) {
            const message = errorMessage(err, 'Unable to load selected group destinations.');
            setDestinationError(message);
            setGroupDestinationsState([]);
        }
    }

    async function refreshLogs() {
        try {
            const entries = await getLogs();
            setLogs(asArray(entries));
        } catch (err: unknown) {
            const message = errorMessage(err, 'Unable to load logs.');
            setLogsError(message);
        }
    }

    async function refreshPendingLiveNowSessions() {
        try {
            const sessions = await listPendingLiveNowSessions();
            setPendingLiveNowSessions(asArray(sessions));
        } catch (err: unknown) {
            const message = errorMessage(err, 'Unable to load pending Live Now sessions.');
            setRecoveryError(message);
        }
    }

    function showToast(message: string, kind: WorkflowToast['kind'] = 'success') {
        if (workflowToastTimeout.current !== null) {
            window.clearTimeout(workflowToastTimeout.current);
        }

        setWorkflowToast({
            id: Date.now(),
            message,
            kind,
        });
        workflowToastTimeout.current = window.setTimeout(() => {
            setWorkflowToast(null);
            workflowToastTimeout.current = null;
        }, 5000);
    }

    async function onPreviewSubmit(event: FormEvent<HTMLFormElement>) {
        event.preventDefault();
        setPreviewLoading(true);
        setPreviewError(null);

        try {
            if (configuredDestinationCount > 0 && activeDestinationCount === 0) {
                setPreviewError(homeTargetMode === 'group' ? 'No enabled destinations in this group.' : 'Select at least one destination for this session.');
                setPreviewItems([]);
                return;
            }

            const preview = await generatePreview(announcement, activeDestinationGroupID, activeDestinationIDs);
            setPreviewItems(asArray(preview));
        } catch (err: unknown) {
            const message = errorMessage(err, 'Unable to generate previews.');
            setPreviewError(message);
            setPreviewItems([]);
        } finally {
            setPreviewLoading(false);
        }
    }

    async function runGoLive() {
        setExecutionLoading('go_live');
        setExecutionError(null);

        try {
            if (configuredDestinationCount > 0 && activeDestinationCount === 0) {
                setExecutionError(homeTargetMode === 'group' ? 'No enabled destinations in this group.' : 'Select at least one destination for this session.');
                setExecutionSummary(null);
                return;
            }

            const summary = await goLive(announcement, activeDestinationGroupID, activeDestinationIDs);
            setExecutionSummary(summary);
            if (!summary.requiresDuplicateConfirmation) {
                showToast(publishStatusMessage(summary), summary.status === 'SUCCESS' ? 'success' : 'warning');
            }
        } catch (err: unknown) {
            const message = errorMessage(err, 'Unable to run execution.');
            setExecutionError(message);
            setExecutionSummary(null);
        } finally {
            setExecutionLoading(null);
        }
    }

    async function onConfirmDuplicateGoLive() {
        setExecutionLoading('go_live');
        setExecutionError(null);
        try {
            const summary = await forceGoLive(announcement, activeDestinationGroupID, activeDestinationIDs);
            setExecutionSummary(summary);
            showToast(publishStatusMessage(summary), summary.status === 'SUCCESS' ? 'success' : 'warning');
        } catch (err: unknown) {
            const message = errorMessage(err, 'Unable to confirm Go Live.');
            setExecutionError(message);
        } finally {
            setExecutionLoading(null);
        }
    }

    async function onEndStream() {
        setExecutionLoading('end_stream');
        setExecutionError(null);

        try {
            if (configuredDestinationCount > 0 && activeDestinationCount === 0) {
                setExecutionError(homeTargetMode === 'group' ? 'No enabled destinations in this group.' : 'Select at least one destination for this session.');
                setExecutionSummary(null);
                return;
            }

            const summary = await endStream(announcement, activeDestinationGroupID, activeDestinationIDs);
            setExecutionSummary(summary);
            showToast(publishStatusMessage(summary), summary.status === 'SUCCESS' ? 'success' : 'warning');
            await refreshPendingLiveNowSessions();
        } catch (err: unknown) {
            const message = errorMessage(err, 'Unable to end stream.');
            setExecutionError(message);
        } finally {
            setExecutionLoading(null);
        }
    }

    async function onClearPendingLiveNow(destinationID: string) {
        setRecoveryError(null);
        setRecoveryStatus(null);
        setRecoveryLoadingId(destinationID);

        try {
            const result = await clearPendingLiveNowSession(destinationID);
            if (result.state === 'SUCCESS') {
                setRecoveryStatus(null);
                showToast(result.message);
            } else {
                setRecoveryError(result.message);
            }
            await refreshPendingLiveNowSessions();
        } catch (err: unknown) {
            const message = errorMessage(err, 'Unable to clear pending Live Now session.');
            setRecoveryError(message);
        } finally {
            setRecoveryLoadingId(null);
        }
    }

    async function onDestinationSubmit(event: FormEvent<HTMLFormElement>) {
        event.preventDefault();
        setDestinationError(null);
        setDestinationStatus(null);
        const wasEditingDestination = destinationForm.id !== '';
        const savedPlatform = destinationForm.platform;

        try {
            const resolvedForm: DestinationFormState = {
                ...destinationForm,
                discordWebhookKey: resolveSecretInput(destinationForm.discordWebhookKey, destinationSecrets.discordWebhookKey),
                blueskyCredentialKey: resolveSecretInput(destinationForm.blueskyCredentialKey, destinationSecrets.blueskyCredentialKey),
                mastodonCredentialKey: resolveSecretInput(destinationForm.mastodonCredentialKey, destinationSecrets.mastodonCredentialKey),
            };
            const saved = await saveDestination(toDestinationInput(resolvedForm));
            const membershipIDs = destinationGroupMembershipIDs.length > 0 ? destinationGroupMembershipIDs : [DEFAULT_DESTINATION_GROUP_ID];
            await saveDestinationGroupsForDestination(saved.id, membershipIDs);
            const savedForm = toDestinationFormState(saved);
            if (wasEditingDestination) {
                setDestinationSecrets(destinationSecretCacheFrom(savedForm));
                setDestinationForm(toMaskedDestinationForm(savedForm));
            } else {
                resetDestinationForm(savedPlatform);
            }
            setDestinationStatus(null);
            showToast('Destination saved.');
            await refreshDestinations();
            await refreshSelectedGroupDestinations();
        } catch (err: unknown) {
            const message = errorMessage(err, 'Unable to save destination.');
            setDestinationError(message);
        }
    }

    function resetDestinationForm(platform = destinationForm.platform) {
        setDestinationSecrets({
            discordWebhookKey: '',
            blueskyCredentialKey: '',
            mastodonCredentialKey: '',
        });
        setDestinationForm(createEmptyDestinationForm(platform));
        setDestinationGroupMembershipIDs([DEFAULT_DESTINATION_GROUP_ID]);
        setDestinationConnectionResult(null);
        setDestinationHelperStatus(null);
        if (discordThumbnailInputRef.current) {
            discordThumbnailInputRef.current.value = '';
        }
        if (blueskyThumbnailInputRef.current) {
            blueskyThumbnailInputRef.current.value = '';
        }
        if (blueskyAdditionalImageInputRef.current) {
            blueskyAdditionalImageInputRef.current.value = '';
        }
        if (mastodonAdditionalImageInputRef.current) {
            mastodonAdditionalImageInputRef.current.value = '';
        }
    }

    function editDestination(destination: DestinationInput) {
        const form = toDestinationFormState(destination);
        setDestinationSecrets(destinationSecretCacheFrom(form));
        setDestinationForm(toMaskedDestinationForm(form));
        setDestinationGroupMembershipIDs(destinationGroupIDsByDestination[destination.id] ?? [DEFAULT_DESTINATION_GROUP_ID]);
        setDestinationConnectionResult(null);
        setDestinationHelperStatus(null);
    }

    function duplicateDestination(destination: DestinationInput) {
        const form = toDestinationFormState(destination);
        const duplicateForm: DestinationFormState = {
            ...form,
            id: '',
            name: `${form.name || platformLabel(form.platform)} Copy`,
            createdAt: '',
            updatedAt: '',
        };
        setDestinationSecrets(destinationSecretCacheFrom(form));
        setDestinationForm(toMaskedDestinationForm(duplicateForm));
        setDestinationGroupMembershipIDs(destinationGroupIDsByDestination[destination.id] ?? [DEFAULT_DESTINATION_GROUP_ID]);
        setDestinationConnectionResult(null);
        setDestinationHelperStatus('Duplicate ready. Review connection details, then save it as a new destination.');
    }

    async function onTestDestinationConnection() {
        setDestinationError(null);
        setDestinationStatus(null);
        setDestinationConnectionResult(null);
        setDestinationConnectionLoading(true);

        try {
            const resolvedForm: DestinationFormState = {
                ...destinationForm,
                discordWebhookKey: resolveSecretInput(destinationForm.discordWebhookKey, destinationSecrets.discordWebhookKey),
                blueskyCredentialKey: resolveSecretInput(destinationForm.blueskyCredentialKey, destinationSecrets.blueskyCredentialKey),
                mastodonCredentialKey: resolveSecretInput(destinationForm.mastodonCredentialKey, destinationSecrets.mastodonCredentialKey),
            };
            const result = await testDestinationConnection(toDestinationInput(resolvedForm));
            setDestinationConnectionResult(result);
            if (result.state === 'SUCCESS') {
                showToast(result.message);
            }
        } catch (err: unknown) {
            const message = errorMessage(err, 'Unable to test destination connection.');
            setDestinationError(message);
        } finally {
            setDestinationConnectionLoading(false);
        }
    }

    async function onTestSavedDestination(destination: DestinationInput) {
        setDestinationError(null);
        setDestinationStatus(null);
        setDestinationConnectionResult(null);
        setDestinationConnectionLoading(true);

        try {
            const current = await getDestination(destination.id);
            editDestination(current);
            const result = await testDestinationConnection(current);
            setDestinationConnectionResult(result);
            if (result.state === 'SUCCESS') {
                showToast(result.message);
            }
        } catch (err: unknown) {
            const message = errorMessage(err, 'Unable to test destination connection.');
            setDestinationError(message);
        } finally {
            setDestinationConnectionLoading(false);
        }
    }

    async function onDeleteSavedDestination(destination: DestinationInput) {
        setDestinationError(null);
        setDestinationStatus(null);

        try {
            await deleteDestination(destination.id);
            if (destinationForm.id === destination.id) {
                setDestinationForm(createEmptyDestinationForm(destination.platform));
                setDestinationSecrets({
                    discordWebhookKey: '',
                    blueskyCredentialKey: '',
                    mastodonCredentialKey: '',
                });
            }
            showToast('Destination deleted.');
            await refreshDestinations();
            await refreshSelectedGroupDestinations();
        } catch (err: unknown) {
            const message = errorMessage(err, 'Unable to delete destination.');
            setDestinationError(message);
        }
    }

    async function onDeleteDestination() {
        if (!destinationForm.id) {
            return;
        }

        setDestinationError(null);
        setDestinationStatus(null);

        try {
            const deletedID = destinationForm.id;
            await deleteDestination(destinationForm.id);
            setDestinationForm(createEmptyDestinationForm(destinationForm.platform));
            setDestinationSecrets({
                discordWebhookKey: '',
                blueskyCredentialKey: '',
                mastodonCredentialKey: '',
            });
            setDestinationStatus(null);
            showToast('Destination deleted.');
            await refreshDestinations();
            await refreshSelectedGroupDestinations();
        } catch (err: unknown) {
            const message = errorMessage(err, 'Unable to delete destination.');
            setDestinationError(message);
        }
    }

    async function onSettingsSubmit(event: FormEvent<HTMLFormElement>) {
        event.preventDefault();
        setSettingsError(null);
        setSettingsStatus(null);

        try {
            const saved = await saveSettings(settings);
            setSettings(saved);
            setAnnouncement((existing) => applyAnnouncementDefaults(existing, saved, !streamURLTouched.current, !hashtagsTouched.current));
            setSettingsStatus(null);
            showToast('Settings saved.');
        } catch (err: unknown) {
            const message = errorMessage(err, 'Unable to save settings.');
            setSettingsError(message);
        }
    }

    async function onCopyDiagnostics() {
        setLogsError(null);
        setDiagnosticsStatus(null);

        try {
            const diagnostics = await getDiagnostics();
            if (navigator.clipboard?.writeText) {
                await navigator.clipboard.writeText(diagnostics);
                setDiagnosticsStatus(null);
                showToast('Diagnostics copied.');
            } else {
                setLogsError('Clipboard is unavailable in this environment.');
            }
        } catch (err: unknown) {
            const message = errorMessage(err, 'Unable to copy diagnostics.');
            setLogsError(message);
        }
    }

    function updateAnnouncement<K extends keyof AnnouncementInput>(field: K, value: AnnouncementInput[K]) {
        if (field === 'streamURL') {
            streamURLTouched.current = true;
        }
        if (field === 'hashtags') {
            hashtagsTouched.current = true;
        }
        if (selectedAnnouncementProfileID) {
            setProfileDirty(true);
        }
        setAnnouncement((current) => ({
            ...current,
            [field]: value,
        }));
    }

    function updateDestination<K extends keyof DestinationFormState>(field: K, value: DestinationFormState[K]) {
        setDestinationConnectionResult(null);
        setDestinationHelperStatus(null);
        setDestinationForm((current) => ({
            ...current,
            [field]: value,
        }));
    }

    function clearAnnouncementImageUpload() {
        if (announcementImageInputRef.current) {
            announcementImageInputRef.current.value = '';
        }
        updateAnnouncement('announcementImageDataURL', '');
    }

    function clearHomeBlueskyThumbnailUpload() {
        if (homeBlueskyThumbnailInputRef.current) {
            homeBlueskyThumbnailInputRef.current.value = '';
        }
        updateAnnouncement('blueskyCardThumbnailDataURL', '');
    }

    async function onAnnouncementImageUpload(file: File | undefined) {
        setPreviewError(null);
        if (!file) {
            return;
        }
        if (!file.type.startsWith('image/')) {
            setPreviewError('Announcement image must be an image.');
            return;
        }
        try {
            const image = await prepareCardThumbnail(file, MASTODON_ADDITIONAL_IMAGE_MAX_BYTES);
            setAnnouncement((current) => ({
                ...current,
                announcementImageURL: '',
                announcementImageDataURL: image.dataURL,
            }));
            if (selectedAnnouncementProfileID) {
                setProfileDirty(true);
            }
            if (announcementImageInputRef.current) {
                announcementImageInputRef.current.value = '';
            }
        } catch (err: unknown) {
            if (announcementImageInputRef.current) {
                announcementImageInputRef.current.value = '';
            }
            setPreviewError(errorMessage(err, 'Unable to prepare selected image.'));
        }
    }

    async function onHomeBlueskyThumbnailUpload(file: File | undefined) {
        setPreviewError(null);
        if (!file) {
            return;
        }
        if (!file.type.startsWith('image/')) {
            setPreviewError('Bluesky card thumbnail must be an image.');
            return;
        }
        try {
            const image = await prepareCardThumbnail(file, BLUESKY_CARD_THUMBNAIL_MAX_BYTES);
            setAnnouncement((current) => ({
                ...current,
                blueskyCardThumbnailURL: '',
                blueskyCardThumbnailDataURL: image.dataURL,
            }));
            if (selectedAnnouncementProfileID) {
                setProfileDirty(true);
            }
            if (homeBlueskyThumbnailInputRef.current) {
                homeBlueskyThumbnailInputRef.current.value = '';
            }
        } catch (err: unknown) {
            if (homeBlueskyThumbnailInputRef.current) {
                homeBlueskyThumbnailInputRef.current.value = '';
            }
            setPreviewError(errorMessage(err, 'Unable to prepare selected image.'));
        }
    }

    function clearBlueskyThumbnailImage() {
        if (blueskyThumbnailInputRef.current) {
            blueskyThumbnailInputRef.current.value = '';
        }
        updateDestination('blueskyCardThumbnailDataURL', '');
    }

    function clearBlueskyAdditionalImage() {
        if (blueskyAdditionalImageInputRef.current) {
            blueskyAdditionalImageInputRef.current.value = '';
        }
        updateDestination('blueskyAdditionalImageDataURL', '');
    }

    function clearMastodonAdditionalImage() {
        if (mastodonAdditionalImageInputRef.current) {
            mastodonAdditionalImageInputRef.current.value = '';
        }
        updateDestination('mastodonAdditionalImageDataURL', '');
    }

    function clearDiscordThumbnailImage() {
        if (discordThumbnailInputRef.current) {
            discordThumbnailInputRef.current.value = '';
        }
        updateDestination('discordCardThumbnailDataURL', '');
    }

    async function onDiscordThumbnailUpload(file: File | undefined) {
        setDestinationConnectionResult(null);
        setDestinationHelperStatus(null);
        setDestinationError(null);

        if (!file) {
            return;
        }
        if (!file.type.startsWith('image/')) {
            setDestinationError('Additional image must be an image.');
            return;
        }

        try {
            const thumbnail = await prepareCardThumbnail(file, DISCORD_CARD_THUMBNAIL_MAX_BYTES);
            setDestinationForm((current) => ({
                ...current,
                discordCardThumbnailURL: '',
                discordCardThumbnailDataURL: thumbnail.dataURL,
            }));
            if (discordThumbnailInputRef.current) {
                discordThumbnailInputRef.current.value = '';
            }
            setDestinationHelperStatus(thumbnail.compressed ? `Selected and compressed ${file.name}` : `Selected ${file.name}`);
        } catch (err: unknown) {
            if (discordThumbnailInputRef.current) {
                discordThumbnailInputRef.current.value = '';
            }
            const message = errorMessage(err, 'Unable to prepare selected image.');
            setDestinationError(message);
        }
    }

    async function onBlueskyThumbnailUpload(file: File | undefined) {
        setDestinationConnectionResult(null);
        setDestinationHelperStatus(null);
        setDestinationError(null);

        if (!file) {
            return;
        }
        if (!file.type.startsWith('image/')) {
            setDestinationError('Card thumbnail must be an image.');
            return;
        }

        try {
            const thumbnail = await prepareCardThumbnail(file, BLUESKY_CARD_THUMBNAIL_MAX_BYTES);
            setDestinationForm((current) => ({
                ...current,
                blueskyCardThumbnailURL: '',
                blueskyCardThumbnailDataURL: thumbnail.dataURL,
            }));
            if (blueskyThumbnailInputRef.current) {
                blueskyThumbnailInputRef.current.value = '';
            }
            setDestinationHelperStatus(thumbnail.compressed ? `Selected and compressed ${file.name}` : `Selected ${file.name}`);
        } catch (err: unknown) {
            if (blueskyThumbnailInputRef.current) {
                blueskyThumbnailInputRef.current.value = '';
            }
            const message = errorMessage(err, 'Unable to prepare selected image.');
            setDestinationError(message);
        }
    }

    async function onBlueskyAdditionalImageUpload(file: File | undefined) {
        setDestinationConnectionResult(null);
        setDestinationHelperStatus(null);
        setDestinationError(null);

        if (!file) {
            return;
        }
        if (!file.type.startsWith('image/')) {
            setDestinationError('Additional image must be an image.');
            return;
        }

        try {
            const image = await prepareCardThumbnail(file, BLUESKY_ADDITIONAL_IMAGE_MAX_BYTES);
            setDestinationForm((current) => ({
                ...current,
                blueskyAdditionalImageURL: '',
                blueskyAdditionalImageDataURL: image.dataURL,
            }));
            if (blueskyAdditionalImageInputRef.current) {
                blueskyAdditionalImageInputRef.current.value = '';
            }
            setDestinationHelperStatus(image.compressed ? `Selected and compressed ${file.name}` : `Selected ${file.name}`);
        } catch (err: unknown) {
            if (blueskyAdditionalImageInputRef.current) {
                blueskyAdditionalImageInputRef.current.value = '';
            }
            const message = errorMessage(err, 'Unable to prepare selected image.');
            setDestinationError(message);
        }
    }

    async function onMastodonAdditionalImageUpload(file: File | undefined) {
        setDestinationConnectionResult(null);
        setDestinationHelperStatus(null);
        setDestinationError(null);

        if (!file) {
            return;
        }
        if (!file.type.startsWith('image/')) {
            setDestinationError('Additional image must be an image.');
            return;
        }

        try {
            const image = await prepareCardThumbnail(file, MASTODON_ADDITIONAL_IMAGE_MAX_BYTES);
            setDestinationForm((current) => ({
                ...current,
                mastodonAdditionalImageURL: '',
                mastodonAdditionalImageDataURL: image.dataURL,
            }));
            if (mastodonAdditionalImageInputRef.current) {
                mastodonAdditionalImageInputRef.current.value = '';
            }
            setDestinationHelperStatus(image.compressed ? `Selected and compressed ${file.name}` : `Selected ${file.name}`);
        } catch (err: unknown) {
            if (mastodonAdditionalImageInputRef.current) {
                mastodonAdditionalImageInputRef.current.value = '';
            }
            const message = errorMessage(err, 'Unable to prepare selected image.');
            setDestinationError(message);
        }
    }

    function insertTemplateVariable(variable: string) {
        setDestinationHelperStatus(null);
        const target = activeTemplateInputRef.current ?? templateInputRef.current;
        const field = target === endStreamTemplateInputRef.current ? 'endStreamTemplate' : 'template';
        const currentValue = field === 'endStreamTemplate' ? destinationForm.endStreamTemplate : destinationForm.template;
        const start = target?.selectionStart ?? currentValue.length;
        const end = target?.selectionEnd ?? currentValue.length;
        const nextValue = `${currentValue.slice(0, start)}${variable}${currentValue.slice(end)}`;

        updateDestination(field, nextValue);
        setDestinationHelperStatus(`Inserted ${variable}`);

        if (target) {
            window.requestAnimationFrame(() => {
                target.focus();
                const nextCursor = start + variable.length;
                target.setSelectionRange(nextCursor, nextCursor);
            });
        }
    }

    function updateSettings<K extends keyof AppSettings>(field: K, value: AppSettings[K]) {
        setSettings((current) => ({
            ...current,
            [field]: value,
        }));
    }

    function updateProfileForm<K extends keyof AnnouncementProfileFormState>(field: K, value: AnnouncementProfileFormState[K]) {
        setProfileStatus(null);
        setProfileError(null);
        setProfileForm((current) => ({
            ...current,
            [field]: value,
        }));
    }

    function resetProfileForm() {
        setProfileForm(createEmptyAnnouncementProfileForm());
        setProfileStatus(null);
        setProfileError(null);
        if (announcementProfileImageInputRef.current) {
            announcementProfileImageInputRef.current.value = '';
        }
        if (announcementProfileBlueskyThumbnailInputRef.current) {
            announcementProfileBlueskyThumbnailInputRef.current.value = '';
        }
    }

    function editAnnouncementProfile(profile: AnnouncementProfileInput) {
        setProfileForm(toAnnouncementProfileFormState(profile));
        setProfileStatus(null);
        setProfileError(null);
    }

    async function onAnnouncementProfileSubmit(event: FormEvent<HTMLFormElement>) {
        event.preventDefault();
        setProfileStatus(null);
        setProfileError(null);
        try {
            const wasEditing = profileForm.id !== '';
            const saved = await saveAnnouncementProfile(toAnnouncementProfileInput(profileForm));
            if (wasEditing) {
                setProfileForm(toAnnouncementProfileFormState(saved));
            } else {
                resetProfileForm();
            }
            setProfileStatus('Announcement preset saved.');
            await refreshAnnouncementProfiles();
        } catch (err: unknown) {
            setProfileError(errorMessage(err, 'Unable to save announcement profile.'));
        }
    }

    async function onDuplicateAnnouncementProfile(profile: AnnouncementProfileInput) {
        setProfileError(null);
        try {
            const duplicate = await duplicateAnnouncementProfile(profile.id);
            setProfileForm(toAnnouncementProfileFormState(duplicate));
            setProfileStatus('Announcement preset duplicated.');
            await refreshAnnouncementProfiles();
        } catch (err: unknown) {
            setProfileError(errorMessage(err, 'Unable to duplicate announcement profile.'));
        }
    }

    async function onDeleteAnnouncementProfile(profile: AnnouncementProfileInput) {
        setProfileError(null);
        try {
            await deleteAnnouncementProfile(profile.id);
            if (selectedAnnouncementProfileID === profile.id) {
                setSelectedAnnouncementProfileID('');
                setProfileDirty(false);
            }
            if (profileForm.id === profile.id) {
                resetProfileForm();
            }
            setProfileStatus('Announcement preset deleted.');
            await refreshAnnouncementProfiles();
        } catch (err: unknown) {
            setProfileError(errorMessage(err, 'Unable to delete announcement profile.'));
        }
    }

    function openSaveCurrentAnnouncementProfile() {
        setHomeProfileName('');
        setHomeProfileDescription('');
        setHomeProfileError(null);
        setSaveProfileModalOpen(true);
    }

    function closeSaveCurrentAnnouncementProfile() {
        if (homeProfileSaving) {
            return;
        }
        setSaveProfileModalOpen(false);
        setHomeProfileError(null);
    }

    function currentAnnouncementProfileInput(base: Pick<AnnouncementProfileInput, 'id' | 'name' | 'description'>): AnnouncementProfileInput {
        return {
            id: base.id,
            name: base.name,
            description: base.description,
            defaultStreamTitle: announcement.streamTitle,
            defaultStreamURL: announcement.streamURL,
            defaultCategory: announcement.category,
            defaultMessage: announcement.message,
            defaultHashtags: announcement.hashtags,
            defaultAnnouncementImageURL: announcement.announcementImageURL,
            defaultAnnouncementImageDataURL: announcement.announcementImageDataURL,
            defaultBlueskyCardThumbnailURL: announcement.blueskyCardThumbnailURL,
            defaultBlueskyCardThumbnailDataURL: announcement.blueskyCardThumbnailDataURL,
            defaultGroupID: homeTargetMode === 'group' ? selectedDestinationGroupID : '',
        };
    }

    async function onSaveCurrentAnnouncementProfile(event: FormEvent<HTMLFormElement>) {
        event.preventDefault();
        const name = homeProfileName.trim();
        const description = homeProfileDescription.trim();

        setHomeProfileError(null);
        if (!name) {
            setHomeProfileError('Profile name is required.');
            return;
        }
        if (announcementProfiles.some((profile) => profile.name.toLocaleLowerCase() === name.toLocaleLowerCase())) {
            setHomeProfileError('A profile with this name already exists.');
            return;
        }

        setHomeProfileSaving(true);
        try {
            const saved = await saveAnnouncementProfile(currentAnnouncementProfileInput({ id: '', name, description }));
            setAnnouncementProfiles((current) => [saved, ...current]);
            setSelectedAnnouncementProfileID(saved.id);
            setProfileDirty(false);
            setSaveProfileModalOpen(false);
            setHomeProfileName('');
            setHomeProfileDescription('');
            if (saved.defaultGroupID) {
                setHomeTargetMode('group');
                await selectDestinationGroup(saved.defaultGroupID);
            }
            await refreshAnnouncementProfiles();
            showToast(`Profile saved: ${saved.name}`);
        } catch (err: unknown) {
            const message = errorMessage(err, 'Unable to save announcement profile.');
            setHomeProfileError(message.toLocaleLowerCase().includes('unique') ? 'A profile with this name already exists.' : message);
        } finally {
            setHomeProfileSaving(false);
        }
    }

    async function onUpdateCurrentAnnouncementProfile() {
        if (!activeAnnouncementProfile || homeProfileSaving) {
            return;
        }

        setHomeProfileSaving(true);
        setProfileError(null);
        try {
            const saved = await saveAnnouncementProfile(currentAnnouncementProfileInput(activeAnnouncementProfile));
            setAnnouncementProfiles((current) => current.map((profile) => profile.id === saved.id ? saved : profile));
            setSelectedAnnouncementProfileID(saved.id);
            setProfileDirty(false);
            if (profileForm.id === saved.id) {
                setProfileForm(toAnnouncementProfileFormState(saved));
            }
            await refreshAnnouncementProfiles();
            showToast(`Profile updated: ${saved.name}`);
        } catch (err: unknown) {
            setProfileError(errorMessage(err, 'Unable to update announcement profile.'));
        } finally {
            setHomeProfileSaving(false);
        }
    }

    async function selectAnnouncementProfile(profileID: string) {
        if (profileID === selectedAnnouncementProfileID) {
            return;
        }
        if (profileDirty && selectedAnnouncementProfileID && profileID !== '') {
            setPendingProfileID(profileID);
            return;
        }
        await applyAnnouncementProfile(profileID);
    }

    async function applyAnnouncementProfile(profileID: string) {
        setPendingProfileID(null);
        if (profileID === '') {
            setSelectedAnnouncementProfileID('');
            setProfileDirty(false);
            return;
        }
        const profile = announcementProfiles.find((item) => item.id === profileID);
        if (!profile) {
            return;
        }
        streamURLTouched.current = profile.defaultStreamURL.trim() !== '';
        hashtagsTouched.current = profile.defaultHashtags.trim() !== '';
        setAnnouncement((current) => ({
            ...current,
            streamTitle: profile.defaultStreamTitle,
            streamURL: profile.defaultStreamURL,
            category: profile.defaultCategory,
            message: profile.defaultMessage,
            hashtags: profile.defaultHashtags,
            announcementImageURL: profile.defaultAnnouncementImageURL,
            announcementImageDataURL: profile.defaultAnnouncementImageDataURL,
            blueskyCardThumbnailURL: profile.defaultBlueskyCardThumbnailURL,
            blueskyCardThumbnailDataURL: profile.defaultBlueskyCardThumbnailDataURL,
        }));
        setSelectedAnnouncementProfileID(profile.id);
        setProfileDirty(false);
        try {
            await markAnnouncementProfileUsed(profile.id);
            await refreshAnnouncementProfiles();
        } catch (err: unknown) {
            setProfileError(errorMessage(err, 'Unable to update announcement profile usage.'));
        }
        if (profile.defaultGroupID) {
            setHomeTargetMode('group');
            await selectDestinationGroup(profile.defaultGroupID);
        }
    }

    async function onProfileImageUpload(file: File | undefined, target: 'announcement' | 'bluesky-card') {
        setProfileError(null);
        if (!file) {
            return;
        }
        if (!file.type.startsWith('image/')) {
            setProfileError('Profile media must be an image.');
            return;
        }
        const ref = target === 'announcement' ? announcementProfileImageInputRef : announcementProfileBlueskyThumbnailInputRef;
        try {
            const image = await prepareCardThumbnail(file, target === 'announcement' ? MASTODON_ADDITIONAL_IMAGE_MAX_BYTES : BLUESKY_CARD_THUMBNAIL_MAX_BYTES);
            setProfileForm((current) => ({
                ...current,
                ...(target === 'announcement'
                    ? { defaultAnnouncementImageURL: '', defaultAnnouncementImageDataURL: image.dataURL }
                    : { defaultBlueskyCardThumbnailURL: '', defaultBlueskyCardThumbnailDataURL: image.dataURL }),
            }));
            if (ref.current) {
                ref.current.value = '';
            }
        } catch (err: unknown) {
            if (ref.current) {
                ref.current.value = '';
            }
            setProfileError(errorMessage(err, 'Unable to prepare selected image.'));
        }
    }

    function toggleSessionDestination(destinationID: string) {
        setSelectedDestinationIDs((current) =>
            current.includes(destinationID) ? current.filter((id) => id !== destinationID) : [...current, destinationID],
        );
    }

    function selectAllDestinations() {
        setSelectedDestinationIDs(destinations.filter((destination) => destination.enabled).map((destination) => destination.id));
    }

    function clearSelectedDestinations() {
        setSelectedDestinationIDs([]);
    }

    async function selectDestinationGroup(groupID: string) {
        setSelectedDestinationGroupID(groupID);
        setPreviewItems([]);
        setPreviewError(null);
        const nextSettings = {
            ...settings,
            selectedDestinationGroupID: groupID,
        };
        setSettings(nextSettings);
        try {
            await saveSettings(nextSettings);
            await refreshSelectedGroupDestinations(groupID);
        } catch (err: unknown) {
            setSettingsError(errorMessage(err, 'Unable to save selected destination group.'));
        }
    }

    function updateDestinationGroupForm<K extends keyof DestinationGroupFormState>(field: K, value: DestinationGroupFormState[K]) {
        setDestinationGroupForm((current) => ({
            ...current,
            [field]: value,
        }));
    }

    async function editDestinationGroup(group: DestinationGroupInput) {
        setDestinationGroupForm(toDestinationGroupFormState(group));
        try {
            const items = await getGroupDestinations(group.id);
            setDestinationIDsForGroupForm(asArray(items).map((destination) => destination.id));
        } catch (err: unknown) {
            setDestinationError(errorMessage(err, 'Unable to load destination group members.'));
        }
    }

    function resetDestinationGroupForm() {
        setDestinationGroupForm(createEmptyDestinationGroupForm());
        setDestinationIDsForGroupForm([]);
    }

    function toggleDestinationGroupMembership(groupID: string) {
        setDestinationGroupMembershipIDs((current) => {
            if (current.includes(groupID)) {
                const next = current.filter((id) => id !== groupID);
                return next.length > 0 ? next : [DEFAULT_DESTINATION_GROUP_ID];
            }
            return [...current, groupID];
        });
    }

    function toggleGroupFormDestination(destinationID: string) {
        setDestinationIDsForGroupForm((current) => (current.includes(destinationID) ? current.filter((id) => id !== destinationID) : [...current, destinationID]));
    }

    async function onDestinationGroupSubmit(event: FormEvent<HTMLFormElement>) {
        event.preventDefault();
        setDestinationError(null);
        const wasEditingGroup = destinationGroupForm.id !== '';
        try {
            const saved = await saveDestinationGroup({
                id: destinationGroupForm.id,
                name: destinationGroupForm.name,
                description: destinationGroupForm.description,
            });
            await setGroupDestinations(saved.id, destinationIDsForGroupForm);
            if (wasEditingGroup) {
                setDestinationGroupForm(toDestinationGroupFormState(saved));
            } else {
                resetDestinationGroupForm();
            }
            showToast('Destination group saved.');
            await refreshDestinationGroups();
            await refreshSelectedGroupDestinations();
        } catch (err: unknown) {
            setDestinationError(errorMessage(err, 'Unable to save destination group.'));
        }
    }

    async function onDuplicateDestinationGroup(group: DestinationGroupInput) {
        setDestinationError(null);
        try {
            const duplicate = await duplicateDestinationGroup(group.id);
            setDestinationGroupForm(toDestinationGroupFormState(duplicate));
            showToast('Destination group duplicated.');
            await refreshDestinationGroups();
        } catch (err: unknown) {
            setDestinationError(errorMessage(err, 'Unable to duplicate destination group.'));
        }
    }

    async function onDeleteDestinationGroup(group: DestinationGroupInput) {
        setDestinationError(null);
        try {
            await deleteDestinationGroup(group.id);
            if (selectedDestinationGroupID === group.id) {
                await selectDestinationGroup(DEFAULT_DESTINATION_GROUP_ID);
            }
            resetDestinationGroupForm();
            showToast('Destination group deleted.');
            await refreshDestinationGroups();
        } catch (err: unknown) {
            setDestinationError(errorMessage(err, 'Unable to delete destination group.'));
        }
    }

    const configuredDestinationCount = destinations.length;
    const selectedDestinationGroup = destinationGroups.find((group) => group.id === selectedDestinationGroupID) ?? destinationGroups.find((group) => group.id === DEFAULT_DESTINATION_GROUP_ID);
    const selectedGroupName = selectedDestinationGroup?.name ?? DEFAULT_DESTINATION_GROUP_NAME;
    const selectedGroupDestinationCount = groupDestinations.length;
    const enabledDestinations = destinations.filter((destination) => destination.enabled);
    const selectedDestinationIDSet = new Set(selectedDestinationIDs);
    const individuallySelectedDestinations = enabledDestinations.filter((destination) => selectedDestinationIDSet.has(destination.id));
    const activeDestinations = homeTargetMode === 'group' ? groupDestinations : individuallySelectedDestinations;
    const activeDestinationCount = activeDestinations.length;
    const activeDestinationIDs = homeTargetMode === 'group' ? [] : activeDestinations.map((destination) => destination.id);
    const activeDestinationGroupID = homeTargetMode === 'group' ? selectedDestinationGroupID : '';
    const activeDestinationIDSet = new Set(activeDestinations.map((destination) => destination.id));
    const activeAnnouncementProfile = announcementProfiles.find((profile) => profile.id === selectedAnnouncementProfileID);
    const activeDestinationsSupportMedia = activeDestinations.some((destination) => destination.platform === 'discord' || destination.platform === 'bluesky' || destination.platform === 'mastodon');
    const activeDestinationsIncludeBluesky = activeDestinations.some((destination) => destination.platform === 'bluesky');
    const selectedEndStreamPostCount = activeDestinations.filter((destination) => toDestinationFormState(destination).endStreamEnabled).length;
    const selectedLiveNowClearCount = pendingLiveNowSessions.filter((session) => activeDestinationIDSet.has(session.destinationID)).length;

    function destinationGroupNamesForDestination(destinationID: string) {
        const groupIDs = destinationGroupIDsByDestination[destinationID] ?? [];
        return groupIDs
            .map((groupID) => destinationGroups.find((group) => group.id === groupID)?.name)
            .filter((name): name is string => Boolean(name));
    }

    function renderDestinationGroupMetadata(destination: DestinationInput) {
        const groupNames = destinationGroupNamesForDestination(destination.id);
        if (groupNames.length === 0) {
            return null;
        }

        return (
            <span className="destination-group-metadata" title={groupNames.join(', ')}>
                <Layers aria-hidden="true" />
                <span>{destinationGroupDisplayLabel(groupNames)}</span>
            </span>
        );
    }

    const destinationPanelSummary =
        configuredDestinationCount === 0
            ? 'Add destinations first.'
            : homeTargetMode === 'group'
                ? destinationGroupSummary(selectedGroupName, selectedGroupDestinationCount)
                : `${activeDestinationCount} of ${enabledDestinations.length} selected`;
    const sortedLogs = newestFirstLogs(logs);
    const filteredLogs = sortedLogs.filter((entry) => {
        const matchesCategory = logCategoryFilter === 'all' || logCategoryForAction(entry.action) === logCategoryFilter;
        return matchesCategory && logMatchesStatusFilter(entry, logStatusFilter);
    });
    const recentActivity = filteredLogs.slice(0, 4);
    const sidebarItems: SidebarItem[] = [
        { key: 'home', label: 'Home', icon: Home },
        { key: 'destinations', label: 'Destinations', icon: Radio },
        { key: 'profiles', label: 'Profiles', icon: ListChecks },
        ...(SHOW_DEBUG_WORKFLOW_UI ? [{ key: 'logs' as const, label: 'Logs', icon: Activity }] : []),
        { key: 'settings', label: 'Settings', icon: Settings, badge: pendingLiveNowSessions.length },
    ];

    function renderHomeDestinationPicker() {
        return (
            <>
                <div className="segmented-control compact-segmented-control" role="group" aria-label="Destination selection mode">
                    <button type="button" className={homeTargetMode === 'destinations' ? 'active' : ''} onClick={() => setHomeTargetMode('destinations')}>Individual</button>
                    <button type="button" className={homeTargetMode === 'group' ? 'active' : ''} onClick={() => setHomeTargetMode('group')}>Group</button>
                </div>
                {homeTargetMode === 'group' ? (
                    <>
                        <label className="field">
                            <span>Destination Group</span>
                            <select aria-label="Destination Group" value={selectedDestinationGroupID} onChange={(event) => void selectDestinationGroup(event.target.value)}>
                                {destinationGroups.map((group) => <option key={group.id} value={group.id}>{group.name}</option>)}
                            </select>
                        </label>
                        <div className="destination-picker-list">
                            {destinations.length === 0 ? (
                                <div className="empty-state compact-empty-state"><h3>No destinations yet</h3><p>Create one in Destinations to enable previews and publishing.</p></div>
                            ) : selectedGroupDestinationCount === 0 ? (
                                <div className="empty-state compact-empty-state"><h3>No enabled destinations</h3><p>Add enabled destinations to this group in Destinations.</p></div>
                            ) : groupDestinations.map((destination) => (
                                <section key={destination.id} className="destination-picker-card readonly selected">
                                    <PlatformIcon platform={destination.platform} />
                                    <span><strong>{destination.name}</strong><small>{destinationSendToMetadata(destination)}</small></span>
                                </section>
                            ))}
                        </div>
                    </>
                ) : (
                    <>
                        <div className="destination-target-actions">
                            <button type="button" className="ghost-button small-button" onClick={selectAllDestinations}>Select All</button>
                            <button type="button" className="ghost-button small-button" onClick={clearSelectedDestinations}>Clear</button>
                        </div>
                        <div className="destination-picker-list" role="group" aria-label="Session Destinations">
                            {destinations.length === 0 ? (
                                <div className="empty-state compact-empty-state"><h3>No destinations yet</h3><p>Create one in Destinations to enable previews and publishing.</p></div>
                            ) : enabledDestinations.length === 0 ? (
                                <div className="empty-state compact-empty-state"><h3>No enabled destinations</h3><p>Enable at least one destination in Destinations.</p></div>
                            ) : enabledDestinations.map((destination) => {
                                const selected = selectedDestinationIDs.includes(destination.id);
                                return (
                                    <label key={destination.id} className={selected ? 'destination-picker-card selected' : 'destination-picker-card'}>
                                        <input type="checkbox" aria-label={destination.name} checked={selected} onChange={() => toggleSessionDestination(destination.id)} />
                                        <PlatformIcon platform={destination.platform} />
                                        <span><strong>{destination.name}</strong><small>{destinationSendToMetadata(destination)}</small></span>
                                    </label>
                                );
                            })}
                        </div>
                    </>
                )}
            </>
        );
    }

    const previewDestinationFor = (item: PreviewItem) => destinations.find((destination) => destination.id === item.destinationID);

    const previewImageSource = (dataURL: string, url: string) => dataURL.trim() || url.trim();

    const renderLinkPreviewCard = (item: PreviewItem, label: string) => {
        if (!item.linkPreview) {
            return null;
        }

        return (
            <section className="platform-preview platform-preview-card" aria-label={`${label} link preview`}>
                <strong>{label}</strong>
                <div className={item.linkPreview.imageURL ? 'platform-link-card has-image' : 'platform-link-card'}>
                    {item.linkPreview.imageURL ? <img src={item.linkPreview.imageURL} alt="" /> : null}
                    <div>
                        <span>{item.linkPreview.title || item.linkPreview.url}</span>
                        {item.linkPreview.description ? <p>{item.linkPreview.description}</p> : null}
                        <small>{item.linkPreview.siteName || item.linkPreview.url}</small>
                    </div>
                </div>
            </section>
        );
    };

    const renderPlatformPreview = (item: PreviewItem) => {
        const destination = previewDestinationFor(item);
        if (!destination) {
            return null;
        }

        const form = toDestinationFormState(destination);
        const platform = item.platform.toLowerCase();
        const announcementImage = previewImageSource(announcement.announcementImageDataURL, announcement.announcementImageURL);
        const blueskyHomeCardImage = previewImageSource(announcement.blueskyCardThumbnailDataURL, announcement.blueskyCardThumbnailURL);
        const discordImage = platform === 'discord' ? (announcementImage || previewImageSource(form.discordCardThumbnailDataURL, form.discordCardThumbnailURL)) : '';
        const blueskyCardImage = platform === 'bluesky' ? (blueskyHomeCardImage || previewImageSource(form.blueskyCardThumbnailDataURL, form.blueskyCardThumbnailURL)) : '';
        const blueskyAttachedImage = platform === 'bluesky' ? (announcementImage || previewImageSource(form.blueskyAdditionalImageDataURL, form.blueskyAdditionalImageURL)) : '';
        const mastodonImage = platform === 'mastodon' ? (announcementImage || previewImageSource(form.mastodonAdditionalImageDataURL, form.mastodonAdditionalImageURL)) : '';

        if (platform === 'discord' && item.linkPreview) {
            return (
                <div className="platform-preview-stack">
                    {renderLinkPreviewCard(item, 'Approximate Discord Link Preview')}
                    {discordImage ? (
                        <section className="platform-preview platform-preview-discord" aria-label="Configured Discord image embed preview">
                            <strong>Configured Image Embed</strong>
                            <img src={discordImage} alt="" />
                        </section>
                    ) : null}
                </div>
            );
        }

        if (platform === 'discord' && discordImage) {
            return (
                <section className="platform-preview platform-preview-discord" aria-label="Approximate Discord preview">
                    <strong>Approximate Preview</strong>
                    <img src={discordImage} alt="" />
                </section>
            );
        }

        if (platform === 'bluesky' && announcement.streamURL.trim()) {
            return (
                <section className="platform-preview platform-preview-card" aria-label="Approximate Bluesky preview">
                    <strong>Approximate Preview</strong>
                    <div className={blueskyCardImage ? 'platform-link-card has-image' : 'platform-link-card'}>
                        {blueskyCardImage ? <img src={blueskyCardImage} alt="" /> : null}
                        <div>
                            <span>{announcement.streamTitle.trim() || 'Stream link'}</span>
                            <p>{announcement.message.trim() || announcement.streamURL}</p>
                            <small>{announcement.streamURL}</small>
                        </div>
                    </div>
                </section>
            );
        }

        if (platform === 'bluesky' && blueskyAttachedImage) {
            return (
                <section className="platform-preview" aria-label="Approximate Bluesky image preview">
                    <strong>Approximate Preview</strong>
                    <img src={blueskyAttachedImage} alt="" />
                </section>
            );
        }

        if (platform === 'mastodon' && item.linkPreview) {
            return (
                <div className="platform-preview-stack">
                    {renderLinkPreviewCard(item, 'Approximate Mastodon Link Preview')}
                    {mastodonImage ? (
                        <section className="platform-preview" aria-label="Configured Mastodon image attachment preview">
                            <strong>Configured Image Attachment</strong>
                            <img src={mastodonImage} alt="" />
                        </section>
                    ) : null}
                </div>
            );
        }

        if (platform === 'mastodon' && mastodonImage) {
            return (
                <section className="platform-preview" aria-label="Approximate Mastodon image preview">
                    <strong>Approximate Preview</strong>
                    <img src={mastodonImage} alt="" />
                </section>
            );
        }

        return null;
    };

    const renderPreviewItem = (item: PreviewItem) => (
        <details key={`${item.destinationID}-${item.previewLabel || 'preview'}`} className="preview-card preview-disclosure">
            <summary className="preview-summary">
                <span className="entity-row">
                    <PlatformIcon platform={item.platform} />
                    <span>
                        <strong>{item.destinationName}</strong>
                        <small>{item.previewLabel || 'Preview'} · {platformLabel(item.platform)}</small>
                    </span>
                </span>
                <span className="preview-summary-meta">
                    <span>{item.characterCount} chars</span>
                    {item.validationNotes.length > 0 ? <span>{item.validationNotes.length} notes</span> : null}
                    {item.validationState === 'WARNING' ? <TriangleAlert className="preview-warning-icon" aria-label="Template field empty" /> : null}
                    <StatusBadge status={item.validationState} label={item.validationState} />
                </span>
            </summary>
            <div className="preview-body">
                <pre className="preview-content">{item.content || '(empty content)'}</pre>
                {renderPlatformPreview(item)}
                {item.validationNotes.length > 0 ? <ul className={item.validationState === 'WARNING' ? 'note-list warning-note-list' : 'note-list'}>{item.validationNotes.map((note) => <li key={note}>{note}</li>)}</ul> : <p className="success-copy">Ready from a validation standpoint.</p>}
            </div>
        </details>
    );

    const renderExecutionPanel = () => {
        if (!shouldShowExecutionPanel(executionSummary, executionError)) {
            return null;
        }

        const detailsOpen = shouldOpenExecutionDetails(executionSummary);
        const statusClass = executionError ? 'publish-status-card status-error' : executionSummary?.status === 'SUCCESS' ? 'publish-status-card status-success' : executionSummary ? 'publish-status-card status-warning' : 'publish-status-card';
        const pendingDuplicateConfirmation = Boolean(executionSummary?.requiresDuplicateConfirmation);

        return (
            <article className="panel dashboard-panel">
                <div className="panel-header compact-header">
                    <div>
                        <h2>Publish Status</h2>
                    </div>
                    <StatusBadge status={executionSummary?.status ?? 'idle'} label={executionSummary?.status ?? 'Idle'} />
                </div>
                {executionError ? <p className="error-banner">{executionError}</p> : null}
                <section className={statusClass}>
                    <div>
                        <strong>{executionSummary ? executionModeLabel(executionSummary.mode) : 'Ready'}</strong>
                        <p>{executionError ?? publishStatusMessage(executionSummary)}</p>
                    </div>
                </section>
                {pendingDuplicateConfirmation ? (
                    <div className="warning-banner stacked-callout">
                        {executionSummary?.duplicateWarningMessage ? <p>{executionSummary.duplicateWarningMessage}</p> : null}
                        <button type="button" className="primary-button gold-button" onClick={() => void onConfirmDuplicateGoLive()} disabled={executionLoading !== null}>
                            {executionLoading === 'go_live' ? 'Confirming...' : 'Confirm Anyway'}
                        </button>
                    </div>
                ) : null}
                {!executionSummary || executionSummary.results.length === 0 ? (
                    <div className="empty-state compact-empty-state">
                        <h3>No publish results yet</h3>
                        <p>Run Go Live or End Stream to see destination status.</p>
                    </div>
                ) : (
                    <details className="result-details" open={detailsOpen}>
                        <summary>
                            <span>Details</span>
                            <small>{executionSummary.results.length} destination{executionSummary.results.length === 1 ? '' : 's'}</small>
                        </summary>
                        <div className="compact-list">
                            {executionSummary.results.map((result) => (
                                <section className="activity-card" key={`${result.destinationID}-${result.state}-${result.message}`}>
                                    <PlatformIcon platform={result.platform} />
                                    <div>
                                        <strong>{result.destinationName}</strong>
                                        <p className="log-context">{executionResultKind(executionSummary.mode, result.message)} · {result.platform}</p><p>{result.message}</p>
                                        {result.content ? <p className="log-content-preview">{result.content}</p> : null}
                                    </div>
                                    <StatusBadge status={result.state} />
                                </section>
                            ))}
                        </div>
                    </details>
                )}
            </article>
        );
    };

    const renderAnnouncementProfileSelector = () => (
        <section className="mini-panel profile-selector-panel">
            <div className="profile-selector-row">
                <label className="field">
                    <span>Announcement Preset</span>
                    <select aria-label="Announcement Preset" value={selectedAnnouncementProfileID} onChange={(event) => void selectAnnouncementProfile(event.target.value)}>
                        <option value="">Manual / No Preset</option>
                        {announcementProfiles.map((profile) => <option key={profile.id} value={profile.id}>{profile.name}</option>)}
                    </select>
                </label>
                <div className="profile-action-buttons">
                    <span className="button-tooltip-wrap">
                        <button type="button" className="ghost-button icon-button profile-save-button" aria-label="Save as New Profile" onClick={openSaveCurrentAnnouncementProfile} disabled={homeProfileSaving}>
                            <FilePlus aria-hidden="true" />
                        </button>
                        <span className="button-tooltip" role="tooltip">Save current announcement as a new profile</span>
                    </span>
                    {activeAnnouncementProfile ? (
                        <span className="button-tooltip-wrap">
                            <button type="button" className="ghost-button icon-button profile-save-button" aria-label="Update Selected Profile" onClick={() => void onUpdateCurrentAnnouncementProfile()} disabled={homeProfileSaving}>
                                <Save aria-hidden="true" />
                            </button>
                            <span className="button-tooltip" role="tooltip">Update selected profile with current announcement</span>
                        </span>
                    ) : null}
                </div>
            </div>
            {activeAnnouncementProfile ? <small>{activeAnnouncementProfile.description || 'Preset applied. Review fields before publishing.'}</small> : null}
            {pendingProfileID ? (
                <div className="warning-banner stacked-callout compact-callout">
                    <p>Applying a new preset will replace current announcement values.</p>
                    <div className="quick-action-row">
                        <button type="button" className="primary-button" onClick={() => void applyAnnouncementProfile(pendingProfileID)}>Apply Preset</button>
                        <button type="button" className="ghost-button" onClick={() => setPendingProfileID(null)}>Cancel</button>
                    </div>
                </div>
            ) : null}
        </section>
    );

    const renderHomeMediaPanel = () => {
        if (!activeDestinationsSupportMedia) {
            return null;
        }
        return (
            <details className="config-section media-section">
                <summary className="config-section-title">
                    <span><FileText aria-hidden="true" /> Media</span>
                    <small>{activeDestinationsIncludeBluesky ? 'Announcement image and Bluesky card thumbnail' : 'Announcement image'}</small>
                </summary>
                <div className="config-section-body">
                    <div className="form-grid">
                        <label className="field">
                            <span>Announcement Image URL</span>
                            <input
                                aria-label="Announcement Image URL"
                                value={announcement.announcementImageURL}
                                onChange={(event) => {
                                    updateAnnouncement('announcementImageURL', event.target.value);
                                    if (event.target.value.trim() !== '') clearAnnouncementImageUpload();
                                }}
                                placeholder="https://example.com/promo.png"
                            />
                        </label>
                        <div className="field">
                            <div className="field-label-row">
                                <label htmlFor="announcement-image-upload">Announcement Image Upload</label>
                                <button type="button" className="info-tooltip" aria-label="Announcement Image Upload information">
                                    i<span role="tooltip">{announcement.announcementImageDataURL ? 'Uploaded image selected.' : 'Used by selected providers that support attached images.'}</span>
                                </button>
                            </div>
                            <input id="announcement-image-upload" ref={announcementImageInputRef} aria-label="Announcement Image Upload" type="file" accept="image/*" onChange={(event) => void onAnnouncementImageUpload(event.target.files?.[0])} />
                            {announcement.announcementImageDataURL ? <span className="upload-selection-indicator">Image upload selected</span> : null}
                        </div>
                        {activeDestinationsIncludeBluesky ? (
                            <>
                                <label className="field">
                                    <span>Bluesky Link Card Thumbnail URL</span>
                                    <input
                                        aria-label="Bluesky Link Card Thumbnail URL"
                                        value={announcement.blueskyCardThumbnailURL}
                                        onChange={(event) => {
                                            updateAnnouncement('blueskyCardThumbnailURL', event.target.value);
                                            if (event.target.value.trim() !== '') clearHomeBlueskyThumbnailUpload();
                                        }}
                                        placeholder="https://example.com/card.png"
                                    />
                                </label>
                                <div className="field">
                                    <div className="field-label-row">
                                        <label htmlFor="home-bluesky-card-thumbnail-upload">Bluesky Link Card Thumbnail Upload</label>
                                        <button type="button" className="info-tooltip" aria-label="Bluesky Link Card Thumbnail Upload information">
                                            i<span role="tooltip">{announcement.blueskyCardThumbnailDataURL ? 'Uploaded thumbnail selected.' : 'Used only for Bluesky link cards.'}</span>
                                        </button>
                                    </div>
                                    <input id="home-bluesky-card-thumbnail-upload" ref={homeBlueskyThumbnailInputRef} aria-label="Bluesky Link Card Thumbnail Upload" type="file" accept="image/*" onChange={(event) => void onHomeBlueskyThumbnailUpload(event.target.files?.[0])} />
                                    {announcement.blueskyCardThumbnailDataURL ? <span className="upload-selection-indicator">Thumbnail upload selected</span> : null}
                                </div>
                            </>
                        ) : null}
                    </div>
                </div>
            </details>
        );
    };

    const renderAnnouncementProfilesPanel = () => (
        <article className="panel settings-panel announcement-profiles-panel">
            <div className="panel-header compact-header">
                <div>
                    <h2>Announcement Profiles</h2>
                    <p>Manage prefill presets for stream types.</p>
                </div>
                <button type="button" className="ghost-button" onClick={resetProfileForm}><Send aria-hidden="true" />New Preset</button>
            </div>
            {profileError ? <p className="error-banner">{profileError}</p> : null}
            {profileStatus ? <p className="success-banner">{profileStatus}</p> : null}
            <div className="compact-list profile-list">
                {announcementProfiles.length === 0 ? (
                    <div className="empty-state compact-empty-state"><h3>No presets yet</h3><p>Create prefill presets for the stream types you repeat.</p></div>
                ) : announcementProfiles.map((profile) => {
                    const group = destinationGroups.find((item) => item.id === profile.defaultGroupID);
                    return (
                        <section className={profileForm.id === profile.id ? 'activity-card active' : 'activity-card'} key={profile.id}>
                            <ListChecks aria-hidden="true" />
                            <div>
                                <strong>{profile.name}</strong>
                                <p>{profile.description || 'No description'}</p>
                                <small>{group ? `Group: ${group.name}` : 'Manual destination selection'}</small>
                            </div>
                            <div className="quick-action-row">
                                <button type="button" className="ghost-button small-button" onClick={() => editAnnouncementProfile(profile)}>Edit</button>
                                <button type="button" className="ghost-button icon-button" aria-label="Duplicate Profile" onClick={() => void onDuplicateAnnouncementProfile(profile)}><Copy aria-hidden="true" /></button>
                                <button type="button" className="danger-button icon-button" aria-label="Delete Profile" onClick={() => void onDeleteAnnouncementProfile(profile)}><Trash2 aria-hidden="true" /></button>
                            </div>
                        </section>
                    );
                })}
            </div>
            <form className="announcement-form compact-form" onSubmit={onAnnouncementProfileSubmit}>
                <div className="form-grid">
                    <label className="field"><span>Preset Name</span><input aria-label="Preset Name" value={profileForm.name} onChange={(event) => updateProfileForm('name', event.target.value)} placeholder="Music Stream" maxLength={80} /></label>
                    <label className="field"><span>Description</span><input aria-label="Preset Description" value={profileForm.description} onChange={(event) => updateProfileForm('description', event.target.value)} placeholder="Music announcements and hashtags" maxLength={250} /></label>
                    <label className="field"><span>Stream Title</span><input aria-label="Profile Stream Title" value={profileForm.defaultStreamTitle} onChange={(event) => updateProfileForm('defaultStreamTitle', event.target.value)} placeholder="Late Night Variety" /></label>
                    <label className="field"><span>Stream URL</span><input aria-label="Profile Stream URL" value={profileForm.defaultStreamURL} onChange={(event) => updateProfileForm('defaultStreamURL', event.target.value)} placeholder="https://example.com/live" /></label>
                    <label className="field"><span>Category / Game</span><input aria-label="Profile Category" value={profileForm.defaultCategory} onChange={(event) => updateProfileForm('defaultCategory', event.target.value)} placeholder="Music, FFXIV, Variety..." /></label>
                    <label className="field"><span>Hashtags</span><input aria-label="Profile Hashtags" value={profileForm.defaultHashtags} onChange={(event) => updateProfileForm('defaultHashtags', event.target.value)} placeholder="#VTuber #Music" /></label>
                    <label className="field"><span>Destination Group</span><select aria-label="Profile Destination Group" value={profileForm.defaultGroupID} onChange={(event) => updateProfileForm('defaultGroupID', event.target.value)}><option value="">Manual / No Group</option>{destinationGroups.map((group) => <option key={group.id} value={group.id}>{group.name}</option>)}</select></label>
                </div>
                <label className="field"><span>Message</span><textarea aria-label="Profile Message" value={profileForm.defaultMessage} onChange={(event) => updateProfileForm('defaultMessage', event.target.value)} rows={3} placeholder="What are we getting into?" /></label>
                <details className="config-section media-section">
                    <summary className="config-section-title"><span><FileText aria-hidden="true" /> Preset Media</span><small>Optional image defaults</small></summary>
                    <div className="config-section-body">
                        <div className="form-grid">
                            <label className="field"><span>Announcement Image URL</span><input aria-label="Profile Announcement Image URL" value={profileForm.defaultAnnouncementImageURL} onChange={(event) => { updateProfileForm('defaultAnnouncementImageURL', event.target.value); if (event.target.value.trim() !== '') updateProfileForm('defaultAnnouncementImageDataURL', ''); }} placeholder="https://example.com/promo.png" /></label>
                            <div className="field">
                                <div className="field-label-row">
                                    <label htmlFor="profile-announcement-image-upload">Announcement Image Upload</label>
                                    <button type="button" className="info-tooltip" aria-label="Profile Announcement Image Upload information">
                                        i<span role="tooltip">{profileForm.defaultAnnouncementImageDataURL ? 'Uploaded image selected.' : 'Used as the default attached announcement image.'}</span>
                                    </button>
                                </div>
                                <input id="profile-announcement-image-upload" ref={announcementProfileImageInputRef} aria-label="Profile Announcement Image Upload" type="file" accept="image/*" onChange={(event) => void onProfileImageUpload(event.target.files?.[0], 'announcement')} />
                                {profileForm.defaultAnnouncementImageDataURL ? <span className="upload-selection-indicator">Image upload selected</span> : null}
                            </div>
                            <label className="field"><span>Bluesky Link Card Thumbnail URL</span><input aria-label="Profile Bluesky Link Card Thumbnail URL" value={profileForm.defaultBlueskyCardThumbnailURL} onChange={(event) => { updateProfileForm('defaultBlueskyCardThumbnailURL', event.target.value); if (event.target.value.trim() !== '') updateProfileForm('defaultBlueskyCardThumbnailDataURL', ''); }} placeholder="https://example.com/card.png" /></label>
                            <div className="field">
                                <div className="field-label-row">
                                    <label htmlFor="profile-bluesky-card-thumbnail-upload">Bluesky Link Card Thumbnail Upload</label>
                                    <button type="button" className="info-tooltip" aria-label="Profile Bluesky Link Card Thumbnail Upload information">
                                        i<span role="tooltip">{profileForm.defaultBlueskyCardThumbnailDataURL ? 'Uploaded thumbnail selected.' : 'Used only for Bluesky link cards.'}</span>
                                    </button>
                                </div>
                                <input id="profile-bluesky-card-thumbnail-upload" ref={announcementProfileBlueskyThumbnailInputRef} aria-label="Profile Bluesky Link Card Thumbnail Upload" type="file" accept="image/*" onChange={(event) => void onProfileImageUpload(event.target.files?.[0], 'bluesky-card')} />
                                {profileForm.defaultBlueskyCardThumbnailDataURL ? <span className="upload-selection-indicator">Thumbnail upload selected</span> : null}
                            </div>
                        </div>
                    </div>
                </details>
                <div className="action-cluster-main">
                    <button type="submit" className="primary-button">{profileForm.id ? 'Save Preset' : 'Create Preset'}</button>
                </div>
            </form>
        </article>
    );

    const renderLiveSessionPanel = () => (
        <article className="panel dashboard-panel session-editor">
            <div className="panel-header compact-header">
                <div>
                    <h2>Live Session</h2>
                    <p>Configure the announcement once, then preview and publish from the same workspace.</p>
                </div>
                <button
                    type="button"
                    className="ghost-button icon-button"
                    aria-label="Reset"
                    onClick={() => {
                        streamURLTouched.current = false;
                        hashtagsTouched.current = false;
                        setSelectedAnnouncementProfileID('');
                        setProfileDirty(false);
                        setPendingProfileID(null);
                        setAnnouncement(applyAnnouncementDefaults(initialAnnouncement, settings, true, true));
                        if (announcementImageInputRef.current) {
                            announcementImageInputRef.current.value = '';
                        }
                        if (homeBlueskyThumbnailInputRef.current) {
                            homeBlueskyThumbnailInputRef.current.value = '';
                        }
                        setPreviewError(null);
                        setPreviewItems([]);
                    }}
                >
                    <RotateCcw aria-hidden="true" />
                </button>
            </div>
            <form className="announcement-form compact-form" onSubmit={onPreviewSubmit}>
                {renderAnnouncementProfileSelector()}
                <div className="form-grid">
                    <div className="field">
                        <div className="field-label-row">
                            <label htmlFor="stream-title">Stream Title</label>
                            <button type="button" className="info-tooltip" aria-label="Stream Title information">
                                i<span role="tooltip">Required for Bluesky Live Now Use</span>
                            </button>
                        </div>
                        <input
                            id="stream-title"
                            aria-label="Stream Title"
                            value={announcement.streamTitle}
                            onChange={(event) => updateAnnouncement('streamTitle', event.target.value)}
                            placeholder="Late Night Variety"
                        />
                    </div>
                    <label className="field">
                        <span>Stream URL</span>
                        <input aria-label="Stream URL" value={announcement.streamURL} onChange={(event) => updateAnnouncement('streamURL', event.target.value)} placeholder="https://example.com/live" />
                    </label>
                    <label className="field">
                        <span>Category / Game</span>
                        <input aria-label="Category / Game" value={announcement.category} onChange={(event) => updateAnnouncement('category', event.target.value)} placeholder="Music, FFXIV, Variety..." />
                    </label>
                    <label className="field">
                        <span>Hashtags</span>
                        <input aria-label="Hashtags" value={announcement.hashtags} onChange={(event) => updateAnnouncement('hashtags', event.target.value)} placeholder="#vtuber #music" />
                    </label>
                </div>
                <label className="field">
                    <span>Optional Message</span>
                    <textarea aria-label="Optional Message" value={announcement.message} onChange={(event) => updateAnnouncement('message', event.target.value)} placeholder="What are we getting into tonight?" rows={3} />
                </label>
                {renderHomeMediaPanel()}
                <div className="workflow-actions command-actions">
                    <button type="submit" className="ghost-button" disabled={previewLoading}>
                        <ClipboardList aria-hidden="true" />
                        {previewLoading ? 'Generating...' : 'Generate Preview'}
                    </button>
                    <button type="button" className="primary-button gold-button" onClick={() => void runGoLive()} disabled={executionLoading !== null}>
                        <Zap aria-hidden="true" />
                        {executionLoading === 'go_live' ? 'Running Go Live...' : 'Go Live'}
                    </button>
                    <span className="button-tooltip-wrap">
                        <button type="button" className="ghost-button" onClick={() => void onEndStream()} disabled={executionLoading !== null} aria-describedby="end-stream-action-hint">
                            <Square aria-hidden="true" />
                            {executionLoading === 'end_stream' ? 'Ending Stream...' : 'End Stream'}
                        </button>
                        <span id="end-stream-action-hint" className="button-tooltip" role="tooltip">
                            {activeDestinationCount > 0 ? endStreamActionHint(selectedEndStreamPostCount, selectedLiveNowClearCount) : homeTargetMode === 'group' ? 'Choose a group with enabled destinations to use End Stream.' : 'Select at least one destination to use End Stream.'}
                        </span>
                    </span>
                </div>
            </form>
        </article>
    );

    const renderPreviewPanel = () => (
        <article className="panel dashboard-panel preview-panel">
            <div className="panel-header compact-header">
                <div>
                    <h2>Preview Content</h2>
                    <p>Validate destination-specific messages before publishing.</p>
                </div>
                <span>{previewItems.length} messages</span>
            </div>
            {previewError ? <p className="error-banner">{previewError}</p> : null}
            {previewItems.length === 0 ? (
                <div className="empty-state compact-empty-state">
                    <h3>No previews yet</h3>
                    <p>{destinations.length === 0 ? 'Add at least one destination first, then generate a preview to see per-destination content.' : activeDestinationCount === 0 ? 'Select destinations or choose a group, then generate a preview.' : 'Generate a preview to validate messages before publishing.'}</p>
                </div>
            ) : (
                <div className="preview-list dense-preview-list">{previewItems.map(renderPreviewItem)}</div>
            )}
        </article>
    );

    const renderRecoveryStatusCard = () => (
        <section className={pendingLiveNowSessions.length > 0 ? 'panel recovery-card recovery-pending' : 'panel recovery-card'}>
            <ShieldAlert aria-hidden="true" />
            <div>
                <strong>Recovery Status</strong>
                <p>{pendingLiveNowSessions.length > 0 ? `${pendingLiveNowSessions.length} pending Live Now recovery item${pendingLiveNowSessions.length === 1 ? '' : 's'}.` : 'No pending Live Now recovery.'}</p>
            </div>
            <button type="button" className="ghost-button small-button" onClick={() => setSelectedTab('settings')}>Open Settings</button>
        </section>
    );

    const renderHomeSideStack = () => (
        <aside className="dashboard-side-stack">
            <section className="panel dashboard-panel">
                <div className="panel-header compact-header">
                    <div>
                        <h2>Destinations</h2>
                        <p>{destinationPanelSummary}</p>
                    </div>
                    <button type="button" className="ghost-button small-button" onClick={() => setSelectedTab('destinations')}>Manage</button>
                </div>
                {renderHomeDestinationPicker()}
            </section>
            {renderRecoveryStatusCard()}
        </aside>
    );

    const renderRecentActivityPanel = () => {
        if (!SHOW_DEBUG_WORKFLOW_UI) {
            return null;
        }

        return (
            <article className="panel dashboard-panel">
                <div className="panel-header compact-header">
                    <div>
                        <h2>Recent Activity</h2>
                    </div>
                    <button type="button" className="ghost-button small-button" onClick={() => setSelectedTab('logs')}>View Logs</button>
                </div>
                {recentActivity.length === 0 ? (
                    <div className="empty-state compact-empty-state">
                        <h3>No recent activity</h3>
                        <p>Preview, publish, recovery, and settings events will appear here.</p>
                    </div>
                ) : (
                    <div className="compact-list">
                        {recentActivity.map((entry, index) => {
                            const platform = inferLogPlatform(entry, destinations);
                            return (
                                <section className="activity-card" key={`${entry.timestamp}-${entry.action}-${index}`}>
                                    <PlatformIcon platform={platform} />
                                    <div>
                                        <strong>{logActionLabel(entry.action)}</strong>
                                        <p>{entry.destination} · {entry.message}</p>
                                        <time dateTime={entry.timestamp}>{entry.timestamp}</time>
                                    </div>
                                    <StatusBadge status={entry.status} />
                                </section>
                            );
                        })}
                    </div>
                )}
            </article>
        );
    };

    const renderHomeDashboard = () => (
        <section className="content-stack home-dashboard">
            <div className="dashboard-top-grid">
                {renderLiveSessionPanel()}
                {renderPreviewPanel()}
                {renderHomeSideStack()}
            </div>
            {renderExecutionPanel()}
            {renderRecentActivityPanel()}
        </section>
    );

    const renderPlatformConnectionFields = () => (
        <>
            {destinationForm.platform === 'discord' ? (
                <div className="form-grid">
                    <label className="field">
                        <span>Server Name</span>
                        <input aria-label="Server Name" value={destinationForm.discordServerName} onChange={(event) => updateDestination('discordServerName', event.target.value)} placeholder="My Discord Server" />
                    </label>
                    <label className="field">
                        <span>Channel Name</span>
                        <input aria-label="Channel Name" value={destinationForm.discordChannelName} onChange={(event) => updateDestination('discordChannelName', event.target.value)} placeholder="go-live" />
                    </label>
                    <label className="field panel-wide">
                        <span>Webhook URL</span>
                        <input aria-label="Webhook URL" value={destinationForm.discordWebhookKey} onChange={(event) => updateDestination('discordWebhookKey', event.target.value)} placeholder="https://discord.com/api/webhooks/..." />
                    </label>
                </div>
            ) : null}
            {destinationForm.platform === 'bluesky' ? (
                <div className="form-grid">
                    <label className="field">
                        <span>Account Identifier</span>
                        <input aria-label="Account Identifier" value={destinationForm.blueskyAccountIdentifier} onChange={(event) => updateDestination('blueskyAccountIdentifier', event.target.value)} placeholder="handle.bsky.social" />
                    </label>
                    <label className="field">
                        <span>App Password</span>
                        <input aria-label="App Password" value={destinationForm.blueskyCredentialKey} onChange={(event) => updateDestination('blueskyCredentialKey', event.target.value)} placeholder="Paste Bluesky app password" />
                    </label>
                </div>
            ) : null}
            {destinationForm.platform === 'mastodon' ? (
                <div className="form-grid">
                    <label className="field">
                        <span>Account Identifier</span>
                        <input aria-label="Mastodon Account Identifier" value={destinationForm.mastodonAccountIdentifier} onChange={(event) => updateDestination('mastodonAccountIdentifier', event.target.value)} placeholder="@streamer@mastodon.social" />
                    </label>
                    <label className="field">
                        <span>Instance URL</span>
                        <input aria-label="Instance URL" value={destinationForm.mastodonInstanceURL} onChange={(event) => updateDestination('mastodonInstanceURL', event.target.value)} placeholder="https://mastodon.social" />
                    </label>
                    <label className="field">
                        <span>Access Token</span>
                        <input aria-label="Access Token" value={destinationForm.mastodonCredentialKey} onChange={(event) => updateDestination('mastodonCredentialKey', event.target.value)} placeholder="Paste Mastodon access token" />
                    </label>
                </div>
            ) : null}
        </>
    );

    const renderAdvancedFields = () => (
        <details className="config-section" data-testid="destination-advanced-section">
            <summary>
                <span><Wrench aria-hidden="true" /> Advanced</span>
                <small>Images, Live Now, and end stream options</small>
            </summary>
            <div className="config-section-body">
                <section className="mini-panel template-helper-panel">
                    <div className="template-helper-header">
                        <h3>End Stream Message</h3>
                        <label className="checkbox-field">
                            <input type="checkbox" aria-label="End Stream Enabled" checked={destinationForm.endStreamEnabled} onChange={(event) => updateDestination('endStreamEnabled', event.target.checked)} />
                            <span>Post when ending stream</span>
                        </label>
                    </div>
                    <label className="field">
                        <span>End Stream Template</span>
                        <textarea ref={endStreamTemplateInputRef} aria-label="End Stream Template" value={destinationForm.endStreamTemplate} onFocus={() => { activeTemplateInputRef.current = endStreamTemplateInputRef.current; }} onChange={(event) => updateDestination('endStreamTemplate', event.target.value)} rows={3} />
                        <small>{destinationForm.endStreamEnabled ? 'This message will be posted when you end a stream.' : 'Saved for later. Enable "Post when ending stream" to send it during End Stream.'}</small>
                    </label>
                </section>
                {destinationForm.platform === 'bluesky' ? (
                    <section className="mini-panel template-helper-panel">
                        <div className="template-helper-header">
                            <h3>Bluesky Live Now</h3>
                            <label className="checkbox-field">
                                <input type="checkbox" aria-label="Bluesky Live Status Enabled" checked={destinationForm.blueskyLiveStatusEnabled} onChange={(event) => updateDestination('blueskyLiveStatusEnabled', event.target.checked)} />
                                <span>Set Live Now status on Go Live</span>
                            </label>
                        </div>
                        <div className="form-grid">
                            <label className="field"><span>Live Status Template</span><input aria-label="Live Status Template" value={destinationForm.blueskyLiveStatusTemplate} onChange={(event) => updateDestination('blueskyLiveStatusTemplate', event.target.value)} placeholder="Live now: {{stream_title}}" /></label>
                            <label className="field"><span>Live Now Duration (Minutes)</span><input aria-label="Live Now Duration Minutes" value={destinationForm.blueskyLiveNowDurationMinutes} onChange={(event) => updateDestination('blueskyLiveNowDurationMinutes', event.target.value)} placeholder="120" /></label>
                        </div>
                    </section>
                ) : null}
                {destinationForm.platform === 'discord' ? (
                    <><label className="field"><span>Card Thumbnail URL</span><input aria-label="Discord Card Thumbnail URL" value={destinationForm.discordCardThumbnailURL} onChange={(event) => { updateDestination('discordCardThumbnailURL', event.target.value); if (event.target.value.trim() !== '') clearDiscordThumbnailImage(); }} placeholder="https://static-cdn.jtvnw.net/..." /></label><label className="field"><span>Card Thumbnail Upload</span><input ref={discordThumbnailInputRef} aria-label="Discord Card Thumbnail Upload" type="file" accept="image/*" onChange={(event) => void onDiscordThumbnailUpload(event.target.files?.[0])} /><small>Used when no thumbnail URL is set.</small></label></>
                ) : null}
                {destinationForm.platform === 'bluesky' ? (
                    <><label className="field"><span>Card Thumbnail URL</span><input aria-label="Bluesky Card Thumbnail URL" value={destinationForm.blueskyCardThumbnailURL} onChange={(event) => { updateDestination('blueskyCardThumbnailURL', event.target.value); if (event.target.value.trim() !== '') clearBlueskyThumbnailImage(); }} placeholder="https://static-cdn.jtvnw.net/..." /></label><label className="field"><span>Card Thumbnail Upload</span><input ref={blueskyThumbnailInputRef} aria-label="Bluesky Card Thumbnail Upload" type="file" accept="image/*" onChange={(event) => void onBlueskyThumbnailUpload(event.target.files?.[0])} /><small>Images are resized for Bluesky card limits.</small></label><label className="field"><span>Additional Image URL</span><input aria-label="Bluesky Additional Image URL" value={destinationForm.blueskyAdditionalImageURL} onChange={(event) => { updateDestination('blueskyAdditionalImageURL', event.target.value); if (event.target.value.trim() !== '') clearBlueskyAdditionalImage(); }} placeholder="https://static-cdn.jtvnw.net/..." /></label><label className="field"><span>Additional Image Upload</span><input ref={blueskyAdditionalImageInputRef} aria-label="Bluesky Additional Image Upload" type="file" accept="image/*" onChange={(event) => void onBlueskyAdditionalImageUpload(event.target.files?.[0])} /><small>Optional image attached to the post.</small></label></>
                ) : null}
                {destinationForm.platform === 'mastodon' ? (
                    <><label className="field"><span>Additional Image URL</span><input aria-label="Mastodon Additional Image URL" value={destinationForm.mastodonAdditionalImageURL} onChange={(event) => { updateDestination('mastodonAdditionalImageURL', event.target.value); if (event.target.value.trim() !== '') clearMastodonAdditionalImage(); }} placeholder="https://static-cdn.jtvnw.net/..." /></label><label className="field"><span>Additional Image Upload</span><input ref={mastodonAdditionalImageInputRef} aria-label="Mastodon Additional Image Upload" type="file" accept="image/*" onChange={(event) => void onMastodonAdditionalImageUpload(event.target.files?.[0])} /><small>Large images are resized before posting.</small></label></>
                ) : null}
            </div>
        </details>
    );

    const renderSaveProfileModal = () => saveProfileModalOpen ? (
        <div className="modal-backdrop" role="presentation">
            <section className="modal-panel save-profile-modal" role="dialog" aria-modal="true" aria-labelledby="save-profile-title">
                <div className="panel-header compact-header">
                    <div>
                        <h2 id="save-profile-title">Save as Profile</h2>
                        <p>This profile will save the current announcement values and selected destination group.</p>
                    </div>
                </div>
                <form className="announcement-form compact-form" onSubmit={onSaveCurrentAnnouncementProfile}>
                    <label className="field">
                        <span>Profile Name</span>
                        <input aria-label="Profile Name" value={homeProfileName} onChange={(event) => setHomeProfileName(event.target.value)} maxLength={80} autoFocus />
                    </label>
                    <label className="field">
                        <span>Description</span>
                        <input aria-label="Profile Description" value={homeProfileDescription} onChange={(event) => setHomeProfileDescription(event.target.value)} maxLength={250} />
                    </label>
                    {homeProfileError ? <p className="error-banner compact-callout">{homeProfileError}</p> : null}
                    <div className="quick-action-row modal-actions">
                        <button type="submit" className="primary-button" disabled={homeProfileSaving}>{homeProfileSaving ? 'Saving...' : 'Save Profile'}</button>
                        <button type="button" className="ghost-button" onClick={closeSaveCurrentAnnouncementProfile} disabled={homeProfileSaving}>Cancel</button>
                    </div>
                </form>
            </section>
        </div>
    ) : null;

    const renderDestinationSetupModal = () => destinationSetupModalOpen ? (
        <div className="modal-backdrop" role="presentation">
            <section className="modal-panel setup-modal" role="dialog" aria-modal="true" aria-labelledby="destination-setup-title">
                <div className="panel-header compact-header">
                    <div>
                        <h2 id="destination-setup-title">{platformLabel(destinationForm.platform)} Setup</h2>
                        <p>Use these steps, then test before saving.</p>
                    </div>
                    <button type="button" className="ghost-button icon-button" aria-label="Close Setup" onClick={() => setDestinationSetupModalOpen(false)}>
                        <X aria-hidden="true" />
                    </button>
                </div>
                <div className="guided-setup-list">
                    {guidedSetupSteps(destinationForm.platform).map((step) => (
                        <section className="guided-setup-step" key={step.title}>
                            <strong>{step.title}</strong>
                            <p>{step.detail}</p>
                        </section>
                    ))}
                </div>
                <div className="quick-action-row modal-actions">
                    <button type="button" className="primary-button" onClick={() => setDestinationSetupModalOpen(false)}>Done</button>
                </div>
            </section>
        </div>
    ) : null;

    const renderAboutPanel = () => (
        <article className="panel settings-panel about-card">
            <div className="about-identity">
                <img src={streamSignalIcon} alt="" className="about-logo" aria-hidden="true" />
                <div>
                    <span className="eyebrow">Starsong Tools</span>
                    <div className="about-product-row">
                        <h2>StreamSignal</h2>
                        <span className="about-version">Version {APP_VERSION}</span>
                    </div>
                    <p>Local-first multi-destination stream announcements for creators.</p>
                </div>
            </div>
            <section className="about-legal-section">
                <h3>Legal Notices</h3>
                <p className="about-legal-copy">{LEGAL_NOTICE}</p>
            </section>
        </article>
    );

    return (
        <main className="app-shell">
            <aside className="app-sidebar" aria-label="Primary navigation"><div className="sidebar-brand-stack"><div className="sidebar-brand"><img src={streamSignalIcon} alt="" className="app-logo" aria-hidden="true" /><div><h1>StreamSignal</h1></div></div>{SHOW_DEBUG_WORKFLOW_UI ? <span className="debug-build-badge">Debug Build</span> : null}</div><nav className="sidebar-nav">{sidebarItems.map((item) => { const Icon = item.icon; return <button key={item.key} type="button" className={selectedTab === item.key ? 'sidebar-nav-item active' : 'sidebar-nav-item'} onClick={() => setSelectedTab(item.key)} aria-current={selectedTab === item.key ? 'page' : undefined}><Icon aria-hidden="true" /><span>{item.label}</span>{item.badge ? <strong className="nav-badge">{item.badge}</strong> : null}</button>; })}</nav></aside>
            <div className="app-frame">
                {selectedTab === 'home' ? renderHomeDashboard() : null}
                {selectedTab === 'destinations' ? <section className="content-stack destinations-workspace"><div className="segmented-control workspace-segmented-control" role="group" aria-label="Destination workspace"><button type="button" className={destinationWorkspaceTab === 'destinations' ? 'active' : ''} onClick={() => setDestinationWorkspaceTab('destinations')}>Destinations</button><button type="button" className={destinationWorkspaceTab === 'groups' ? 'active' : ''} onClick={() => setDestinationWorkspaceTab('groups')}>Groups</button></div>{destinationWorkspaceTab === 'groups' ? <article className="panel dashboard-panel"><div className="panel-header compact-header"><div><h2>Destination Groups</h2><p>Organize where a workflow posts.</p></div><button type="button" className="ghost-button" onClick={resetDestinationGroupForm}><Send aria-hidden="true" />New Group</button></div><div className="destination-card-grid">{destinationGroups.map((group) => <section key={group.id} className={destinationGroupForm.id === group.id ? 'destination-manage-card active' : 'destination-manage-card'}><div className="entity-row"><ListChecks aria-hidden="true" /><div><strong>{group.name}</strong><span>{group.description || 'No description'}</span></div></div><div className="quick-action-row"><button type="button" className="ghost-button small-button" onClick={() => void editDestinationGroup(group)}>Edit</button><button type="button" className="ghost-button icon-button" aria-label="Duplicate Group" onClick={() => void onDuplicateDestinationGroup(group)}><Copy aria-hidden="true" /></button><button type="button" className="danger-button icon-button" aria-label="Delete Group" onClick={() => void onDeleteDestinationGroup(group)} disabled={group.id === DEFAULT_DESTINATION_GROUP_ID}><Trash2 aria-hidden="true" /></button></div></section>)}</div><form className="announcement-form" onSubmit={onDestinationGroupSubmit}><div className="form-grid"><label className="field"><span>Group Name</span><input aria-label="Group Name" value={destinationGroupForm.name} onChange={(event) => updateDestinationGroupForm('name', event.target.value)} placeholder="Main Announcements" maxLength={80} /></label><label className="field"><span>Description</span><input aria-label="Group Description" value={destinationGroupForm.description} onChange={(event) => updateDestinationGroupForm('description', event.target.value)} placeholder="Primary live announcement targets" maxLength={250} /></label></div><section className="mini-panel template-helper-panel"><div className="template-helper-header"><h3>Members</h3><span>{destinationIDsForGroupForm.length} selected</span></div><div className="membership-checklist" role="group" aria-label="Group Destinations">{destinations.length === 0 ? <p>No destinations yet.</p> : destinations.map((destination) => <label key={destination.id} className="checkbox-field"><input type="checkbox" aria-label={destination.name} checked={destinationIDsForGroupForm.includes(destination.id)} onChange={() => toggleGroupFormDestination(destination.id)} /><span>{destination.name}</span></label>)}</div></section><div className="action-cluster-main"><button type="submit" className="primary-button">{destinationGroupForm.id ? 'Save Group' : 'Create Group'}</button></div></form></article> : null}{destinationWorkspaceTab === 'destinations' ? <><article className="panel dashboard-panel"><div className="panel-header compact-header"><div><h2>Saved Destinations</h2><p>Manage integrations quickly, or load one into the editor.</p></div><div className="form-actions"><span>{configuredDestinationCount} configured</span><button type="button" className="ghost-button" onClick={() => resetDestinationForm()}><Send aria-hidden="true" />New Destination</button></div></div>{destinationError ? <p className="error-banner">{destinationError}</p> : null}<div className="destination-card-grid">{destinations.length === 0 ? <div className="empty-state"><h3>No destinations yet</h3><p>Create a destination to start building platform-specific previews and go-live targets.</p></div> : destinations.map((item) => <section key={item.id} className={destinationForm.id === item.id ? 'destination-manage-card active' : 'destination-manage-card'}><div className="entity-row"><PlatformIcon platform={item.platform} /><div><strong>{item.name}</strong><span>{destinationManageMetadata(item)}</span>{renderDestinationGroupMetadata(item)}</div></div><div className="quick-action-row"><button type="button" className="ghost-button small-button" onClick={() => editDestination(item)}>Edit</button><button type="button" className="ghost-button icon-button" aria-label="Duplicate" onClick={() => duplicateDestination(item)}><Copy aria-hidden="true" /></button><button type="button" className="ghost-button icon-button" aria-label="Test" onClick={() => void onTestSavedDestination(item)} disabled={destinationConnectionLoading}><Play aria-hidden="true" /></button><button type="button" className="danger-button icon-button" aria-label="Delete" onClick={() => void onDeleteSavedDestination(item)}><Trash2 aria-hidden="true" /></button></div></section>)}</div></article><div className="destination-editor-grid"><article className="panel dashboard-panel"><div className="panel-header compact-header"><div><h2>{destinationForm.id ? 'Edit Destination' : 'New Destination'}</h2><p>{destinationForm.id ? 'Save changes to the selected integration.' : 'Create a new publishing integration.'}</p><div className="destination-mode-indicator">{destinationForm.id ? `Editing: ${destinationForm.name || 'Selected destination'}` : 'New destination'}</div></div><StatusBadge status={destinationConnectionResult?.state ?? 'pending'} label={destinationConnectionResult ? destinationConnectionResult.state : 'Not tested'} /></div><form className="announcement-form" onSubmit={onDestinationSubmit}><section className="config-section open-section"><div className="config-section-title"><span><ExternalLink aria-hidden="true" /> Connection</span><small>Platform and credential details</small></div><div className="config-section-body"><div className="form-grid"><div className="field"><div className="platform-label-row"><label htmlFor="destination-platform">Platform</label><button type="button" className="ghost-button icon-button compact-icon-button" aria-label={"Open " + platformLabel(destinationForm.platform) + " Setup"} title={"Open " + platformLabel(destinationForm.platform) + " setup steps"} onClick={() => setDestinationSetupModalOpen(true)}><Info aria-hidden="true" /></button></div><select id="destination-platform" aria-label="Platform" value={destinationForm.platform} onChange={(event) => { setDestinationSecrets({ discordWebhookKey: '', blueskyCredentialKey: '', mastodonCredentialKey: '' }); setDestinationConnectionResult(null); setDestinationForm((current) => ({ ...createEmptyDestinationForm(event.target.value as DestinationInput['platform']), id: current.id, name: current.name, createdAt: current.createdAt, updatedAt: current.updatedAt })); }}><option value="discord">Discord</option><option value="bluesky">Bluesky</option><option value="mastodon">Mastodon</option></select></div><label className="field"><span>Friendly Name</span><input aria-label="Friendly Name" value={destinationForm.name} onChange={(event) => updateDestination('name', event.target.value)} placeholder="Main Discord" /></label><div className="field destination-groups-field"><div className="field-label-row"><span>Groups</span><button type="button" className="ghost-button small-button" onClick={() => setDestinationWorkspaceTab('groups')}>Manage Groups</button></div><div className="membership-checklist flat-membership-list" role="group" aria-label="Destination Groups">{destinationGroups.map((group) => <label key={group.id} className="checkbox-field"><input type="checkbox" aria-label={group.name} checked={destinationGroupMembershipIDs.includes(group.id)} onChange={() => toggleDestinationGroupMembership(group.id)} disabled={group.id === DEFAULT_DESTINATION_GROUP_ID && destinationGroupMembershipIDs.length === 1} /><span>{group.name}</span></label>)}</div></div></div>{renderPlatformConnectionFields()}</div></section><section className="config-section open-section"><div className="config-section-title"><span><FileText aria-hidden="true" /> Template</span><small>Message composition and token insertion</small></div><div className="config-section-body"><label className="field"><span>{destinationTemplateLabel(destinationForm.platform)}</span><textarea ref={templateInputRef} aria-label={destinationTemplateLabel(destinationForm.platform)} value={destinationForm.template} onFocus={() => { activeTemplateInputRef.current = templateInputRef.current; }} onChange={(event) => updateDestination('template', event.target.value)} placeholder="{{stream_title}}" rows={5} /></label><div className="template-token-field"><div className="field-label-row"><span>Template Tokens</span><button type="button" className="ghost-button small-button" onClick={() => updateDestination('template', defaultTemplateForPlatform(destinationForm.platform))}>Reset Template</button></div>{destinationHelperStatus ? <p className="success-copy">{destinationHelperStatus}</p> : null}<div className="template-token-list compact-token-list" role="list" aria-label="Available template variables">{TEMPLATE_VARIABLES.map((variable) => <button key={variable} type="button" className="template-token" onClick={() => insertTemplateVariable(variable)}>{variable}</button>)}</div></div></div></section>{renderAdvancedFields()}<div className="action-cluster-main sticky-editor-actions"><button type="submit" className="primary-button">{destinationForm.id ? 'Save Changes' : 'Create Destination'}</button><button type="button" className="ghost-button" onClick={() => void onTestDestinationConnection()} disabled={destinationConnectionLoading}>{destinationConnectionLoading ? 'Testing Connection...' : `Test ${credentialSetupLabel(destinationForm.platform)}`}</button><button type="button" className="danger-button" onClick={() => void onDeleteDestination()} disabled={!destinationForm.id}>Delete</button></div>{destinationConnectionResult ? <p className={destinationConnectionResult.state === 'SUCCESS' ? 'success-banner' : 'error-banner'}>{destinationConnectionResult.message}</p> : null}</form></article></div></> : null}</section> : null}
                {selectedTab === 'profiles' ? <section className="content-stack profiles-workspace">{renderAnnouncementProfilesPanel()}</section> : null}
                {selectedTab === 'settings' ? <section className="content-stack"><article className="panel dashboard-panel settings-page-panel"><div className="panel-header compact-header"><div><h2>Settings</h2><p>Configure defaults and recovery behavior.</p></div>{pendingLiveNowSessions.length > 0 ? <StatusBadge status="pending" label={`${pendingLiveNowSessions.length} Pending Recovery`} /> : null}</div>{settingsError ? <p className="error-banner">{settingsError}</p> : null}<section className="two-column-layout"><article className="panel settings-panel"><div className="panel-header compact-header"><div><h2>Session Defaults</h2><p>Defaults prefill manual sessions and protect against accidental duplicate publishes.</p></div></div><form className="announcement-form settings-grid" onSubmit={onSettingsSubmit}><label className="field checkbox-field"><input aria-label="Duplicate Protection Enabled" type="checkbox" checked={settings.duplicateProtectionEnabled} onChange={(event) => updateSettings('duplicateProtectionEnabled', event.target.checked)} /><span>Duplicate Protection Enabled<small>Warns when a similar Go Live action happens inside the duplicate window.</small></span></label><label className="field"><span>Default Stream URL</span><input aria-label="Default Stream URL" value={settings.defaultStreamURL} onChange={(event) => updateSettings('defaultStreamURL', event.target.value)} placeholder="https://example.com/live" /><small>Prefills new Home sessions until edited.</small></label><label className="field"><span>Default Hashtags</span><input aria-label="Default Hashtags" value={settings.defaultHashtags} onChange={(event) => updateSettings('defaultHashtags', event.target.value)} placeholder="#vtuber #music" /><small>Applied to new announcements until changed.</small></label><label className="field"><span>Duplicate Window (Minutes)</span><input aria-label="Duplicate Window (Minutes)" type="number" min={1} value={settings.duplicateWindowMinutes} onChange={(event) => updateSettings('duplicateWindowMinutes', Number(event.target.value))} /><small>How long StreamSignal watches for repeated publish attempts.</small></label><button type="submit" className="primary-button">Save Settings</button></form></article><article className={pendingLiveNowSessions.length > 0 ? 'panel recovery-center recovery-pending' : 'panel recovery-center'}><div className="panel-header compact-header"><div><h2>Live Now Recovery</h2><p>Clear Bluesky Live Now states left active after interruptions.</p></div>{pendingLiveNowSessions.length > 0 ? <StatusBadge status="pending" label="Pending Recovery" /> : null}</div>{recoveryError ? <p className="error-banner">{recoveryError}</p> : null}{recoveryStatus ? <p className="success-banner">{recoveryStatus}</p> : null}{pendingLiveNowSessions.length === 0 ? <div className="empty-state compact-empty-state"><h3>No pending Live Now sessions.</h3><p>When recovery is needed, each session will appear here with a one-click clear action.</p></div> : <div className="compact-list">{pendingLiveNowSessions.map((session) => <section className="activity-card" key={session.destinationID}><PlatformIcon platform={session.platform} /><div><strong>{session.destinationName}</strong><p>{session.streamTitle || 'Untitled stream'} · {session.streamURL}</p><time dateTime={session.startedAt}>{session.startedAt}</time></div><button type="button" className="primary-button" onClick={() => void onClearPendingLiveNow(session.destinationID)} disabled={recoveryLoadingId !== null}>{recoveryLoadingId === session.destinationID ? 'Recovering Live Now...' : 'Recover and Clear Live Now'}</button></section>)}</div>}</article>{renderAboutPanel()}</section></article></section> : null}
                {SHOW_DEBUG_WORKFLOW_UI && selectedTab === 'logs' ? <section className="content-stack"><article className="panel dashboard-panel"><div className="panel-header compact-header"><div><h2>Logs</h2><p>Newest activity first, optimized for quick scanning.</p></div><span>{filteredLogs.length} of {logs.length} recent entries</span></div>{logsError ? <p className="error-banner">{logsError}</p> : null}<div className="log-toolbar"><button type="button" className="ghost-button" onClick={() => void refreshLogs()}><RefreshCw aria-hidden="true" />Refresh Logs</button><button type="button" className="primary-button" onClick={() => void onCopyDiagnostics()}><ListChecks aria-hidden="true" />Copy Diagnostics</button></div><div className="log-filter-panel" aria-label="Log filters"><div className="log-filter-group"><span>Category</span><div className="segmented-control" role="group" aria-label="Log category filter">{LOG_CATEGORY_FILTERS.map((filter) => <button key={filter.value} type="button" className={logCategoryFilter === filter.value ? 'active' : ''} onClick={() => setLogCategoryFilter(filter.value)}>{filter.label}</button>)}</div></div><div className="log-filter-group"><span>Status</span><div className="segmented-control" role="group" aria-label="Log status filter">{LOG_STATUS_FILTERS.map((filter) => <button key={filter.value} type="button" className={logStatusFilter === filter.value ? 'active' : ''} onClick={() => setLogStatusFilter(filter.value)}>{filter.label}</button>)}</div></div></div>{logs.length === 0 ? <div className="empty-state"><h3>No logs yet</h3><p>Workflow activity and validation outcomes will appear here as you use the app.</p></div> : filteredLogs.length === 0 ? <div className="empty-state"><h3>No matching logs</h3><p>Adjust the filters to see more recent activity.</p></div> : <div className="log-card-list">{filteredLogs.map((entry, index) => { const platform = inferLogPlatform(entry, destinations); return <section className="log-card" key={`${entry.timestamp}-${entry.action}-${index}`}><PlatformIcon platform={platform} /><div><strong>{logActionLabel(entry.action)}</strong><p>{entry.destination}</p><time dateTime={entry.timestamp}>{entry.timestamp}</time></div><p>{entry.message}</p><StatusBadge status={entry.status} /></section>; })}</div>}</article></section> : null}
            </div>
            {renderSaveProfileModal()}
            {renderDestinationSetupModal()}
            {workflowToast ? <div className="toast-region" aria-live="polite" aria-atomic="true"><div className={`workflow-toast toast-${workflowToast.kind}`} key={workflowToast.id}>{workflowToast.message}</div></div> : null}
        </main>
    );

}

export default App;
