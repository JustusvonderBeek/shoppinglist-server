package database

import (
	"log"
	"testing"
	"time"

	"shop.cloudsheeptech.com/server/data"
)

func TestCreatingRecipe(t *testing.T) {
	log.Print("Testing creating a new recipe")
	recipe := data.Recipe{
		ReceiptId:      0,
		Name:           "new recipe",
		CreatedBy:      12345,
		CreatedAt:      time.Now(),
		LastUpdate:     time.Now(),
		DefaultPortion: 2,
		Ingredients: []data.Ingredient{
			data.Ingredient{
				Name:         "ingredient 1",
				Icon:         "icon ingredient 1",
				Quantity:     12,
				QuantityType: "g",
			},
			data.Ingredient{
				Name:         "ingredient 2",
				Icon:         "icon ingredient 2",
				Quantity:     2,
				QuantityType: "kg",
			},
		},
		Description: []data.RecipeDescription{
			data.RecipeDescription{
				Order: 1,
				Step:  "this is the first step",
			},
			data.RecipeDescription{
				Order: 2,
				Step:  "this is the second step with more text",
			},
			data.RecipeDescription{
				Order: 3,
				Step:  "this is the third step with a lot of text: Lorem ipsum dolor sit amet, consetetur sadipscing elitr, sed diam nonumy eirmod tempor invidunt ut labore et dolore magna aliquyam erat, sed diam voluptua. At vero eos et accusam et justo duo dolores et ea rebum. Stet clita kasd gubergren, no sea takimata sanctus est Lorem ipsum dolor sit amet. Lorem ipsum dolor sit amet, consetetur sadipscing elitr, sed diam nonumy eirmod tempor invidunt ut labore et dolore magna aliquyam erat, sed diam voluptua. At vero eos et accusam et justo duo dolores et ea rebum. Stet clita kasd gubergren, no sea takimata sanctus est Lorem ipsum dolor sit amet. Lorem ipsum dolor sit amet, consetetur sadipscing elitr, sed diam nonumy eirmod tempor invidunt ut labore et dolore magna aliquyam erat, sed diam voluptua. At vero eos et accusam et justo duo dolores et ea rebum. Stet clita kasd gubergren, no sea takimata sanctus est Lorem ipsum dolor sit amet. Duis autem vel eum iriure dolor in hendrerit in vulputate velit esse molestie consequat, vel illum dolore eu feugiat nulla facilisis at vero eros et accumsan et iusto odio dignissim qui blandit praesent luptatum zzril delenit augue duis dolore te feugait nulla facilisi. Lorem ipsum dolor sit amet, consectetuer adipiscing elit, sed diam nonummy nibh euismod tincidunt ut laoreet dolore magna aliquam erat volutpat. Ut wisi enim ad minim veniam, quis nostrud exerci tation ullamcorper suscipit lobortis nisl ut aliquip ex ea commodo consequat. Duis autem vel eum iriure dolor in hendrerit in vulputate velit esse molestie consequat, vel illum dolore eu feugiat nulla facilisis at vero eros et accumsan et iusto odio dignissim qui blandit praesent luptatum zzril delenit augue duis dolore te feugait nulla facilisi. Nam liber tempor cum soluta nobis eleifend option congue nihil imperdiet doming id quod mazim placerat facer possim assum. Lorem ipsum dolor sit amet, consectetuer adipiscing elit, sed diam nonummy nibh euismod tincidunt ut laoreet dolore magna aliquam erat volutpat. Ut wisi enim ad minim veniam, quis nostrud exerci tation ullamcorper suscipit lobortis nisl ut aliquip ex ea commodo consequat. Duis autem vel eum iriure dolor in hendrerit in vulputate velit esse molestie consequat, vel illum dolore eu feugiat nulla facilisis. At vero eos et accusam et justo duo dolores et ea rebum. Stet clita kasd gubergren, no sea takimata sanctus est Lorem ipsum dolor sit amet. Lorem ipsum dolor sit amet, consetetur sadipscing elitr, sed diam nonumy eirmod tempor invidunt ut labore et dolore magna aliquyam erat, sed diam voluptua. At vero eos et accusam et justo duo dolores et ea rebum. Stet clita kasd gubergren, no sea takimata sanctus est Lorem ipsum dolor sit amet. Lorem ipsum dolor sit amet, consetetur sadipscing elitr, At accusam aliquyam diam diam dolore dolores duo eirmod eos erat, et nonumy sed tempor et et invidunt justo labore Stet clita ea et gubergren, kasd magna no rebum. sanctus sea sed takimata ut vero voluptua. est Lorem ipsum dolor sit amet. Lorem ipsum dolor sit amet, consetetur",
			},
		},
	}
	if err := CreateRecipe(recipe); err != nil {
		log.Printf("Failed to create recipe: %s", err)
		t.FailNow()
	}
	log.Printf("Recipe created")
	// Now checking again by reading the recipe
	dbRecipe, err := GetRecipe(recipe.ReceiptId, recipe.CreatedBy)
	if err != nil {
		log.Printf("Failed to read created recipe: %s", err)
		t.FailNow()
	}
	if recipe.CreatedAt != dbRecipe.CreatedAt || recipe.LastUpdate != dbRecipe.LastUpdate || len(recipe.Description) != len(dbRecipe.Description) || len(recipe.Ingredients) != len(dbRecipe.Ingredients) {
		log.Printf("The retrieved recipe and the created one do not match")
		t.FailNow()
	}

}