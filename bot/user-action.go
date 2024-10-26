package bot

type UserAction string

const (
	UAStart           UserAction = "start"
	UAList            UserAction = "list"
	UAAdd             UserAction = "add"
	UACurrent         UserAction = "current"
	UACurrentSet      UserAction = "current_set"
	UACurrentRandom   UserAction = "current_random"
	UACurrentAbort    UserAction = "current_abort"
	UACurrentComplete UserAction = "current_complete"
	UACurrentDeadline UserAction = "current_deadline"
	UARemove          UserAction = "remove"
	UAHistory         UserAction = "history"
	UAHistoryAdd      UserAction = "history_add"
	UAHistoryRemove   UserAction = "history_remove"
)
