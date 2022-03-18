package rubix

type MembershipRole string

const (
	MembershipRoleOwner   MembershipRole = "owner"   // full access
	MembershipRoleMember  MembershipRole = "member"  // Permissions only
	MembershipRoleSupport MembershipRole = "support" // Support agent
)

type Membership struct {
	MemberID string
	Role     MembershipRole
}
