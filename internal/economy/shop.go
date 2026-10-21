package economy

const (
	KindTheme  = "theme"
	KindFelt   = "felt"
	KindBack   = "back"
	KindAvatar = "avatar"
)

type Item struct {
	Kind  string
	ID    string
	Price int64
}

func (i Item) Key() string {
	return i.Kind + ":" + i.ID
}

var Catalog = []Item{
	{KindTheme, "paper", 0},
	{KindTheme, "hall", 0},
	{KindTheme, "desert", 2500},
	{KindTheme, "blush", 2500},
	{KindTheme, "forest", 3000},
	{KindTheme, "ocean", 3000},
	{KindTheme, "noir", 4000},

	{KindFelt, "linen", 0},
	{KindFelt, "green", 0},
	{KindFelt, "sand", 1500},
	{KindFelt, "wine", 1500},
	{KindFelt, "navy", 1500},
	{KindFelt, "graphite", 1500},
	{KindFelt, "teal", 2000},
	{KindFelt, "terracotta", 2000},
	{KindFelt, "royal", 2500},

	{KindBack, "classic", 0},
	{KindBack, "bito", 0},
	{KindBack, "checker", 1500},
	{KindBack, "grove", 1500},
	{KindBack, "emerald", 2000},
	{KindBack, "sakura", 2000},
	{KindBack, "waves", 2500},
	{KindBack, "deco", 2500},
	{KindBack, "royal", 3000},

	{KindAvatar, "initial", 0},
	{KindAvatar, "spade", 0},
	{KindAvatar, "heart", 0},
	{KindAvatar, "diamond", 0},
	{KindAvatar, "club", 0},
	{KindAvatar, "cat", 1000},
	{KindAvatar, "dog", 1000},
	{KindAvatar, "rabbit", 1000},
	{KindAvatar, "fish", 1000},
	{KindAvatar, "bird", 1000},
	{KindAvatar, "cactus", 1500},
	{KindAvatar, "ghost", 1500},
	{KindAvatar, "robot", 1500},
	{KindAvatar, "alien", 1500},
	{KindAvatar, "sunglasses", 1500},
	{KindAvatar, "clover", 2000},
	{KindAvatar, "lightning", 2000},
	{KindAvatar, "fire", 2000},
	{KindAvatar, "anchor", 2000},
	{KindAvatar, "rocket", 2000},
	{KindAvatar, "horse", 2500},
	{KindAvatar, "sword", 2500},
	{KindAvatar, "detective", 2500},
	{KindAvatar, "skull", 2500},
	{KindAvatar, "trophy", 3000},
	{KindAvatar, "crown", 3000},
}

var DefaultLook = map[string]string{
	KindTheme:  "paper",
	KindFelt:   "linen",
	KindBack:   "classic",
	KindAvatar: "initial",
}

var catalogByKey = func() map[string]Item {
	m := make(map[string]Item, len(Catalog))
	for _, it := range Catalog {
		m[it.Key()] = it
	}
	return m
}()

func ItemByKey(key string) (Item, bool) {
	it, ok := catalogByKey[key]
	return it, ok
}
