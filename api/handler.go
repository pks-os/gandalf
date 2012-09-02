package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/timeredbull/gandalf/db"
	"io"
	"io/ioutil"
	"labix.org/v2/mgo/bson"
	"net/http"
	"reflect"
)

func GrantAccess(w http.ResponseWriter, r *http.Request) {
	repo := repository{Name: r.URL.Query().Get(":name")}
	c := db.Session.Repository()
	c.Find(bson.M{"_id": repo.Name}).One(&repo)
	req := map[string][]string{}
	err := parseBody(r.Body, &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for _, u := range req["users"] {
		_, err = getUserOr404(u)
		if err != nil {
			if len(req["users"]) == 1 {
				// user should exist, return error
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			} else {
				// #TODO (flaviamissi): log a warning saying the user "u" was not found and skip it
				continue
			}
		}
		repo.Users = append(repo.Users, u)
	}
	err = c.Update(bson.M{"_id": repo.Name}, &repo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func AddKey(w http.ResponseWriter, r *http.Request) {
	u := user{Name: r.URL.Query().Get(":name")}
	c := db.Session.User()
	err := c.Find(bson.M{"_id": u.Name}).One(&u)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}
	params := map[string]string{}
	err = parseBody(r.Body, &params)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if params["key"] == "" {
		http.Error(w, "A key is needed", http.StatusBadRequest)
		return
	}
	u.Keys = append(u.Keys, params["key"])
	err = c.Update(bson.M{"_id": u.Name}, u)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "Key \"%s\" successfuly created", params["key"])
}

func NewUser(w http.ResponseWriter, r *http.Request) {
	var u user
	err := parseBody(r.Body, &u)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if u.Name == "" {
		http.Error(w, "User needs a name", http.StatusBadRequest)
		return
	}
	err = db.Session.User().Insert(&u)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "User %s successfuly created", u.Name)
}

func NewRepository(w http.ResponseWriter, r *http.Request) {
	var p repository
	err := parseBody(r.Body, &p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// #TODO (flaviamissi) ensure repository name is a valid directory name
	if p.Name == "" {
		http.Error(w, "Repository needs a name", http.StatusBadRequest)
		return
	}
	if len(p.Users) == 0 {
		http.Error(w, "Repository needs a user", http.StatusBadRequest)
		return
	}
	err = db.Session.Repository().Insert(&p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "Repository %s successfuly created", p.Name)
}

func parseBody(body io.ReadCloser, result interface{}) error {
	if reflect.ValueOf(result).Kind() == reflect.Struct {
		return errors.New("parseBody function cannot deal with struct. Use pointer")
	}
	b, err := ioutil.ReadAll(body)
	if err != nil {
		return err
	}
	err = json.Unmarshal(b, &result)
	if err != nil {
		e := fmt.Sprintf("Could not parse json: %s", err.Error())
		return errors.New(e)
	}
	return nil
}
