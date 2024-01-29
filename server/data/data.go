package data

// Helping file containing all data structures

// ------------------------------------------------------------
// The data for authentication and login
// ------------------------------------------------------------

type User struct {
	ID       int64
	Username string
	Password string
}

// ------------------------------------------------------------
// The answer status structure
// ------------------------------------------------------------

type Answer struct {
	Status string
}

// ------------------------------------------------------------
// The list data structures
// ------------------------------------------------------------

type ListCreator struct {
	ID   int64
	Name string
}

type Shoppinglist struct {
	ListId     int64
	Name       string
	CreatedBy  ListCreator
	LastEdited string
	Items      []ItemWire
}

type ListShared struct {
	ID         int64
	ListId     int64
	SharedWith int64
}

type ListSharedWire struct {
	ListId     int64
	SharedWith int64
}

type ItemPerList struct {
	ID       int64
	ListId   int64
	ItemId   int64
	Quantity int64
	Checked  bool
	AddedBy  int64
}

// ------------------------------------------------------------
// The items that are stored in the list
// ------------------------------------------------------------

type Item struct {
	ID   int64 // Only for interal reasons
	Name string
	Icon string
}

type ItemWire struct {
	Name     string
	Icon     string
	Quantity int64
	Checked  bool
}

// ------------------------------------------------------------
// The recipe data structures
// ------------------------------------------------------------

type Recipe struct {
	ID                  int64
	Name                string
	DescriptionFilePath string
	CreatedBy           int64
	DefaultPortion      int
}

type ItemPerRecipe struct {
	ID       int64
	RecipeId int64
	ItemId   int64
	Quantity float32
}

type RecipeShared struct {
	ID         int64
	RecipeId   int64
	SharedWith int64
}
