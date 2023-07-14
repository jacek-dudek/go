// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package main

import (
	"github.com/openmpp/go/ompp/db"
	"github.com/openmpp/go/ompp/omppLog"
)

// ParameterCellConverter return parameter value converter between code cell and id's cell.
// If isToId true then from code to id cell else other way around
func (mc *ModelCatalog) ParameterCellConverter(
	isToId bool, dn string, name string,
) (
	func(interface{}) (interface{}, error), bool,
) {

	// if model digest-or-name is empty then return empty results
	if dn == "" {
		omppLog.Log("Warning: invalid (empty) model digest and name")
		return nil, false
	}

	// get model metadata and database connection
	meta, _, ok := mc.modelMeta(dn)
	if !ok {
		omppLog.Log("Warning: model digest or name not found: ", dn)
		return nil, false // return empty result: model not found or error
	}

	// check if parameter name exist in the model
	if _, ok = meta.ParamByName(name); !ok {
		omppLog.Log("Error: model parameter not found: ", dn, ": ", name)
		return nil, false
	}

	// create converter
	csvCvt := db.CellParamConverter{
		ModelDef:  meta,
		Name:      name,
		DoubleFmt: theCfg.doubleFmt,
	}
	var cvt func(interface{}) (interface{}, error)
	var err error

	if isToId {
		cvt, err = csvCvt.CodeToIdCell(meta, name)
	} else {
		cvt, err = csvCvt.IdToCodeCell(meta, name)
	}
	if err != nil {
		omppLog.Log("Failed to create parameter cell value converter: ", name, ": ", err.Error())
		return nil, false
	}

	return cvt, true
}

// TableToCodeCellConverter return output table value converter from id's cell into code cell.
func (mc *ModelCatalog) TableToCodeCellConverter(dn string, name string, isAcc, isAllAcc bool) (func(interface{}) (interface{}, error), bool) {

	// if model digest-or-name is empty then return empty results
	if dn == "" {
		omppLog.Log("Warning: invalid (empty) model digest and name")
		return nil, false
	}

	// get model metadata and database connection
	meta, _, ok := mc.modelMeta(dn)
	if !ok {
		omppLog.Log("Warning: model digest or name not found: ", dn)
		return nil, false // return empty result: model not found or error
	}

	// check if output table name exist in the model
	if _, ok = meta.OutTableByName(name); !ok {
		omppLog.Log("Error: model output table not found: ", dn, ": ", name)
		return nil, false
	}

	// create converter
	ctc := db.CellTableConverter{
		ModelDef: meta,
		Name:     name,
	}
	var cvt func(interface{}) (interface{}, error)
	var err error

	switch {
	case isAcc && isAllAcc:
		csvCvt := db.CellAllAccConverter{
			CellTableConverter: ctc,
			IsIdCsv:            false,
			DoubleFmt:          theCfg.doubleFmt,
			ValueName:          "",
		}
		cvt, err = csvCvt.IdToCodeCell(meta, name)
	case isAcc:
		csvCvt := db.CellAccConverter{
			CellTableConverter: ctc,
			IsIdCsv:            false,
			DoubleFmt:          theCfg.doubleFmt,
		}
		cvt, err = csvCvt.IdToCodeCell(meta, name)
	default:
		csvCvt := db.CellExprConverter{
			CellTableConverter: ctc,
			IsIdCsv:            false,
			DoubleFmt:          theCfg.doubleFmt,
		}
		cvt, err = csvCvt.IdToCodeCell(meta, name)
	}
	if err != nil {
		omppLog.Log("Failed to create output table cell id's to code converter: ", name, ": ", err.Error())
		return nil, false
	}

	return cvt, true
}

