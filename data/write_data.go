package data

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"zuri.chat/zccore/utils"
)

type writeDataRequest struct {
	PluginID       string      `json:"plugin_id"`
	CollectionName string      `json:"collection_name"`
	OrganizationID string      `json:"organization_id"`
	BulkWrite      bool        `json:"bulk_write"`
	ObjectID       string      `json:"object_id,omitempty"`
	ObjectIDs      []string    `json:"object_ids,omitempty"`
	Payload        interface{} `json:"payload,omitempty"`
}

func WriteData(w http.ResponseWriter, r *http.Request) {
	reqData := new(writeDataRequest)
	if err := json.NewDecoder(r.Body).Decode(reqData); err != nil {
		utils.GetError(fmt.Errorf("error processing request: %v", err), http.StatusUnprocessableEntity, w)
		return
	}

	if !recordExists("plugins", reqData.PluginID) {
		msg := "plugin with this id does not exist"
		utils.GetError(errors.New(msg), http.StatusNotFound, w)
		return
	}

	if !recordExists("organization", reqData.OrganizationID) {
		// organization with this id does not exist
		msg := "organization with this id does not exist"
		utils.GetError(errors.New(msg), http.StatusNotFound, w)
		return
	}

	// if plugin is accessing this collection the first time, we create a record linking this collection to the plugin.
	if !pluginHasCollection(reqData.PluginID, reqData.OrganizationID, reqData.CollectionName) {
		createPluginCollectionRecord(reqData.PluginID, reqData.OrganizationID, reqData.CollectionName)
	}

	switch r.Method {
	case "POST":
		reqData.handlePost(w, r)
	case "PUT":
		reqData.handlePut(w, r)
	case "DELETE":
		reqData.handleDelete(w, r)
	}
}

func (wdr *writeDataRequest) handlePost(w http.ResponseWriter, r *http.Request) {
	var err error
	writeCount := 0
	if wdr.BulkWrite {
		writeCount, err = insertMany(wdr.prefixCollectionName(), wdr.Payload)
		if err != nil {
			utils.GetError(fmt.Errorf("an error occured: %v", err), http.StatusInternalServerError, w)
			return
		}
	} else {
		if err := insertOne(wdr.prefixCollectionName(), wdr.Payload); err != nil {
			utils.GetError(fmt.Errorf("an error occured: %v", err), http.StatusInternalServerError, w)
			return
		}
		writeCount = 1
	}

	w.WriteHeader(http.StatusCreated)
	utils.GetSuccess("success", M{"insert_count": writeCount}, w)
}

func (wdr *writeDataRequest) handlePut(w http.ResponseWriter, r *http.Request) {
	if wdr.CollectionName == "" || wdr.PluginID == "" {
		utils.GetError(errors.New("invalid data destination"), http.StatusBadRequest, w)
		return
	}
	var err error
	writeCount := 0
	if wdr.BulkWrite {
		writeCount, err = updateMany(wdr.prefixCollectionName(), wdr.ObjectIDs, wdr.Payload)
		if err != nil {
			utils.GetError(fmt.Errorf("an error occured: %v", err), http.StatusInternalServerError, w)
			return
		}
	} else {
		if err := updateOne(wdr.prefixCollectionName(), wdr.ObjectID, wdr.Payload); err != nil {
			utils.GetError(fmt.Errorf("an error occured: %v", err), http.StatusInternalServerError, w)
			return
		}
		writeCount = 1
	}

	utils.GetSuccess("success", M{"update_count": writeCount}, w)
}

func (wdr *writeDataRequest) handleDelete(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "hello world")
}

func (wdr *writeDataRequest) prefixCollectionName() string {
	return getPrefixedCollectionName(wdr.PluginID, wdr.OrganizationID, wdr.CollectionName)
}

func insertMany(collName string, data interface{}) (int, error) {
	_, ok := data.([]interface{})
	if !ok {
		return 0, errors.New("type assertion error")
	}
	// call mongodb insert many here
	return 0, nil
}

func insertOne(collName string, data interface{}) error {
	doc, ok := data.(map[string]interface{})
	if !ok {
		return errors.New("type assertion error")
	}
	if _, err := utils.CreateMongoDbDoc(collName, doc); err != nil {
		return err
	}
	return nil
}

func updateOne(collName, id string, upd interface{}) error {
	_, ok := upd.(map[string]interface{})
	if !ok {
		return errors.New("type assertion error")
	}
	// do updateOne
	return nil
}

func updateMany(collName string, id []string, upd interface{}) (int, error) {
	_, ok := upd.([]interface{})
	if !ok {
		return 0, errors.New("type assertion error")
	}
	// do update many
	return 0, nil
}

func deleteOne(collName, id string) error {
	return nil
}

func deleteMany(collName, ids []string) (int, error) {
	return 0, nil
}
