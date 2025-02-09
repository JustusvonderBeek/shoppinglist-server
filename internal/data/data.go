package data

import (
	"time"
)

// Helping file containing all data structures

// NOTE: The field names are kept in camelCase
// that is, 'field' or 'thisIsAField'

// Furthermore, structs like the user can be used in multiple calls
// and scenarios, therefore fields can be omitted which must be explicitly marked

// ------------------------------------------------------------
// Operational data structures
// ------------------------------------------------------------

// ? Shouldn't this already be contained in the framework and http status?,
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
	Role      string    `json:"userRights,omitempty"`
	Created   time.Time `json:"created,omitempty"`
	LastLogin time.Time `json:"lastLogin,omitempty"` // <- what do we need this information for? would only be relevant when displaying or using this in the app
}

func (u *User) ToWireFormat() User {
	return User{
		OnlineID: u.OnlineID,
		Username: u.Username,
		Role:     u.Role,
	}
}

const (
	USER  = "US"
	ADMIN = "AD"
)

type Role struct {
	UserID int64  `json:"userId"`
	Role   string `json:"role"`
}

func (r *Role) ToEnumConstant() string {
	switch r.Role {
	case "AD":
		return ADMIN
	case "US":
		return USER
	default:
		return USER
	}
}

// ------------------------------------------------------------
// The list data structures
// ------------------------------------------------------------

// ListCreator Note: This can also be represented by the user format with fields omitted!!!
type ListCreator struct {
	ID   int64  `json:"onlineId"`
	Name string `json:"username"`
}

type ListPK struct {
	ListID    int64
	CreatedBy int64
}

type List struct {
	ListId      int64       `json:"listId"`
	CreatedBy   ListCreator `json:"createdBy"`
	Title       string      `json:"title"`
	CreatedAt   time.Time   `json:"createdAt,omitempty"`
	LastUpdated time.Time   `json:"lastUpdated"`
	Version     int64       `json:"version"`
	Items       []ItemWire  `json:"items"`
}

type Item struct {
	ItemId int64  `json:"itemId"` // Only for internal reasons
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
	ListId    int64 `json:"listId"`
	CreatedBy int64 `json:"createdBy"`
	ItemId    int64 `json:"itemId"`
	Quantity  int64 `json:"quantity,omitempty"`
	Checked   bool  `json:"checked,omitempty"`
	AddedBy   int64 `json:"addedBy,omitempty"`
}

type ListShared struct {
	ListId       int64     `json:"listId"`
	CreatedBy    int64     `json:"createdBy"`
	SharedWithId int64     `json:"sharedWithId"`
	Created      time.Time `json:"created,omitempty"`
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

type RecipePK struct {
	RecipeId  int64
	CreatedBy int64
}

type Recipe struct {
	RecipeId       int64               `json:"onlineId" db:"recipeId"`
	Name           string              `json:"name" db:"name"`
	CreatedBy      ListCreator         `json:"createdBy" db:"createdBy"`
	CreatedAt      time.Time           `json:"createdAt" db:"createdAt"`
	LastUpdate     time.Time           `json:"lastUpdated" db:"lastUpdate"`
	Version        int                 `json:"version" db:"version"`
	DefaultPortion int                 `json:"defaultPortion" db:"defaultPortion"`
	Ingredients    []Ingredient        `json:"ingredients"`
	Description    []RecipeDescription `json:"description"`
}

type DBRecipe struct {
	RecipeId       int64     `db:"recipeId"`
	CreatedBy      int64     `db:"createdBy"`
	Name           string    `db:"name"`
	CreatedAt      time.Time `db:"createdAt"`
	LastUpdated    time.Time `db:"lastUpdate"`
	DefaultPortion int       `db:"defaultPortion"`
}

type Ingredient struct {
	Name         string `json:"name" db:"name"`
	Icon         string `json:"icon" db:"icon"`
	Quantity     int    `json:"quantity" db:"quantity"`
	QuantityType string `json:"quantityType" db:"quantityType"`
}

type RecipeDescription struct {
	Order int    `json:"order"`
	Step  string `json:"step"`
}

type IngredientPerRecipe struct {
	RecipeId     int64   `json:"recipeId" db:"description"`
	CreatedBy    int64   `json:"createdBy" db:"createdBy"`
	ItemId       int64   `json:"itemId" db:"itemId"`
	Quantity     float32 `json:"quantity" db:"quantity"`
	QuantityType string  `json:"quantityType" db:"quantityType"`
}

type DescriptionPerRecipe struct {
	RecipeId         int64  `json:"recipeId" db:"recipeId"`
	CreatedBy        int64  `json:"createdBy" db:"createdBy"`
	Description      string `json:"step" db:"description"`
	DescriptionOrder int    `json:"order" db:"descriptionOrder"`
}

type RecipeShared struct {
	ID         int64 `json:"id,omitempty"`
	RecipeId   int64 `json:"recipeId"`
	SharedWith int64 `json:"sharedWith"`
}