// TableToCodeCalcCellConverter return output table calculated value converter from id's cell into code cell and run id(s).
// Function accept base run digest-or-stamp-or-name and optional list of variant runs digest-or-stamp-or-name.
// All runs must be completed sucessfully.
func (mc *ModelCatalog) TableToCodeCalcCellConverter(
	dn string, rdsn string, tableName string, runLst []string,
) (func(interface{}) (interface{}, error), int, []int, bool) {

	// if model digest-or-name is empty then return empty results
	if dn == "" {
		omppLog.Log("Warning: invalid (empty) model digest and name")
		return nil, 0, nil, false
	}

	// get model metadata and database connection
	meta, _, ok := mc.modelMeta(dn)
	if !ok {
		omppLog.Log("Warning: model digest or name not found: ", dn)
		return nil, 0, nil, false // return empty result: model not found or error
	}

	// check if output table name exist in the model
	if _, ok = meta.OutTableByName(tableName); !ok {
		omppLog.Log("Error: model output table not found: ", dn, ": ", tableName)
		return nil, 0, nil, false
	}

	// create converter
	csvCvt := db.CellTableCalcConverter{
		CellTableConverter: db.CellTableConverter{
			ModelDef: meta,
			Name:     tableName,
		},
		IsIdCsv:    false,
		DoubleFmt:  theCfg.doubleFmt,
		IdToDigest: map[int]string{},
		DigestToId: map[string]int{},
	}
	cvt, err := csvCvt.IdToCodeCell(meta, tableName)

	// find model run id by digest-or-stamp-or-name
	r, ok := mc.CompletedRunByDigestOrStampOrName(dn, rdsn)
	if !ok {
		return nil, 0, nil, false // return empty result: run select error
	}
	if r.Status != db.DoneRunStatus {
		omppLog.Log("Warning: model run not completed successfully: ", rdsn, ": ", r.Status)
		return nil, 0, nil, false
	}
	baseRunId := r.RunId // source run id

	csvCvt.IdToDigest[r.RunId] = r.RunDigest // add base run digest to converter
	csvCvt.DigestToId[r.RunDigest] = r.RunId

	// check if all additional model runs completed sucessfully
	runIds := []int{}

	if len(runLst) > 0 {
		rLst, _ := mc.RunRowListByModel(meta.Model.Digest)

		for _, rn := range runLst {

			rId := 0
			for k := 0; rId <= 0 && k < len(rLst); k++ {

				if rn == rLst[k].RunDigest || rn == rLst[k].RunStamp || rn == rLst[k].Name {
					rId = rLst[k].RunId
				}
				if rId > 0 {
					if rLst[k].Status != db.DoneRunStatus {
						omppLog.Log("Warning: model run not completed successfully: ", rLst[k].RunDigest, ": ", rLst[k].Status)
						return nil, 0, nil, false
					}
					csvCvt.IdToDigest[rLst[k].RunId] = rLst[k].RunDigest // add run digest to converter
					csvCvt.DigestToId[rLst[k].RunDigest] = rLst[k].RunId
				}
			}
			if rId <= 0 {
				omppLog.Log("Warning: model run not found: ", rn)
				continue
			}
			// else: model run completed successfully, include run id into the list of additional runs

			isFound := false
			for k := 0; !isFound && k < len(runIds); k++ {
				isFound = rId == runIds[k]
			}
			if !isFound {
				runIds = append(runIds, rId)
			}
		}
	}

	if err != nil {
		omppLog.Log("Failed to create output table cell id's to code converter: ", tableName, ": ", err.Error())
		return nil, 0, nil, false
	}

	return cvt, baseRunId, runIds, true
}

// TableToCodeCalcCellConverter return calculate layout for all expressions by aggregation name: sum avg count min max var sd se cv
func (mc *ModelCatalog) TableAllExprCalculateLayout(dn string, name string, aggr string) ([]db.CalculateTableLayout, bool) {

	// check aggregation operation
	fnc := ""
	switch aggr {
	case "sum":
		fnc = "OM_SUM"
	case "avg":
		fnc = "OM_AVG"
	case "count":
		fnc = "OM_COUNT"
	case "min":
		fnc = "OM_MIN"
	case "max":
		fnc = "OM_MAX"
	case "var":
		fnc = "OM_VAR"
	case "sd":
		fnc = "OM_SD"
	case "se":
		fnc = "OM_SE"
	case "cv":
		fnc = "OM_CV"
	default:
		omppLog.Log("Error: invalid aggregation name: ", aggr, ": ", dn, ": ", name)
		return []db.CalculateTableLayout{}, false
	}

	// if model digest-or-name is empty then return empty results
	if dn == "" {
		omppLog.Log("Warning: invalid (empty) model digest and name")
		return []db.CalculateTableLayout{}, false
	}

	// get model metadata and database connection
	meta, _, ok := mc.modelMeta(dn)
	if !ok {
		omppLog.Log("Warning: model digest or name not found: ", dn)
		return []db.CalculateTableLayout{}, false // return empty result: model not found or error
	}

	// find output table by name
	idx, ok := meta.OutTableByName(name)
	if !ok {
		omppLog.Log("Error: model output table not found: ", dn, ": ", name)
		return []db.CalculateTableLayout{}, false
	}
	table := &meta.Table[idx]

	// calculate output table expression and aggregation expression
	// for example: if Expr1 = AVG(acc1) and aggregation function is SD()
	// then append to calculation: Expr1 and SD(acc1)
	calcLt := []db.CalculateTableLayout{}

	for _, ex := range table.Expr {

		// append output table expression
		calcLt = append(calcLt, db.CalculateTableLayout{
			CalculateLayout: db.CalculateLayout{
				Calculate: ex.Name,
				CalcId:    ex.ExprId,
			},
			IsAggr: false,
		})

		for _, acc := range table.Acc {

			// find derived accumulator with the same name as expression name
			if acc.IsDerived && acc.Name == ex.Name {

				calcLt = append(calcLt, db.CalculateTableLayout{
					CalculateLayout: db.CalculateLayout{
						Calculate: fnc + "(" + acc.SrcAcc + ")",
						CalcId:    ex.ExprId + db.CALCULATED_ID_OFFSET,
					},
					IsAggr: true,
				})
				break
			}
		}
	}

	return calcLt, true
}

