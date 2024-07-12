package extensions

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/sprungknoedl/dagobert/internal/model"
)

func RunPlaso(store model.Store, obj model.Evidence) error {
	name := strings.TrimSuffix(obj.Name, filepath.Ext(obj.Name))
	dst := filepath.Join("files", "evidences", obj.CaseID, name+".plaso")
	src, err := clone(obj)
	if err != nil {
		return err
	}
	defer os.Remove(src)

	err = runDocker(src, dst, "log2timeline/plaso", []string{
		"psteal.py",
		"--unattended",
		// CDQR 'datt' parser set
		"--parsers", "text/bash_history,bencode,czip,esedb,filestat,lnk,mcafee_protection,olecf,pe,prefetch,recycle_bin,recycle_bin_info2,text/sccm,text/sophos_av,sqlite,symantec_scanlog,winevt,winevtx,webhist,text/winfirewall,winjob,winreg,text/zsh_extended_history",
		"--output-format", "dynamic",
		"--source", "/in/" + filepath.Base(src),
		"--storage-file", "/out/" + filepath.Base(dst),
		"--write", "/out/" + filepath.Base(dst) + ".csv",
	})

	if err != nil {
		// try to clean up
		os.Remove(dst)
		os.Remove(dst + ".csv")
		return err
	}

	if err := addFromFS(store, model.Evidence{
		Type:     "Other",
		Name:     filepath.Base(dst),
		Source:   obj.Source,
		Notes:    "ext-plaso",
		Location: filepath.Base(dst),
		CaseID:   obj.CaseID,
	}); err != nil {
		return err
	}

	if err := addFromFS(store, model.Evidence{
		Type:     "Other",
		Name:     filepath.Base(dst) + ".csv",
		Source:   obj.Source,
		Notes:    "ext-plaso",
		Location: filepath.Base(dst) + ".csv",
		CaseID:   obj.CaseID,
	}); err != nil {
		return err
	}

	return nil
}
