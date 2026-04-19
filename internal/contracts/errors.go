package contracts

type ErrorCode string

const (
	ErrCodeInvalidURL       ErrorCode = "INVALID_URL"
	ErrCodeProfileNotFound  ErrorCode = "PROFILE_NOT_FOUND"
	ErrCodeMangaNotFound    ErrorCode = "MANGA_NOT_FOUND"
	ErrCodeChapterNotFound  ErrorCode = "CHAPTER_NOT_FOUND"
	ErrCodeDownloadFailed   ErrorCode = "DOWNLOAD_FAILED"
	ErrCodeJobNotFound      ErrorCode = "JOB_NOT_FOUND"
	ErrCodeStoreFailure     ErrorCode = "STORE_FAILURE"
	ErrCodeSettingsInvalid  ErrorCode = "SETTINGS_INVALID"
	ErrCodeBootstrapFailure ErrorCode = "BOOTSTRAP_FAILURE"
)

type ContractError struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
}

func (e ContractError) Error() string {
	if e.Message != "" {
		return string(e.Code) + ": " + e.Message
	}

	return string(e.Code)
}
