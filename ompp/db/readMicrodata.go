// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package db

import (
	"database/sql"
	"errors"
	"strconv"
)

// ReadMicrodataTo read entity microdata rows (microdata key, attributes) from model run results and process each row by cvtTo().
func ReadMicrodataTo(dbConn *sql.DB, modelDef *ModelMeta, layout *ReadMicroLayout, cvtTo func(src interface{}) (bool, error)) (*ReadPageLayout, error) {

	// validate parameters
	if modelDef == nil {
		return nil, errors.New("invalid (empty) model metadata, look like model not found")
	}
	if layout == nil {
		return nil, errors.New("invalid (empty) page layout")
	}
	if layout.Name == "" {
		return nil, errors.New("invalid (empty) parameter name")
	}

	// find entity by name
	eIdx, ok := modelDef.EntityByName(layout.Name)
	if !ok {
		return nil, errors.New("entity not found: " + layout.Name)
	}
	entity := &modelDef.Entity[eIdx]

	// check if model run exist and model run completed
	runRow, err := GetRun(dbConn, layout.FromId)
	if err != nil {
		return nil, err
	}
	if runRow == nil {
		return nil, errors.New("model run not found, id: " + strconv.Itoa(layout.FromId))
	}
	if runRow.Status != DoneRunStatus {
		return nil, errors.New("model run not completed successfully, id: " + strconv.Itoa(layout.FromId))
	}

	// find entity generation and generation attributes
	egLst, err := GetEntityGenList(dbConn, layout.FromId)
	if err != nil {
		return nil, err
	}
	var entGen *EntityGenMeta

	for k := range egLst {
		if egLst[k].GenDigest == layout.GenDigest {
			entGen = &egLst[k]
			break
		}
	}
	if entGen == nil {
		return nil, errors.New("model run does not contain entity generation: " + layout.GenDigest + " " + entity.Name + " in run, id: " + strconv.Itoa(layout.FromId))
	}

	entityAttrs := make([]EntityAttrRow, len(entGen.GenAttr))

	for k, ga := range entGen.GenAttr {

		aIdx, isOk := entity.AttrByKey(ga.AttrId)
		if !isOk {
			return nil, errors.New("entity attribute id not found: " + strconv.Itoa(ga.AttrId) + " " + entity.Name)
		}
		entityAttrs[k] = entity.Attr[aIdx]
	}

	// make sql to select microdata from model run:
	// 	 SELECT entity_key, attr4, attr7
	//   FROM Person_g87abcdef
	//   WHERE run_id = (SELECT base_run_id FROM run_entity WHERE run_id = 1234 AND entity_gen_hid = 1)
	//   ORDER BY 1, 2
	//
	q := "SELECT entity_key "

	for _, ea := range entityAttrs {
		q += ", " + ea.colName
	}

	q += " FROM " + entGen.DbEntityTable +
		" WHERE run_id =" +
		" (SELECT base_run_id FROM run_entity" +
		" WHERE run_id = " + strconv.Itoa(layout.FromId) +
		" AND entity_gen_hid = " + strconv.Itoa(entGen.GenHid) + ")"

	// append attribute enum code filters, if specified
	for k := range layout.Filter {

		// find attribute index by name
		aIdx := -1
		for j := range entityAttrs {
			if entityAttrs[j].Name == layout.Filter[k].Name {
				aIdx = j
				break
			}
		}
		if aIdx < 0 {
			return nil, errors.New("entity " + entity.Name + " does not have attribute " + layout.Filter[k].Name)
		}

		f, err := makeWhereFilter(
			&layout.Filter[k], "", entityAttrs[aIdx].colName, entityAttrs[aIdx].typeOf, false, entityAttrs[aIdx].Name, "entity "+entity.Name)
		if err != nil {
			return nil, err
		}

		q += " AND " + f
	}

	// append attribute enum id filters, if specified
	for k := range layout.FilterById {

		// find attribute index by name
		aIdx := -1
		for j := range entityAttrs {
			if entityAttrs[j].Name == layout.FilterById[k].Name {
				aIdx = j
				break
			}
		}
		if aIdx < 0 {
			return nil, errors.New("entity " + entity.Name + " does not have attribute " + layout.FilterById[k].Name)
		}

		f, err := makeWhereIdFilter(
			&layout.FilterById[k], "", entityAttrs[aIdx].colName, entityAttrs[aIdx].typeOf, entityAttrs[aIdx].Name, "entity "+entity.Name)
		if err != nil {
			return nil, err
		}

		q += " AND " + f
	}

	// append order by
	q += makeOrderBy(0, layout.OrderBy, 1)

	// prepare db-row scan conversion buffer: entity key, attributes value
	// and define conversion function to make new cell from scan buffer
	scanBuf, fc := scanSqlRowToCellMicro(entity, entityAttrs)

	// adjust page layout: starting offset and page size
	nStart := layout.Offset
	if nStart < 0 {
		nStart = 0
	}
	nSize := layout.Size
	if nSize < 0 {
		nSize = 0
	}
	var nRow int64

	lt := ReadPageLayout{
		Offset:     nStart,
		Size:       0,
		IsLastPage: false,
	}

	// select microdata cells: (entity key, attributes value)
	err = SelectRowsTo(dbConn, q,
		func(rows *sql.Rows) (bool, error) {

			// if page size is limited then select only a page of rows
			nRow++
			if nSize > 0 && nRow > nStart+nSize {
				return false, nil
			}
			if nRow <= nStart {
				return true, nil
			}

			// select next row
			if e := rows.Scan(scanBuf...); e != nil {
				return false, e
			}
			lt.Size++

			// make new cell from conversion buffer
			c := CellMicro{Attrs: make([]attrValue, len(entityAttrs))}

			if e := fc(&c); e != nil {
				return false, e
			}

			return cvtTo(c) // process cell
		})
	if err != nil && err != sql.ErrNoRows { // microdata not found is not an error
		return nil, err
	}

	// check for the empty result page or last page
	if lt.Size <= 0 {
		lt.Offset = nRow
	}
	lt.IsLastPage = nSize <= 0 || nSize > 0 && nRow <= nStart+nSize

	return &lt, nil
}

