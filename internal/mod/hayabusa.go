package mod

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sprungknoedl/dagobert/internal/fp"
	"github.com/sprungknoedl/dagobert/internal/model"
)

func init() {
	Register(Mod{
		Name:        "Hayabusa",
		Description: "Hayabusa (隼) is a sigma-based threat hunting and fast forensics timeline generator for Windows event logs.",
		Supports:    func(e model.Evidence) bool { return filepath.Ext(e.Name) == ".zip" || filepath.Ext(e.Name) == ".evtx" },
		Run:         RunHayabusa,
	})

	Register(Mod{
		Name:        "Ingest Hayabusa Timeline",
		Description: "Ingest high and critical alerts of the timeline generated by Hayabusa.",
		Supports:    func(e model.Evidence) bool { return strings.HasSuffix(e.Name, ".hayabusa.jsonl") },
		Run:         IngestHayabusa,
	})
}

func RunHayabusa(store *model.Store, obj model.Evidence) error {
	zip := filepath.Ext(obj.Name) == ".zip"
	name := strings.TrimSuffix(obj.Name, filepath.Ext(obj.Name))
	dst := filepath.Join("files", "evidences", obj.CaseID, name+".hayabusa.jsonl")

	var src string
	var err error
	if zip {
		src, err = unpack(obj)
		if err != nil {
			return err
		}
		defer os.RemoveAll(src)
	} else {
		src, err = clone(obj)
		if err != nil {
			return err
		}
		defer os.Remove(src)
	}

	if err := runDocker(src, dst, "sprungknoedl/hayabusa", []string{
		"json-timeline",
		"--JSONL-output",
		"--RFC-3339",
		"--UTC",
		"--no-wizard",
		"--min-level", "informational",
		"--profile", "timesketch-verbose",
		fp.If(zip, "--directory", "--file"), fp.If(zip, "/in/", "/in/"+filepath.Base(src)),
		"--output", "/out/" + filepath.Base(dst),
	}); err != nil {
		// try to clean up
		os.Remove(dst)
		return err
	}

	if err := AddFromFS("Hayabusa", store, model.Evidence{
		ID:       random(10),
		CaseID:   obj.CaseID,
		Type:     "Logs",
		Name:     filepath.Base(dst),
		Source:   obj.Source,
		Notes:    "ext-hayabusa",
		Location: filepath.Base(dst),
	}); err != nil {
		return err
	}

	return nil
}

type HayabusaRecord struct {
	Datetime       string `json:"datetime"`
	TimestampDesc  string `json:"timestamp_desc"`
	Message        string `json:"message"`
	Level          string
	Computer       string
	Channel        string
	EventID        int
	MitreTactics   []string
	RecordID       int
	Details        json.RawMessage
	ExtraFieldInfo json.RawMessage
	RuleFile       string
	EvtxFile       string
}

func IngestHayabusa(store *model.Store, obj model.Evidence) error {
	src := filepath.Join("files", "evidences", obj.CaseID, obj.Location)
	fh, err := os.Open(src)
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(fh)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		record := HayabusaRecord{}
		err := json.Unmarshal(scanner.Bytes(), &record)
		if err != nil {
			return err
		}

		if record.Level != "high" && record.Level != "critical" {
			continue
		}

		// map computer to assets
		asset, _ := store.GetAssetByName(obj.CaseID, record.Computer)
		if asset.ID == "" {
			asset := model.Asset{
				ID:     random(10),
				CaseID: obj.CaseID,
				Name:   record.Computer,
				Status: "Under investigation",
				Type:   "Other",
			}
			if err := store.SaveAsset(obj.CaseID, asset); err != nil {
				return err
			}

			store.SaveAuditlog(model.User{Name: "Hayabusa", UPN: "Extension"}, model.Case{}, "asset:"+asset.ID, fmt.Sprintf("Added asset (hayabusa ingest) %q", obj.Name))
		}

		// translate mitre tactics
		translator := map[string]string{
			"Recon":      "Reconnaissance",
			"ResDev":     "Resource Development",
			"InitAccess": "Initial Access",
			"Exec":       "Execution",
			"Persis":     "Persistence",
			"PrivEsc":    "Privilege Escalation",
			"Evas":       "Defense Evasion",
			"CredAccess": "Credential Access",
			"Disc":       "Discovery",
			"LatMov":     "Lateral Movement",
			"Collect":    "Collection",
			"C2":         "Command and Control",
			"Exfil":      "Exfiltration",
			"Impact":     "Impact",
		}

		t, err := time.Parse("2006-01-02 15:04:05.000000Z07:00", record.Datetime)
		if err != nil {
			return err
		}

		e := model.Event{
			ID:     random(10),
			CaseID: obj.CaseID,
			Time:   model.Time(t),
			Type:   translator[record.MitreTactics[0]],
			Event:  record.Message,
			Raw:    scanner.Text(),
			Assets: []model.Asset{asset},
		}

		if err := store.SaveEvent(obj.CaseID, e, true); err != nil {
			return err
		}

		store.SaveAuditlog(model.User{Name: "Hayabusa", UPN: "Extension"}, model.Case{}, "event:"+obj.ID, fmt.Sprintf("Added event %q", e.Event))
	}

	return nil
}
