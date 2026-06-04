import { describe, expect, it, vi, beforeEach } from 'vitest';

vi.mock('../../../wailsjs/go/main/App', () => ({
    ClearPendingLiveNowSession: vi.fn(),
    DeleteAnnouncementProfile: vi.fn(),
    DeleteDestination: vi.fn(),
    DeleteDestinationGroup: vi.fn(),
    DuplicateAnnouncementProfile: vi.fn(),
    DuplicateDestinationGroup: vi.fn(),
    EndStream: vi.fn(),
    ForceGoLive: vi.fn(),
    GeneratePreview: vi.fn(),
    GetAnnouncementProfile: vi.fn(),
    GetDestination: vi.fn(),
    GetDestinationGroup: vi.fn(),
    GetDestinationGroupIDs: vi.fn(),
    GetDiagnostics: vi.fn(),
    GetGroupDestinations: vi.fn(),
    GetLogs: vi.fn(),
    GetSettings: vi.fn(),
    GoLive: vi.fn(),
    ListAnnouncementProfiles: vi.fn(),
    ListDestinationGroups: vi.fn(),
    ListPendingLiveNowSessions: vi.fn(),
    ListDestinations: vi.fn(),
    MarkAnnouncementProfileUsed: vi.fn(),
    SaveDestination: vi.fn(),
    SaveAnnouncementProfile: vi.fn(),
    SaveDestinationGroup: vi.fn(),
    SaveSettings: vi.fn(),
    SetDestinationGroups: vi.fn(),
    SetGroupDestinations: vi.fn(),
    TestDestinationConnection: vi.fn(),
}));

import * as bindings from '../../../wailsjs/go/main/App';
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
    getAnnouncementProfile,
    getDestination,
    getDestinationGroup,
    getDestinationGroupIDs,
    getGroupDestinations,
    getDiagnostics,
    getLogs,
    getSettings,
    goLive,
    listAnnouncementProfiles,
    listDestinationGroups,
    listDestinations,
    listPendingLiveNowSessions,
    markAnnouncementProfileUsed,
    saveAnnouncementProfile,
    saveDestination,
    saveDestinationGroup,
    saveSettings,
    setDestinationGroups,
    setGroupDestinations,
    testDestinationConnection,
} from './streamsignal';

