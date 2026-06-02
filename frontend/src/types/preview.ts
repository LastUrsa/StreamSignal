export interface AnnouncementInput {
    streamTitle: string;
    streamURL: string;
    category: string;
    message: string;
    hashtags: string;
}

export interface PreviewItem {
    destinationID: string;
    destinationName: string;
    platform: string;
    previewLabel?: string;
    content: string;
    characterCount: number;
    validationState: string;
    validationNotes: string[];
    linkPreview?: LinkPreviewMetadata | null;
}

export interface LinkPreviewMetadata {
    url: string;
    title: string;
    description: string;
    imageURL: string;
    siteName: string;
}
