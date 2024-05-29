// Copyright OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package main

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strconv"

	_ "github.com/mattn/go-sqlite3"

	"github.com/openmpp/go/ompp/db"
	"github.com/openmpp/go/ompp/omppLog"
)

// write model list from database into text csv or json file
func modelList(srcDb *sql.DB) error {

	// get model list
	mLst, err := db.GetModelList(srcDb)
	if err != nil {
		return err
	}
	if len(mLst) <= 0 {
		omppLog.Log("Database is empty, models not found")
		return nil
	}

	// use specified file name or make default
	fp := ""

	if theCfg.isConsole {
		omppLog.Log("Do model-list")
	} else {

		fp = theCfg.fileName
		if fp == "" {
			fp = "model-list" + outputExt()
		}
		fp = filepath.Join(theCfg.dir, fp)

		omppLog.Log("Do model-list: " + fp)
	}

	// write json output into file or console
	if theCfg.kind == asJson {

		type mItem struct {
			Model     db.ModelDicRow
			DescrNote db.DescrNote
		}
		mtLst := []mItem{}

		for k := range mLst {

			mt := mItem{
				Model:     mLst[k],
				DescrNote: db.DescrNote{},
			}

			// append description and notes if any exist
			lc := ""
			if !theCfg.isNoLang && theCfg.userLang != "" {

				lc, err = matchUserLang(srcDb, mLst[k])
				if err != nil {
					return err
				}
			}
			if theCfg.isNoLang || lc == "" {
				lc = mLst[k].DefaultLangCode
				omppLog.Log("Using default model language: ", lc)
			}
			if lc != "" {
				txt, e := db.GetModelTextRowById(srcDb, mLst[k].ModelId, lc)
				if e != nil {
					return e // error at model_dic_txt select
				}
				if len(txt) > 0 && txt[0].LangCode != "" && (txt[0].Descr != "" || txt[0].Note != "") {
					mt.DescrNote.LangCode = txt[0].LangCode
					mt.DescrNote.Descr = txt[0].Descr
					mt.DescrNote.Note = txt[0].Note
				}
			}
			mtLst = append(mtLst, mt)
		}

		return toJsonOutput(fp, mtLst) // save results
	}
	// else write csv or tsv output into file or console

	// write model master row into csv, including description
	type noteItem struct {
		name string // model name
		lang string // language code
		note string // notes
	}
	noteLst := []noteItem{}

	row := make([]string, 9)

	idx := 0
	err = toCsvOutput(
		fp,
		[]string{"model_id", "model_name", "model_digest", "model_type", "model_ver", "create_dt", "default_lang_code", "lang_code", "descr"},
		func() (bool, []string, error) {
			if 0 <= idx && idx < len(mLst) {
				row[0] = strconv.Itoa(mLst[idx].ModelId)
				row[1] = mLst[idx].Name
				row[2] = mLst[idx].Digest
				row[3] = strconv.Itoa(mLst[idx].Type)
				row[4] = mLst[idx].Version
				row[5] = mLst[idx].CreateDateTime
				row[6] = mLst[idx].DefaultLangCode
				row[7] = ""
				row[8] = ""

				// append description to the row and save notes if any exist
				lc := ""
				var e error
				if !theCfg.isNoLang && theCfg.userLang != "" {

					lc, e = matchUserLang(srcDb, mLst[idx])
					if e != nil {
						return true, row, e // error at language match or lang_dic select
					}
				}
				if theCfg.isNoLang || lc == "" {
					lc = mLst[idx].DefaultLangCode
					omppLog.Log("Using default model language: ", lc)
				}
				if lc != "" {
					txt, e := db.GetModelTextRowById(srcDb, mLst[idx].ModelId, lc)
					if e != nil {
						return true, row, e // error at model_dic_txt select
					}
					if len(txt) > 0 {
						row[7] = txt[0].LangCode
						row[8] = txt[0].Descr

						if theCfg.isNote && txt[0].LangCode != "" && txt[0].Note != "" {
							noteLst = append(noteLst, noteItem{name: mLst[idx].Name, lang: txt[0].LangCode, note: txt[0].Note})
						}
					}
				}

				idx++
				return false, row, nil
			}
			return true, row, nil // end of model_dic rows
		})
	if err != nil {
		return errors.New("failed to write model into csv " + err.Error())
	}

	// write notes into .md files
	if theCfg.isNote {
		for k := range noteLst {

			p := filepath.Join(theCfg.dir, noteLst[k].name+"."+noteLst[k].lang+".md")

			err = os.WriteFile(p, []byte(noteLst[k].note), 0644)
			if err != nil {
				return errors.New("failed to write model model notes " + err.Error())
			}
		}
	}

	return nil
}


// Not sure if we want a database handle as argument or stuff coming from cfg variables.
// We will need to initialize theCatalog as oms does it.
// Then call getModelMetaWithText with language and model name arguments coming from theCfg.
// Then handle the return value according output option picked in theCfg as well.
func getModelMetadata( ) error {

}