describe('streamsignal api wrappers', () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    it('delegates each wrapper to the matching Wails binding', async () => {
        const announcement = {
            streamTitle: 'Going Live',
            streamURL: 'https://example.com/live',
            category: 'Music',
            message: 'See you there',
            hashtags: '#vtuber',
            announcementImageURL: 'https://example.com/promo.png',
            announcementImageDataURL: '',
            blueskyCardThumbnailURL: 'https://example.com/card.png',
            blueskyCardThumbnailDataURL: '',
        };
        const destinationGroupID = 'all-destinations';
        const destinationGroup = {
            id: destinationGroupID,
            name: 'All Destinations',
            description: '',
            createdAt: '',
            updatedAt: '',
        };
        const destination = {
            id: 'discord-main',
            platform: 'discord',
            name: 'Main Discord',
            enabled: true,
            template: '{{stream_title}}',
            configJSON: '{"webhookKey":"discord/main"}',
            createdAt: '',
            updatedAt: '',
        };
        const profile = {
            id: 'music-profile',
            name: 'Music Stream',
            description: 'Music nights',
            defaultStreamTitle: 'Late Night Music',
            defaultStreamURL: 'https://example.com/live',
            defaultCategory: 'Music',
            defaultMessage: 'See you there',
            defaultHashtags: '#vtuber',
            defaultAnnouncementImageURL: 'https://example.com/promo.png',
            defaultAnnouncementImageDataURL: '',
            defaultBlueskyCardThumbnailURL: 'https://example.com/card.png',
            defaultBlueskyCardThumbnailDataURL: '',
            defaultGroupID: destinationGroupID,
            createdAt: '',
            updatedAt: '',
        };
        const settings = {
            defaultStreamURL: '',
            defaultHashtags: '',
            duplicateProtectionEnabled: true,
            duplicateWindowMinutes: 10,
            selectedDestinationGroupID: destinationGroupID,
        };

        vi.mocked(bindings.GetLogs).mockResolvedValue([] as never);
        vi.mocked(bindings.GetDiagnostics).mockResolvedValue('diag' as never);
        vi.mocked(bindings.GeneratePreview).mockResolvedValue([] as never);
        vi.mocked(bindings.GoLive).mockResolvedValue({
            mode: 'go_live',
            status: 'SUCCESS',
            testModeActive: false,
            results: [],
            totalCount: 0,
            successCount: 0,
            failedCount: 0,
            skippedCount: 0,
            validationErrorCount: 0,
            requiresDuplicateConfirmation: false,
            duplicateWarningMessage: '',
        } as never);
        vi.mocked(bindings.ForceGoLive).mockResolvedValue({
            mode: 'go_live',
            status: 'SUCCESS',
            testModeActive: false,
            results: [],
            totalCount: 0,
            successCount: 0,
            failedCount: 0,
            skippedCount: 0,
            validationErrorCount: 0,
            requiresDuplicateConfirmation: false,
            duplicateWarningMessage: '',
        } as never);
        vi.mocked(bindings.EndStream).mockResolvedValue({
            mode: 'end_stream',
            status: 'SUCCESS',
            testModeActive: false,
            results: [],
            totalCount: 0,
            successCount: 0,
            failedCount: 0,
            skippedCount: 0,
            validationErrorCount: 0,
            requiresDuplicateConfirmation: false,
            duplicateWarningMessage: '',
        } as never);
        vi.mocked(bindings.ListPendingLiveNowSessions).mockResolvedValue([] as never);
        vi.mocked(bindings.ClearPendingLiveNowSession).mockResolvedValue({
            destinationID: 'bluesky-main',
            destinationName: 'Main Bluesky',
            platform: 'bluesky',
            state: 'SUCCESS',
            message: 'Recovered and cleared pending Live Now session.',
            content: '',
        } as never);
        vi.mocked(bindings.ListDestinations).mockResolvedValue([] as never);
        vi.mocked(bindings.GetDestination).mockResolvedValue(destination as never);
        vi.mocked(bindings.ListDestinationGroups).mockResolvedValue([destinationGroup] as never);
        vi.mocked(bindings.GetDestinationGroup).mockResolvedValue(destinationGroup as never);
        vi.mocked(bindings.SaveDestinationGroup).mockResolvedValue(destinationGroup as never);
        vi.mocked(bindings.DuplicateDestinationGroup).mockResolvedValue({ ...destinationGroup, id: 'all-destinations-copy', name: 'All Destinations Copy' } as never);
        vi.mocked(bindings.DeleteDestinationGroup).mockResolvedValue(undefined as never);
        vi.mocked(bindings.GetGroupDestinations).mockResolvedValue([destination] as never);
        vi.mocked(bindings.SetGroupDestinations).mockResolvedValue(undefined as never);
        vi.mocked(bindings.GetDestinationGroupIDs).mockResolvedValue([destinationGroupID] as never);
        vi.mocked(bindings.SetDestinationGroups).mockResolvedValue(undefined as never);
        vi.mocked(bindings.ListAnnouncementProfiles).mockResolvedValue([profile] as never);
        vi.mocked(bindings.GetAnnouncementProfile).mockResolvedValue(profile as never);
        vi.mocked(bindings.SaveAnnouncementProfile).mockResolvedValue(profile as never);
        vi.mocked(bindings.DuplicateAnnouncementProfile).mockResolvedValue({ ...profile, id: 'music-profile-copy', name: 'Music Stream Copy' } as never);
        vi.mocked(bindings.MarkAnnouncementProfileUsed).mockResolvedValue(undefined as never);
        vi.mocked(bindings.DeleteAnnouncementProfile).mockResolvedValue(undefined as never);
        vi.mocked(bindings.SaveDestination).mockResolvedValue(destination as never);
        vi.mocked(bindings.DeleteDestination).mockResolvedValue(undefined as never);
        vi.mocked(bindings.GetSettings).mockResolvedValue(settings as never);
        vi.mocked(bindings.SaveSettings).mockResolvedValue(settings as never);
        vi.mocked(bindings.TestDestinationConnection).mockResolvedValue({
            platform: 'discord',
            state: 'SUCCESS',
            message: 'Connection verified.',
        } as never);

        await getLogs();
        await getDiagnostics();
        await generatePreview(announcement, destinationGroupID);
        await goLive(announcement, destinationGroupID);
        await forceGoLive(announcement, destinationGroupID);
        await endStream(announcement, destinationGroupID);
        await listPendingLiveNowSessions();
        await clearPendingLiveNowSession('bluesky-main');
        await listDestinations();
        await getDestination(destination.id);
        await listDestinationGroups();
        await getDestinationGroup(destinationGroupID);
        await saveDestinationGroup(destinationGroup);
        await duplicateDestinationGroup(destinationGroupID);
        await deleteDestinationGroup(destinationGroupID);
        await getGroupDestinations(destinationGroupID);
        await setGroupDestinations(destinationGroupID, [destination.id]);
        await getDestinationGroupIDs(destination.id);
        await setDestinationGroups(destination.id, [destinationGroupID]);
        await listAnnouncementProfiles();
        await getAnnouncementProfile(profile.id);
        await saveAnnouncementProfile(profile);
        await duplicateAnnouncementProfile(profile.id);
        await markAnnouncementProfileUsed(profile.id);
        await deleteAnnouncementProfile(profile.id);
        await saveDestination(destination);
        await deleteDestination('discord-main');
        await getSettings();
        await saveSettings(settings);
        await testDestinationConnection(destination);

        expect(bindings.GetLogs).toHaveBeenCalled();
        expect(bindings.GetDiagnostics).toHaveBeenCalled();
        expect(bindings.GeneratePreview).toHaveBeenCalledWith(expect.objectContaining({ ...announcement, destinationGroupID }));
        expect(bindings.GoLive).toHaveBeenCalledWith(expect.objectContaining({ ...announcement, destinationGroupID }));
        expect(bindings.ForceGoLive).toHaveBeenCalledWith(expect.objectContaining({ ...announcement, destinationGroupID }));
        expect(bindings.EndStream).toHaveBeenCalledWith(expect.objectContaining({ ...announcement, destinationGroupID }));
        expect(bindings.ListPendingLiveNowSessions).toHaveBeenCalled();
        expect(bindings.ClearPendingLiveNowSession).toHaveBeenCalledWith('bluesky-main');
        expect(bindings.ListDestinations).toHaveBeenCalled();
        expect(bindings.GetDestination).toHaveBeenCalledWith(destination.id);
        expect(bindings.ListDestinationGroups).toHaveBeenCalled();
        expect(bindings.GetDestinationGroup).toHaveBeenCalledWith(destinationGroupID);
        expect(bindings.SaveDestinationGroup).toHaveBeenCalledWith(expect.objectContaining({
            id: destinationGroup.id,
            name: destinationGroup.name,
            description: destinationGroup.description,
        }));
        expect(bindings.DuplicateDestinationGroup).toHaveBeenCalledWith(destinationGroupID);
        expect(bindings.DeleteDestinationGroup).toHaveBeenCalledWith(destinationGroupID);
        expect(bindings.GetGroupDestinations).toHaveBeenCalledWith(destinationGroupID);
        expect(bindings.SetGroupDestinations).toHaveBeenCalledWith(destinationGroupID, [destination.id]);
        expect(bindings.GetDestinationGroupIDs).toHaveBeenCalledWith(destination.id);
        expect(bindings.SetDestinationGroups).toHaveBeenCalledWith(destination.id, [destinationGroupID]);
        expect(bindings.ListAnnouncementProfiles).toHaveBeenCalled();
        expect(bindings.GetAnnouncementProfile).toHaveBeenCalledWith(profile.id);
        expect(bindings.SaveAnnouncementProfile).toHaveBeenCalledWith(expect.objectContaining({
            id: profile.id,
            name: profile.name,
            defaultStreamTitle: profile.defaultStreamTitle,
            defaultStreamURL: profile.defaultStreamURL,
            defaultAnnouncementImageURL: profile.defaultAnnouncementImageURL,
            defaultGroupID: profile.defaultGroupID,
        }));
        expect(bindings.DuplicateAnnouncementProfile).toHaveBeenCalledWith(profile.id);
        expect(bindings.MarkAnnouncementProfileUsed).toHaveBeenCalledWith(profile.id);
        expect(bindings.DeleteAnnouncementProfile).toHaveBeenCalledWith(profile.id);
        expect(bindings.SaveDestination).toHaveBeenCalledWith(expect.objectContaining({
            id: destination.id,
            platform: destination.platform,
            name: destination.name,
            enabled: destination.enabled,
            template: destination.template,
            configJSON: destination.configJSON,
        }));
        expect(bindings.SaveDestination).toHaveBeenCalledWith(expect.not.objectContaining({
            createdAt: '',
            updatedAt: '',
        }));
        expect(bindings.DeleteDestination).toHaveBeenCalledWith('discord-main');
        expect(bindings.GetSettings).toHaveBeenCalled();
        expect(bindings.SaveSettings).toHaveBeenCalledWith(expect.objectContaining(settings));
        expect(bindings.TestDestinationConnection).toHaveBeenCalledWith(expect.objectContaining({
            id: destination.id,
            platform: destination.platform,
            name: destination.name,
            enabled: destination.enabled,
            template: destination.template,
            configJSON: destination.configJSON,
        }));
    });

    it('keeps Wails string rejections readable', () => {
        expect(errorMessage('Discord webhook key is required', 'Unable to save destination.')).toBe('Discord webhook key is required');
        expect(errorMessage(new Error('Bluesky credential key is required'), 'Unable to save destination.')).toBe('Bluesky credential key is required');
        expect(errorMessage(null, 'Unable to save destination.')).toBe('Unable to save destination.');
    });

    it('normalizes null Wails list responses to empty arrays', async () => {
        const announcement = {
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

        vi.mocked(bindings.GetLogs).mockResolvedValue(null as never);
        vi.mocked(bindings.GeneratePreview).mockResolvedValue(null as never);
        vi.mocked(bindings.ListPendingLiveNowSessions).mockResolvedValue(null as never);
        vi.mocked(bindings.ListDestinations).mockResolvedValue(null as never);
        vi.mocked(bindings.ListDestinationGroups).mockResolvedValue(null as never);
        vi.mocked(bindings.ListAnnouncementProfiles).mockResolvedValue(null as never);
        vi.mocked(bindings.GoLive).mockResolvedValue({
            mode: 'go_live',
            status: 'SUCCESS',
            testModeActive: false,
            results: null,
            totalCount: 0,
            successCount: 0,
            failedCount: 0,
            skippedCount: 0,
            validationErrorCount: 0,
            requiresDuplicateConfirmation: false,
            duplicateWarningMessage: '',
        } as never);

        await expect(getLogs()).resolves.toEqual([]);
        await expect(generatePreview(announcement)).resolves.toEqual([]);
        await expect(listPendingLiveNowSessions()).resolves.toEqual([]);
        await expect(listDestinations()).resolves.toEqual([]);
        await expect(listDestinationGroups()).resolves.toEqual([]);
        await expect(listAnnouncementProfiles()).resolves.toEqual([]);
        await expect(goLive(announcement)).resolves.toEqual(
            expect.objectContaining({
                results: [],
            }),
        );
    });

    it('normalizes null preview validation notes to an empty array', async () => {
        vi.mocked(bindings.GeneratePreview).mockResolvedValue([
            {
                destinationID: 'discord-main',
                destinationName: 'Main Discord',
                platform: 'discord',
                content: 'Going live',
                characterCount: 10,
                validationState: 'VALID',
                validationNotes: null,
            },
        ] as never);

        await expect(
            generatePreview({
                streamTitle: '',
                streamURL: '',
                category: '',
                message: '',
                hashtags: '',
                announcementImageURL: '',
                announcementImageDataURL: '',
                blueskyCardThumbnailURL: '',
                blueskyCardThumbnailDataURL: '',
            }),
        ).resolves.toEqual([
            expect.objectContaining({
                validationNotes: [],
            }),
        ]);
    });
});
