package console

import (
	"encoding/json"
	"net/http"
)

func writeJSON(writer http.ResponseWriter, value any) {
	data, err := json.Marshal(value)
	if err != nil {
		http.Error(writer, "response unavailable", http.StatusInternalServerError)

		return
	}

	writer.Header().Set("Content-Type", "application/json")
	_, _ = writer.Write(data)
}
