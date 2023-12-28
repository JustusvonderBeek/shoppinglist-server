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

type SharingMapping struct {
	ID           int64
	ListId       int64
	SharedWithId int64
}

type ItemPerList struct {
	ID       int64
	ListId   int64
	ItemId   int64
	Quantity int64
	Checked  bool
	AddedBy  int64
}
