export interface ExecutionResult {
    destinationID: string;
    destinationName: string;
    platform: string;
    state: 'SUCCESS' | 'FAILED' | 'SKIPPED' | 'VALIDATION_ERROR';
    message: string;
    content: string;
    receipt?: PublishReceipt;
}

export interface PublishReceipt {
    provider: string;
    id?: string;
    uri?: string;
    cid?: string;
}

export interface ExecutionSummary {
    mode: 'go_live' | 'end_stream';
    status: 'SUCCESS' | 'WARNING' | 'PARTIAL';
    testModeActive: boolean;
    results: ExecutionResult[];
    totalCount: number;
    successCount: number;
    failedCount: number;
    skippedCount: number;
    validationErrorCount: number;
    requiresDuplicateConfirmation: boolean;
    duplicateWarningMessage: string;
}
