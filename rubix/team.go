package rubix

type TeamLevel string

const (
	TeamLevelMember  TeamLevel = "member"
	TeamLevelManager TeamLevel = "manager"
	TeamLevelOwner   TeamLevel = "owner"
)

type Team struct {
	Workspace   string
	ID          string
	Name        string
	Description string
	Users       []string   // Backwards-compat: user IDs only
	Members     []UserTeam // Detailed membership with levels
}

type UserTeam struct {
	Workspace string
	User      string
	Team      string
	Level     TeamLevel
}

type MutateTeamPayload struct {
	Title       *string
	Description *string
	UsersToAdd  map[string]TeamLevel // userID -> level
	UsersToRem  []string
	UsersLevel  map[string]TeamLevel // userID -> new level
}

type MutateTeamOption func(*MutateTeamPayload)

func WithTeamName(title string) MutateTeamOption {
	return func(p *MutateTeamPayload) { p.Title = &title }
}

func WithTeamDescription(description string) MutateTeamOption {
	return func(p *MutateTeamPayload) { p.Description = &description }
}

func WithTeamUsersToAdd(level TeamLevel, users ...string) MutateTeamOption {
	return func(p *MutateTeamPayload) {
		if p.UsersToAdd == nil {
			p.UsersToAdd = make(map[string]TeamLevel)
		}
		for _, u := range users {
			p.UsersToAdd[u] = level
		}
	}
}

func WithTeamUsersToRemove(users ...string) MutateTeamOption {
	return func(p *MutateTeamPayload) { p.UsersToRem = append(p.UsersToRem, users...) }
}

func WithTeamUsersLevel(level TeamLevel, users ...string) MutateTeamOption {
	return func(p *MutateTeamPayload) {
		if p.UsersLevel == nil {
			p.UsersLevel = make(map[string]TeamLevel)
		}
		for _, u := range users {
			p.UsersLevel[u] = level
		}
	}
}