// MicrodataCellConverter return microdata value converter between code cell and id's cell.
// If isToId true then from code to id cell else other way around.
// Return model run id, entity generation digest, microdata value converter and boolean Ok flag.
func (mc *ModelCatalog) MicrodataCellConverter(
	isToId bool, dn string, rdsn string, name string,
) (
	int, string, func(interface{}) (interface{}, error), bool,
) {

	// validate parameters and return empty results on empty input
	if dn == "" {
		omppLog.Log("Warning: invalid (empty) model digest and name")
		return 0, "", nil, false
	}
	if rdsn == "" {
		omppLog.Log("Warning: invalid (empty) model run digest, stamp and name")
		return 0, "", nil, false
	}
	if name == "" {
		omppLog.Log("Warning: invalid (empty) model entity name")
		return 0, "", nil, false
	}

	// get model metadata and database connection
	meta, _, ok := mc.modelMeta(dn)
	if !ok {
		omppLog.Log("Warning: model digest or name not found: ", dn)
		return 0, "", nil, false // return empty result: model not found or error
	}

	// get run_lst db row by digest, stamp or run name
	r, ok := mc.CompletedRunByDigestOrStampOrName(dn, rdsn)
	if !ok {
		return 0, "", nil, false // run not found or not completed
	}
	if r.Status != db.DoneRunStatus {
		omppLog.Log("Warning: model run not completed successfully: ", rdsn, ": ", r.Status)
		return 0, "", nil, false
	}

	// find entity generation by entity name
	entGen, ok := mc.EntityGenByName(dn, r.RunId, name)
	if !ok {
		return r.RunId, "", nil, false // entity generation not found
	}

	// create converter
	cvtMicro := &db.CellMicroConverter{
		ModelDef:  meta,
		Name:      name,
		EntityGen: entGen,
		IsIdCsv:   isToId,
		DoubleFmt: theCfg.doubleFmt,
	}
	var cvt func(interface{}) (interface{}, error)
	var err error

	if isToId {
		cvt, err = cvtMicro.CodeToIdCell(meta, name)
	} else {
		cvt, err = cvtMicro.IdToCodeCell(meta, name)
	}
	if err != nil {
		omppLog.Log("Failed to create microdata cell value converter: ", name, ": ", err.Error())
		return r.RunId, "", nil, false
	}

	return r.RunId, entGen.GenDigest, cvt, true
}

// ParameterToCsvConverter return csv header as string array, parameter csv converter and boolean Ok flag.
func (mc *ModelCatalog) ParameterToCsvConverter(dn string, isCode bool, name string) ([]string, func(interface{}, []string) error, bool) {

	// if model digest-or-name is empty then return empty results
	if dn == "" {
		omppLog.Log("Warning: invalid (empty) model digest and name")
		return []string{}, nil, false
	}

	// get model metadata and database connection
	meta, _, ok := mc.modelMeta(dn)
	if !ok {
		omppLog.Log("Warning: model digest or name not found: ", dn)
		return []string{}, nil, false // return empty result: model not found or error
	}

	// check if parameter name exist in the model
	if _, ok = meta.ParamByName(name); !ok {
		omppLog.Log("Error: model parameter not found: ", dn, ": ", name)
		return []string{}, nil, false // return empty result: parameter not found or error
	}

	// make csv header
	csvCvt := db.CellParamConverter{
		ModelDef:  meta,
		Name:      name,
		IsIdCsv:   !isCode,
		DoubleFmt: theCfg.doubleFmt,
	}

	hdr, err := csvCvt.CsvHeader()
	if err != nil {
		omppLog.Log("Failed to make parameter csv header: ", dn, ": ", name, ": ", err.Error())
		return []string{}, nil, false
	}

	// create converter from db cell into csv row []string
	var cvt func(interface{}, []string) error

	if isCode {
		cvt, err = csvCvt.ToCsvRow()
	} else {
		cvt, err = csvCvt.ToCsvIdRow()
	}
	if err != nil {
		omppLog.Log("Failed to create parameter converter to csv: ", dn, ": ", name, ": ", err.Error())
		return []string{}, nil, false
	}

	return hdr, cvt, true
}

