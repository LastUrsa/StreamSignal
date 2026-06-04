export interface AnnouncementProfileInput {
    id: string;
    name: string;
    description: string;
    defaultStreamTitle: string;
    defaultStreamURL: string;
    defaultCategory: string;
    defaultMessage: string;
    defaultHashtags: string;
    defaultAnnouncementImageURL: string;
    defaultAnnouncementImageDataURL: string;
    defaultBlueskyCardThumbnailURL: string;
    defaultBlueskyCardThumbnailDataURL: string;
    defaultGroupID: string;
    lastUsedAt?: string;
    useCount?: number;
    createdAt?: string;
    updatedAt?: string;
}

export interface AnnouncementProfileFormState {
    id: string;
    name: string;
    description: string;
    defaultStreamTitle: string;
    defaultStreamURL: string;
    defaultCategory: string;
    defaultMessage: string;
    defaultHashtags: string;
    defaultAnnouncementImageURL: string;
    defaultAnnouncementImageDataURL: string;
    defaultBlueskyCardThumbnailURL: string;
    defaultBlueskyCardThumbnailDataURL: string;
    defaultGroupID: string;
}

export function createEmptyAnnouncementProfileForm(): AnnouncementProfileFormState {
    return {
        id: '',
        name: '',
        description: '',
        defaultStreamTitle: '',
        defaultStreamURL: '',
        defaultCategory: '',
        defaultMessage: '',
        defaultHashtags: '',
        defaultAnnouncementImageURL: '',
        defaultAnnouncementImageDataURL: '',
        defaultBlueskyCardThumbnailURL: '',
        defaultBlueskyCardThumbnailDataURL: '',
        defaultGroupID: '',
    };
}

export function toAnnouncementProfileFormState(profile: AnnouncementProfileInput): AnnouncementProfileFormState {
    return {
        id: profile.id,
        name: profile.name,
        description: profile.description,
        defaultStreamTitle: profile.defaultStreamTitle ?? '',
        defaultStreamURL: profile.defaultStreamURL,
        defaultCategory: profile.defaultCategory,
        defaultMessage: profile.defaultMessage,
        defaultHashtags: profile.defaultHashtags,
        defaultAnnouncementImageURL: profile.defaultAnnouncementImageURL,
        defaultAnnouncementImageDataURL: profile.defaultAnnouncementImageDataURL,
        defaultBlueskyCardThumbnailURL: profile.defaultBlueskyCardThumbnailURL,
        defaultBlueskyCardThumbnailDataURL: profile.defaultBlueskyCardThumbnailDataURL,
        defaultGroupID: profile.defaultGroupID,
    };
}

export function toAnnouncementProfileInput(form: AnnouncementProfileFormState): AnnouncementProfileInput {
    return {
        id: form.id,
        name: form.name,
        description: form.description,
        defaultStreamTitle: form.defaultStreamTitle,
        defaultStreamURL: form.defaultStreamURL,
        defaultCategory: form.defaultCategory,
        defaultMessage: form.defaultMessage,
        defaultHashtags: form.defaultHashtags,
        defaultAnnouncementImageURL: form.defaultAnnouncementImageURL,
        defaultAnnouncementImageDataURL: form.defaultAnnouncementImageDataURL,
        defaultBlueskyCardThumbnailURL: form.defaultBlueskyCardThumbnailURL,
        defaultBlueskyCardThumbnailDataURL: form.defaultBlueskyCardThumbnailDataURL,
        defaultGroupID: form.defaultGroupID,
    };
}
