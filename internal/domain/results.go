package domain

type ExecutionState string
type ExecutionSummaryStatus string

const (
	ExecutionStateSuccess         ExecutionState = "SUCCESS"
	ExecutionStateFailed          ExecutionState = "FAILED"
	ExecutionStateSkipped         ExecutionState = "SKIPPED"
	ExecutionStateValidationError ExecutionState = "VALIDATION_ERROR"

	ExecutionSummaryStatusSuccess ExecutionSummaryStatus = "SUCCESS"
	ExecutionSummaryStatusWarning ExecutionSummaryStatus = "WARNING"
	ExecutionSummaryStatusPartial ExecutionSummaryStatus = "PARTIAL"
)

type ExecutionStage struct {
	Phase     string `json:"phase"`
	State     string `json:"state"`
	ErrorCode string `json:"errorCode,omitempty"`
}

type ExecutionResult struct {
	Stages      []ExecutionStage `json:"stages,omitempty"`
	Published   *bool            `json:"published"`
	Phase       string           `json:"phase,omitempty"`
	ErrorCode   string           `json:"errorCode,omitempty"`
	StartedAt   string           `json:"startedAt,omitempty"`
	CompletedAt string           `json:"completedAt,omitempty"`

	DestinationID   string              `json:"destinationID"`
	DestinationName string              `json:"destinationName"`
	Platform        DestinationPlatform `json:"platform"`
	State           ExecutionState      `json:"state"`
	Message         string              `json:"message"`
	Content         string              `json:"content"`
	Receipt         *PublishReceipt     `json:"receipt,omitempty"`
}

type PublishReceipt struct {
	Provider string `json:"provider"`
	ID       string `json:"id,omitempty"`
	URI      string `json:"uri,omitempty"`
	CID      string `json:"cid,omitempty"`
}

type ExecutionSummary struct {
	Mode                          ExecutionMode          `json:"mode"`
	Status                        ExecutionSummaryStatus `json:"status"`
	TestModeActive                bool                   `json:"testModeActive"`
	Results                       []ExecutionResult      `json:"results"`
	TotalCount                    int                    `json:"totalCount"`
	SuccessCount                  int                    `json:"successCount"`
	FailedCount                   int                    `json:"failedCount"`
	SkippedCount                  int                    `json:"skippedCount"`
	ValidationErrorCount          int                    `json:"validationErrorCount"`
	RequiresDuplicateConfirmation bool                   `json:"requiresDuplicateConfirmation"`
	DuplicateWarningMessage       string                 `json:"duplicateWarningMessage"`
}
