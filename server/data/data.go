package data

// Helping file containing all data structures

// ------------------------------------------------------------
// The data for authentication and login
// ------------------------------------------------------------

type User struct {
	ID       int64
	Username string
	Passwd   string
}

// ------------------------------------------------------------
// The list data structures
// ------------------------------------------------------------

type Shoppinglist struct {
	ID        int64
	Name      string
	CreatedBy int64
}

type ListShared struct {
	ID         int64
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
	ID   int64
	Name string
	Icon string
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
}
