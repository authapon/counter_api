package main

import (
	"log"
	"net/http"
	"sync"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"github.com/pocketbase/pocketbase/models"
	"github.com/pocketbase/pocketbase/models/schema"
	"github.com/pocketbase/pocketbase/tools/types"
)

type (
	Counter struct {
		Name  string `db:"name" json:"name"`
		Count int64  `db:"count" json:"count"`
	}
)

var (
	csync sync.Mutex
	app   *pocketbase.PocketBase
)

func main() {
	app = pocketbase.New()

	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		createCollection()
		createRouter(e)
		return nil
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}

func createCollection() {
	collection := &models.Collection{
		Name:       "counter",
		Type:       models.CollectionTypeBase,
		ListRule:   nil,
		ViewRule:   nil,
		CreateRule: nil,
		UpdateRule: nil,
		DeleteRule: nil,
		Schema: schema.NewSchema(
			&schema.SchemaField{
				Name:     "name",
				Type:     schema.FieldTypeText,
				Required: true,
			},
			&schema.SchemaField{
				Name:     "count",
				Type:     schema.FieldTypeNumber,
				Required: true,
			},
		),
		Indexes: types.JsonArray[string]{
			"CREATE UNIQUE INDEX unique_counter_name ON counter (name)",
		},
	}

	app.Dao().SaveCollection(collection)
}

func createRouter(e *core.ServeEvent) {
	e.Router.GET("/counter/:name", func(c echo.Context) error {
		name := c.PathParam("name")

		csync.Lock()
		defer csync.Unlock()

		counter := []Counter{}
		count := int64(1)
		app.Dao().DB().Select("count").From("counter").Where(dbx.NewExp("name = {:name}", dbx.Params{"name": name})).All(&counter)
		if len(counter) == 0 {
			app.Dao().DB().Insert("counter", dbx.Params{
				"name":  name,
				"count": 1,
			}).Execute()
		} else {
			count = counter[0].Count + 1
			app.Dao().DB().Update("counter",
				dbx.Params{
					"count": count,
				},
				dbx.NewExp("name = {:name}", dbx.Params{"name": name})).Execute()
		}
		return c.JSON(http.StatusOK, map[string]int64{"counter": count})
	})

	e.Router.GET("/counter/:name/get", func(c echo.Context) error {
		name := c.PathParam("name")

		csync.Lock()
		defer csync.Unlock()

		counter := []Counter{}
		count := int64(1)
		app.Dao().DB().Select("count").From("counter").Where(dbx.NewExp("name = {:name}", dbx.Params{"name": name})).All(&counter)

		if len(counter) == 0 {
			count = 0
		} else {
			count = counter[0].Count
		}

		return c.JSON(http.StatusOK, map[string]int64{"counter": count})
	})
}
