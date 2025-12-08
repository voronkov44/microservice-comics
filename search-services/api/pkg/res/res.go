package res

import (
	"encoding/json"
	"log"
	"net/http"
)

func Json(w http.ResponseWriter, data interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	enc := json.NewEncoder(w)
	if err := enc.Encode(data); err != nil {
		log.Printf("write json failed: %v", err)
	}

}