// TableToCsvConverter return csv header as starting array , output table cell to csv converter and and boolean Ok flag.
func (mc *ModelCatalog) TableToCsvConverter(dn string, isCode bool, name string, isAcc, isAllAcc bool) ([]string, func(interface{}, []string) error, bool) {

	// if model digest-or-name is empty then return empty results
	if dn == "" {
		omppLog.Log("Warning: invalid (empty) model digest and name")
		return []string{}, nil, false
	}

	// get model metadata and database connection
	meta, _, ok := mc.modelMeta(dn)
	if !ok {
		omppLog.Log("Warning: model digest or name not found: ", dn)
		return []string{}, nil, false // return empty result: model not found or error
	}

	// check if output table name exist in the model
	if _, ok = meta.OutTableByName(name); !ok {
		omppLog.Log("Error: model output table not found: ", dn, ": ", name)
		return []string{}, nil, false // return empty result: output table not found or error
	}

	// set cell conveter to csv
	ctc := db.CellTableConverter{
		ModelDef: meta,
		Name:     name,
	}
	var csvCvt db.CsvConverter

	switch {
	case isAcc && isAllAcc:
		csvCvt = &db.CellAllAccConverter{
			CellTableConverter: ctc,
			IsIdCsv:            !isCode,
			DoubleFmt:          theCfg.doubleFmt,
			ValueName:          "",
		}
	case isAcc:
		csvCvt = &db.CellAccConverter{
			CellTableConverter: ctc,
			IsIdCsv:            !isCode,
			DoubleFmt:          theCfg.doubleFmt,
		}
	default:
		csvCvt = &db.CellExprConverter{
			CellTableConverter: ctc,
			IsIdCsv:            !isCode,
			DoubleFmt:          theCfg.doubleFmt,
		}
	}

	// make csv header
	hdr, err := csvCvt.CsvHeader()
	if err != nil {
		omppLog.Log("Failed to make output table csv header: ", dn, ": ", name, ": ", err.Error())
		return []string{}, nil, false
	}

	// create converter from db cell into csv row []string
	var cvt func(interface{}, []string) error

	if isCode {
		cvt, err = csvCvt.ToCsvRow()
	} else {
		cvt, err = csvCvt.ToCsvIdRow()
	}
	if err != nil {
		omppLog.Log("Failed to create output table converter to csv: ", dn, ": ", name, ": ", err.Error())
		return []string{}, nil, false
	}

	return hdr, cvt, true
}

// MicrodataToCsvConverter return model run id, entity generation digest,
// csv header as starting array, microdata cell to csv converter and boolean Ok flag.
func (mc *ModelCatalog) MicrodataToCsvConverter(
	dn string, isCode bool, rdsn, name string,
) (
	int, string, []string, func(interface{}, []string) error, bool,
) {

	// validate parameters and return empty results on empty input
	if dn == "" {
		omppLog.Log("Warning: invalid (empty) model digest and name")
		return 0, "", []string{}, nil, false
	}
	if rdsn == "" {
		omppLog.Log("Warning: invalid (empty) model run digest, stamp and name")
		return 0, "", []string{}, nil, false
	}
	if name == "" {
		omppLog.Log("Warning: invalid (empty) model entity name")
		return 0, "", []string{}, nil, false
	}

	// get model metadata and database connection
	meta, _, ok := mc.modelMeta(dn)
	if !ok {
		omppLog.Log("Warning: model digest or name not found: ", dn)
		return 0, "", []string{}, nil, false // return empty result: model not found or error
	}

	// get run_lst db row by digest, stamp or run name
	r, ok := mc.CompletedRunByDigestOrStampOrName(dn, rdsn)
	if !ok {
		return 0, "", []string{}, nil, false // run not found or not completed
	}
	if r.Status != db.DoneRunStatus {
		omppLog.Log("Warning: model run not completed successfully: ", rdsn, ": ", r.Status)
		return r.RunId, "", []string{}, nil, false
	}

	// find entity generation by entity name
	entGen, ok := mc.EntityGenByName(dn, r.RunId, name)
	if !ok {
		return 0, "", []string{}, nil, false // entity generation not found
	}

	// make csv header
	cvtMicro := &db.CellMicroConverter{
		ModelDef:  meta,
		Name:      name,
		EntityGen: entGen,
		IsIdCsv:   !isCode,
		DoubleFmt: theCfg.doubleFmt,
	}

	hdr, err := cvtMicro.CsvHeader()
	if err != nil {
		omppLog.Log("Failed to make microdata csv header: ", dn, ": ", name, ": ", err.Error())
		return r.RunId, "", []string{}, nil, false
	}

	// create converter from db cell into csv row []string
	var cvt func(interface{}, []string) error

	if isCode {
		cvt, err = cvtMicro.ToCsvRow()
	} else {
		cvt, err = cvtMicro.ToCsvIdRow()
	}
	if err != nil {
		omppLog.Log("Failed to create microdata converter to csv: ", dn, ": ", name, ": ", err.Error())
		return r.RunId, "", []string{}, nil, false
	}

	return r.RunId, entGen.GenDigest, hdr, cvt, true
}
