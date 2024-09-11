package data

import "time"

// Helping file containing all data structures

// NOTE: The field names are kept in camelCase
// that is, 'field' or 'thisIsAField'

// Furthermore, structs like the user can be used in multiple calls
// and scenarios, therefore fields can be omitted which must be explicitly marked

// ------------------------------------------------------------
// Operational data structures
// ------------------------------------------------------------

// ? Shouldn't this already be contained in the framework and http status?
type Answer struct {
	Status string
}

// ------------------------------------------------------------
// The data for authentication, login and display of information
// ------------------------------------------------------------

type User struct {
	OnlineID  int64     `json:"onlineId"`
	Username  string    `json:"username"`
	Password  string    `json:"password,omitempty"`
	Created   time.Time `json:"created,omitempty"`
	LastLogin time.Time `json:"lastLogin,omitempty"` // <- what do we need this information for? would only be relevant when displaying or using this in the app
}

// ------------------------------------------------------------
// The list data structures
// ------------------------------------------------------------

// Note: This can also be represented by the user format with fields omitted!!!
// Deprecated: ListCreated is deprecated. Use user instead
type ListCreator struct {
	ID   int64  `json:"onlineId"`
	Name string `json:"username"`
}

type List struct {
	ListId int64  `json:"listId"`
	Title  string `json:"title"`
	// Elements    int32       `json:"elements"`
	CreatedBy   ListCreator `json:"createdBy"`
	CreatedAt   time.Time   `json:"createdAt,omitempty"`
	LastUpdated time.Time   `json:"lastUpdated"`
	Items       []ItemWire  `json:"items"`
}

type Item struct {
	ItemId int64  `json:"itemId"` // Only for interal reasons
	Name   string `json:"name"`
	Icon   string `json:"icon,omitempty"`
}

type ItemWire struct {
	Name     string `json:"name"`
	Icon     string `json:"icon"`
	Quantity int64  `json:"quantity"`
	Checked  bool   `json:"checked"`
	AddedBy  int64  `json:"addedBy"`
}

type ListItem struct {
	ID        int64 `json:"id,omitempty"`
	ListId    int64 `json:"listId"`
	ItemId    int64 `json:"itemId"`
	Quantity  int64 `json:"quantity,omitempty"`
	Checked   bool  `json:"checked,omitempty"`
	CreatedBy int64 `json:"createdBy,omitempty"`
	AddedBy   int64 `json:"addedBy,omitempty"`
}

type ListShared struct {
	ID         int64     `json:"shareId,omitempty"`
	ListId     int64     `json:"listId"`
	CreatedBy  int64     `json:"createdBy"`
	SharedWith []int64   `json:"sharedWith"`
	Created    time.Time `json:"created,omitempty"`
}

type ListSharedWire struct {
	SharedBy   int64     `json:"sharedBy"` // Could also be obtained by the request token or user...?
	SharedWith []int64   `json:"sharedWith"`
	Created    time.Time `json:"created,omitempty"`
}

// ------------------------------------------------------------
// The items that are stored in the list
// ------------------------------------------------------------

// ------------------------------------------------------------
// The recipe data structures
// ------------------------------------------------------------

type Receipt struct {
	ReceiptId      int64                `json:"receiptId"`
	Name           string               `json:"name"`
	CreatedBy      int64                `json:"createdBy"`
	CreatedAt      time.Time            `json:"createdAt"`
	LastUpdate     time.Time            `json:"lastUpdated"`
	DefaultPortion int                  `json:"defaultPortion"`
	Ingredients    []Ingredient         `json:"ingredients"`
	Description    []ReceiptDescription `json:"description"`
}

type DbReceipt struct {
	ReceiptId      int64
	Name           string
	CreatedBy      int64
	CreatedAt      time.Time
	LastUpdated    time.Time
	DefaultPortion int
}

type Ingredient struct {
	Name         string `json:"name"`
	Icon         string `json:"icon"`
	Quantity     int    `json:"quantity"`
	QuantityType string `json:"quantityType"`
}

type ReceiptDescription struct {
	Order int    `json:"descriptionOrder"`
	Step  string `json:"step"`
}

type IngredientPerReceipt struct {
	ID           int64
	RecipeId     int64
	CreatedBy    int64
	ItemId       int64
	Quantity     float32
	QuantityType string
}

type DescriptionPerReceipt struct {
	ID               int64
	ReceiptId        int64
	CreatedBy        int64
	Description      string
	DescriptionOrder int
}

type RecipeShared struct {
	ID         int64
	RecipeId   int64
	SharedWith int64
}
