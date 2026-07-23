package models

type VoteType string

const (
	VoteVerify VoteType = "verify"
	VoteReject VoteType = "reject"
)

// IsValid checks if the vote is a recognized option
func (v VoteType) IsValid() bool {
	switch v {
	case VoteVerify, VoteReject:
		return true
	default:
		return false
	}
}
