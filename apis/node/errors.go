package node

type CheckPiecesErrorType = uint64

const (
	UndefinedCheckPiecesErrorType CheckPiecesErrorType = iota
	CheckPiecesAPI
	CheckPiecesInvalidDeals
	CheckPiecesExpiredDeals
)

type CheckPiecesError struct {
	inner error
	EType CheckPiecesErrorType
}

func NewCheckPiecesError(inner error, etype CheckPiecesErrorType) *CheckPiecesError {
	return &CheckPiecesError{inner: inner, EType: etype}
}

func (c CheckPiecesError) Inner() error {
	return c.inner
}

type CheckSealingErrorType = uint64

const (
	UndefinedCheckSealingErrorType CheckSealingErrorType = iota
	CheckSealingAPI
	CheckSealingBadCommD
	CheckSealingExpiredTicket
)

type CheckSealingError struct {
	inner error
	EType CheckSealingErrorType
}

func NewCheckSealingError(inner error, etype CheckSealingErrorType) *CheckSealingError {
	return &CheckSealingError{inner: inner, EType: etype}
}

func (c CheckSealingError) Inner() error {
	return c.inner
}

type GetSealSeedErrorType = uint64

const (
	UndefinedGetSealSeedErrorType GetSealSeedErrorType = iota
	GetSealSeedFailedError
	GetSealSeedFatalError
)

type GetSealSeedError struct {
	inner error
	EType GetSealSeedErrorType
}

func NewGetSealSeedError(inner error, etype GetSealSeedErrorType) *GetSealSeedError {
	return &GetSealSeedError{inner: inner, EType: etype}
}

func (c GetSealSeedError) Unwrap() error {
	return c.inner
}
