package rubix

type UserType string

const (
	UserTypeOwner   UserType = "owner"
	UserTypeMember  UserType = "member"
	UserTypeSupport UserType = "support"
)

type UserRowState int

const (
	UserRowStatePending   UserRowState = 0
	UserRowStateActive    UserRowState = 1
	UserRowStateSuspended UserRowState = 2
	UserRowStateArchived  UserRowState = 3
	UserRowStateRemoved   UserRowState = 4
)
