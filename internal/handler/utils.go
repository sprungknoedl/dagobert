package handler

import (
	"encoding/csv"
	"io"
	"net/http"
	"time"

	"github.com/sprungknoedl/dagobert/internal/model"
	"github.com/sprungknoedl/dagobert/internal/utils"
)

var ZeroID string = "0"
var ZeroTime time.Time

func ImportCSV(store *model.Store, w http.ResponseWriter, r *http.Request, uri string, numFields int, cb func(rec []string)) {
	if r.Method == http.MethodGet {
		utils.Render(store, w, r, http.StatusOK, "internal/views/utils-import.html", map[string]any{})
		return
	}

	fr, _, err := r.FormFile("file")
	if err != nil {
		utils.Warn(w, r, err)
		return
	}

	cr := csv.NewReader(fr)
	cr.FieldsPerRecord = numFields
	cr.Read() // skip header

	for {
		rec, err := cr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			utils.Warn(w, r, err)
			return
		}

		cb(rec)
	}

	http.Redirect(w, r, uri, http.StatusSeeOther)
}
