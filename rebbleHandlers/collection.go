package rebbleHandlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"pebble-dev/rebblestore-api/db"
	"strconv"

	"github.com/gorilla/mux"
)

type RebbleCollection struct {
	Id    string          `json:"id"`
	Name  string          `json:"name"`
	Pages int             `json:"pages"`
	Cards []db.RebbleCard `json:"cards"`
}

func insert(apps *([]db.RebbleApplication), location int, app db.RebbleApplication) *([]db.RebbleApplication) {
	beggining := (*apps)[:location]
	end := make([]db.RebbleApplication, len(*apps)-len(beggining))
	copy(end, (*apps)[location:])
	beggining = append(beggining, app)
	beggining = append(beggining, end...)

	return &beggining
}

func remove(apps *([]db.RebbleApplication), location int) *([]db.RebbleApplication) {
	new := make([]db.RebbleApplication, location)
	copy(new, (*apps)[:location])
	new = append(new, (*apps)[location+1:]...)

	return &new
}

func in_array(s string, array []string) bool {
	for _, item := range array {
		if item == s {
			return true
		}
	}

	return false
}

func nCompatibleApps(apps *([]db.RebbleApplication), platform string) int {
	var n int
	for _, app := range *apps {
		if platform == "all" || in_array(platform, app.SupportedPlatforms) {
			n = n + 1
		}
	}

	return n
}

func bestApps(apps *([]db.RebbleApplication), sortByPopular bool, nApps int, platform string) *([]db.RebbleApplication) {
	newApps := make([]db.RebbleApplication, 0)

	for _, app := range *apps {
		if platform == "all" || in_array(platform, app.SupportedPlatforms) {
			newApps = append(newApps, app)
		}

		if len(newApps) > nApps {
			if sortByPopular {
				worst := 0
				for i, newApp := range newApps {
					if newApp.ThumbsUp < newApps[worst].ThumbsUp {
						worst = i
					}
				}
				newApps = *(remove(&newApps, worst))
			} else {
				worst := 0
				for i, newApp := range newApps {
					if newApp.Published.UnixNano() < newApps[worst].Published.UnixNano() {
						worst = i
					}
				}
				newApps = *(remove(&newApps, worst))
			}
		}
	}

	return &newApps
}

func sortApps(apps *([]db.RebbleApplication), sortByPopular bool) *([]db.RebbleApplication) {
	newApps := make([]db.RebbleApplication, 0)

	for _, app := range *apps {
		if len(newApps) == 0 {
			newApps = []db.RebbleApplication{app}

			continue
		} else if len(newApps) == 1 {
			if sortByPopular {
				if newApps[0].ThumbsUp > app.ThumbsUp {
					newApps = []db.RebbleApplication{newApps[0], app}
				} else {
					newApps = []db.RebbleApplication{app, newApps[0]}
				}
			} else {
				if newApps[0].Published.UnixNano() > app.Published.UnixNano() {
					newApps = []db.RebbleApplication{app, newApps[0]}
				} else {
					newApps = []db.RebbleApplication{newApps[0], app}
				}
			}

			continue
		}

		if sortByPopular {
			added := false
			for i, newApp := range newApps {
				if newApp.ThumbsUp < app.ThumbsUp {
					newApps = *(insert(&newApps, i, app))
					added = true
					break
				}
			}
			if !added {
				newApps = *(insert(&newApps, len(newApps), app))
			}
		} else {
			added := false
			for i, newApp := range newApps {
				if app.Published.UnixNano() > newApp.Published.UnixNano() {
					newApps = *(insert(&newApps, i, app))
					added = true
					break
				}
			}
			if !added {
				newApps = *(insert(&newApps, len(newApps), app))
			}
		}
	}

	return &newApps
}

// CollectionHandler serves a list of cards from a collection
func CollectionHandler(ctx *HandlerContext, w http.ResponseWriter, r *http.Request) (int, error) {
	urlquery := r.URL.Query()

	if _, ok := mux.Vars(r)["id"]; !ok {
		return http.StatusBadRequest, errors.New("Missing 'id' parameter")
	}

	var sortByPopular bool
	if o, ok := urlquery["order"]; ok {
		if len(o) > 1 {
			return http.StatusBadRequest, errors.New("Multiple 'order' parameters are not allowed")
		} else if o[0] == "popular" {
			sortByPopular = true
		} else if o[0] == "new" {
			sortByPopular = false
		} else {
			return http.StatusBadRequest, errors.New("Invalid 'order' parameter")
		}
	}
	platform := "all"
	if o, ok := urlquery["platform"]; ok {
		if len(o) > 1 {
			return http.StatusBadRequest, errors.New("Multiple 'platform' parameters are not allowed")
		} else if o[0] == "aplite" || o[0] == "basalt" || o[0] == "chalk" || o[0] == "diorite" {
			platform = o[0]
		} else {
			return http.StatusBadRequest, errors.New("Invalid 'platform' parameter")
		}
	}
	page := 1
	if o, ok := urlquery["page"]; ok {
		if len(o) > 1 {
			return http.StatusBadRequest, errors.New("Multiple pages not allowed")
		} else {
			var err error
			page, err = strconv.Atoi(o[0])
			if err != nil || page < 1 {
				return http.StatusBadRequest, errors.New("Parameter 'page' should be a positive, non-nul integer")
			}
		}
	}

	apps, err := ctx.Database.GetAppsForCollection(mux.Vars(r)["id"])
	if err != nil {
		return http.StatusInternalServerError, err
	}
	nCompatibleApps := nCompatibleApps(&apps, platform)
	apps = *(bestApps(&apps, sortByPopular, page*12, platform))
	apps = *(sortApps(&apps, sortByPopular))

	collectionName, err := ctx.Database.GetCollectionName(mux.Vars(r)["id"])
	if err != nil {
		return http.StatusInternalServerError, err
	}

	pages := nCompatibleApps / 12
	if nCompatibleApps%12 > 0 {
		pages = pages + 1
	}

	// Only allow to view up to 20 pages - More pages = more computation time
	if pages > 20 {
		pages = 20
	}

	collection := RebbleCollection{
		Id:    mux.Vars(r)["id"],
		Name:  collectionName,
		Pages: pages,
	}

	if page > pages {
		return http.StatusBadRequest, errors.New("Requested inexistant page number")
	}

	if page != 1 && page != pages {
		apps = apps[(page-1)*12 : page*12]
	} else if page == pages {
		apps = apps[(page-1)*12:]
	}

	for _, app := range apps {
		image := ""
		if len(*app.Assets.Screenshots) != 0 && len((*app.Assets.Screenshots)[0].Screenshots) != 0 {
			image = (*app.Assets.Screenshots)[0].Screenshots[0]
		}
		collection.Cards = append(collection.Cards, db.RebbleCard{
			Id:       app.Id,
			Title:    app.Name,
			Type:     app.Type,
			ImageUrl: image,
			ThumbsUp: app.ThumbsUp,
		})
	}

	data, err := json.MarshalIndent(collection, "", "\t")
	if err != nil {
		return http.StatusInternalServerError, err
	}

	// Send the JSON object back to the user
	w.Header().Add("content-type", "application/json")
	w.Write(data)

	return http.StatusOK, nil
}
