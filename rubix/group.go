package rubix

type GroupLevel string

const (
	GroupLevelMember  GroupLevel = "member"
	GroupLevelManager GroupLevel = "manager"
	GroupLevelOwner   GroupLevel = "owner"
)

type Group struct {
	Workspace   string
	ID          string
	Name        string
	Description string
	Users       []string    // Backwards-compat: user IDs only
	Members     []UserGroup // Detailed membership with levels
}

type UserGroup struct {
	Workspace string
	User      string
	Group     string
	Level     GroupLevel
}

type MutateGroupPayload struct {
	Title       *string
	Description *string
	UsersToAdd  map[string]GroupLevel // userID -> level
	UsersToRem  []string
	UsersLevel  map[string]GroupLevel // userID -> new level
}

type MutateGroupOption func(*MutateGroupPayload)

func WithGroupName(title string) MutateGroupOption {
	return func(p *MutateGroupPayload) { p.Title = &title }
}

func WithGroupDescription(description string) MutateGroupOption {
	return func(p *MutateGroupPayload) { p.Description = &description }
}

func WithGroupUsersToAdd(level GroupLevel, users ...string) MutateGroupOption {
	return func(p *MutateGroupPayload) {
		if p.UsersToAdd == nil {
			p.UsersToAdd = make(map[string]GroupLevel)
		}
		for _, u := range users {
			p.UsersToAdd[u] = level
		}
	}
}

func WithGroupUsersToRemove(users ...string) MutateGroupOption {
	return func(p *MutateGroupPayload) { p.UsersToRem = append(p.UsersToRem, users...) }
}

func WithGroupUsersLevel(level GroupLevel, users ...string) MutateGroupOption {
	return func(p *MutateGroupPayload) {
		if p.UsersLevel == nil {
			p.UsersLevel = make(map[string]GroupLevel)
		}
		for _, u := range users {
			p.UsersLevel[u] = level
		}
	}
}
