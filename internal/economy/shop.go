package economy

const (
	KindTheme = "theme"
	KindFelt  = "felt"
	KindBack  = "back"
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
}

var DefaultLook = map[string]string{
	KindTheme: "paper",
	KindFelt:  "linen",
	KindBack:  "classic",
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