// trxReadMicrodataTo read ead entity microdata rows (microdata key, attributes) from workset or model run results and process each row by cvtTo().
func trxReadMicrodataTo(trx *sql.Tx, entity *EntityMeta, entityAttrs []EntityAttrRow, query string, cvtTo func(src interface{}) error) error {

	// select microdata cells: (microdata key, attributes)
	scanBuf, fc := scanSqlRowToCellMicro(entity, entityAttrs)

	err := TrxSelectRows(trx, query,
		func(rows *sql.Rows) error {

			// select next row
			if e := rows.Scan(scanBuf...); e != nil {
				return e
			}

			// make new cell from conversion buffer
			c := CellMicro{Attrs: make([]attrValue, len(entityAttrs))}

			if e := fc(&c); e != nil {
				return e
			}

			return cvtTo(c) // process cell
		})
	return err
}

// prepare to scan sql rows and convert each row to CellMicro
// retun scan buffer to be popualted by rows.Scan() and closure to that buffer into CellMicro
func scanSqlRowToCellMicro(entity *EntityMeta, entityAttrs []EntityAttrRow) ([]interface{}, func(*CellMicro) error) {

	nAttr := len(entityAttrs)
	scanBuf := make([]interface{}, 1+nAttr) // entity key and attributes

	var eKey uint64
	scanBuf[0] = &eKey // first column is entity key

	fd := make([]func(interface{}) (attrValue, error), nAttr) // conversion functions for all attributes

	// for each attribute create conversion function by type
	for na, ea := range entityAttrs {

		switch {
		case ea.typeOf.IsBool(): // logical attribute

			var v interface{}
			scanBuf[1+na] = &v

			fd[na] = func(src interface{}) (attrValue, error) {

				av := attrValue{}
				av.IsNull = false // logical attribute expected to be NOT NULL

				is := false
				switch vn := v.(type) {
				case nil: // 2018: unexpected today, may be in the future
					is = false
					av.IsNull = true
				case bool:
					is = vn
				case int64:
					is = vn != 0
				case uint64:
					is = vn != 0
				case int32:
					is = vn != 0
				case uint32:
					is = vn != 0
				case int16:
					is = vn != 0
				case uint16:
					is = vn != 0
				case int8:
					is = vn != 0
				case uint8:
					is = vn != 0
				case uint:
					is = vn != 0
				case float32: // oracle (very unlikely)
					is = vn != 0.0
				case float64: // oracle (often)
					is = vn != 0.0
				case int:
					is = vn != 0
				default:
					return av, errors.New("invalid attribute value type, integer expected: " + entity.Name + "." + ea.Name)
				}
				av.Value = is

				return av, nil
			}

		case ea.typeOf.IsString(): // string attribute

			var vs string
			scanBuf[1+na] = &vs

			fd[na] = func(src interface{}) (attrValue, error) {

				if src == nil {
					return attrValue{IsNull: true}, nil
				}
				return attrValue{IsNull: false, Value: vs}, nil
			}

		case ea.typeOf.IsFloat(): // float attribute, can be NULL

			var vf sql.NullFloat64
			scanBuf[1+na] = &vf

			fd[na] = func(src interface{}) (attrValue, error) {

				if src == nil {
					return attrValue{IsNull: true}, nil
				}
				if vf.Valid {
					return attrValue{IsNull: false, Value: vf.Float64}, nil
				}
				return attrValue{IsNull: true, Value: 0.0}, nil
			}

		default:
			var v interface{}
			scanBuf[1+na] = &v

			fd[na] = func(src interface{}) (attrValue, error) { return attrValue{IsNull: src == nil, Value: v}, nil }
		}

	}

	// sql row conevrsion function: convert entity key and each attribute value from scan buffer
	cvt := func(c *CellMicro) error {

		c.Key = eKey

		for k := 0; k < nAttr; k++ {
			v, e := fd[k](scanBuf[1+k])
			if e != nil {
				return e
			}
			c.Attrs[k] = v
		}
		return nil
	}

	return scanBuf, cvt
}
