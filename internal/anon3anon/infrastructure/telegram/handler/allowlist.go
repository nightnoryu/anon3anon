package handler

// AllowList is the set of Telegram user IDs permitted to register as recipients.
// An empty list allows everyone.
type AllowList map[int64]struct{}

func NewAllowList(ids []int64) AllowList {
	set := make(AllowList, len(ids))
	for _, id := range ids {
		set[id] = struct{}{}
	}
	return set
}

func (a AllowList) Allowed(id int64) bool {
	if len(a) == 0 {
		return true
	}
	_, ok := a[id]
	return ok
}
