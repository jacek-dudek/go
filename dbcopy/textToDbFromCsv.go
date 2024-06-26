// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package main

import (
	"database/sql"
	"encoding/csv"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/openmpp/go/ompp/db"
	"github.com/openmpp/go/ompp/helper"
)

// writeParamFromCsvFile read parameter csv file and write into db parameter value table.
func writeParamFromCsvFile(
	dbConn *sql.DB,
	modelDef *db.ModelMeta,
	layout db.WriteParamLayout,
	csvDir string,
	csvCvt db.CellParamConverter,
) error {

	// converter from csv row []string to db cell
	cvt, err := csvCvt.ToCell()
	if err != nil {
		return errors.New("invalid converter from csv row: " + err.Error())
	}

	// open csv file, convert to utf-8 and parse csv into db cells
	// reading from .id.csv files not supported by converters
	fn, err := csvCvt.CsvFileName()
	if err != nil {
		return errors.New("invalid csv file name: " + err.Error())
	}
	chs, err := csvCvt.CsvHeader()
	if err != nil {
		return errors.New("Error at building csv parameter header " + layout.Name + ": " + err.Error())
	}
	ch := strings.Join(chs, ",")

	f, err := os.Open(filepath.Join(csvDir, fn))
	if err != nil {
		return errors.New("csv file open error: " + err.Error())
	}
	defer f.Close()

	from, err := makeFromCsvReader(fn, f, ch, cvt)
	if err != nil {
		return errors.New("fail to create parameter csv reader: " + err.Error())
	}

	// write each csv row into parameter table
	err = db.WriteParameterFrom(dbConn, modelDef, &layout, from)
	if err != nil {
		return err
	}

	return nil
}

// writeTableFromCsvFiles read output table csv files (accumulators and expressions) and write it into db output tables.
func writeTableFromCsvFiles(
	dbConn *sql.DB,
	modelDef *db.ModelMeta,
	layout db.WriteTableLayout,
	csvDir string,
	cvtExpr db.CellExprConverter,
	cvtAcc db.CellAccConverter) error {

	// accumulator converter from csv row []string to db cell
	aToCell, err := cvtAcc.ToCell()
	if err != nil {
		return errors.New("invalid converter from accumulators csv row: " + err.Error())
	}

	// open accumulators csv file
	aFn, err := cvtAcc.CsvFileName()
	if err != nil {
		return errors.New("invalid accumulators csv file name: " + err.Error())
	}
	ahs, err := cvtAcc.CsvHeader()
	if err != nil {
		return errors.New("Error at building csv accumulators header " + layout.Name + ": " + err.Error())
	}
	ah := strings.Join(ahs, ",")

	accFile, err := os.Open(filepath.Join(csvDir, aFn))
	if err != nil {
		return errors.New("accumulators csv file open error: " + err.Error())
	}
	defer accFile.Close()

	accFrom, err := makeFromCsvReader(aFn, accFile, ah, aToCell)
	if err != nil {
		return errors.New("fail to create accumulators csv reader: " + err.Error())
	}

	// expression converter from csv row []string to db cell
	eToCell, err := cvtExpr.ToCell()
	if err != nil {
		return errors.New("invalid converter from expressions csv row: " + err.Error())
	}

	// open expressions csv file
	eFn, err := cvtExpr.CsvFileName()
	if err != nil {
		return errors.New("invalid expressions csv file name: " + err.Error())
	}
	ehs, err := cvtExpr.CsvHeader()
	if err != nil {
		return errors.New("Error at building csv expressions header " + layout.Name + ": " + err.Error())
	}
	eh := strings.Join(ehs, ",")

	exprFile, err := os.Open(filepath.Join(csvDir, eFn))
	if err != nil {
		return errors.New("expressions csv file open error: " + err.Error())
	}
	defer exprFile.Close()

	exprFrom, err := makeFromCsvReader(eFn, exprFile, eh, eToCell)
	if err != nil {
		return errors.New("fail to create expressions csv reader: " + err.Error())
	}

	// write each accumulator(s) csv rows into accumulator(s) output table
	// write each expression(s) csv rows into expression(s) output table
	err = db.WriteOutputTableFrom(dbConn, modelDef, &layout, accFrom, exprFrom)
	if err != nil {
		return err
	}

	return nil
}

// writeMicroFromCsvFile read microdata csv file and write into db enity generation value table.
func writeMicroFromCsvFile(
	dbConn *sql.DB,
	dbFacet db.Facet,
	modelDef *db.ModelMeta,
	runMeta *db.RunMeta,
	layout db.WriteMicroLayout,
	csvDir string,
	csvCvt db.CellMicroConverter,
) error {

	// converter from csv row []string to db cell
	cvt, err := csvCvt.ToCell()
	if err != nil {
		return errors.New("invalid converter from csv row: " + err.Error())
	}

	// open csv file, convert to utf-8 and parse csv into db cells
	// reading from .id.csv files not supported by converters
	fn, err := csvCvt.CsvFileName()
	if err != nil {
		return errors.New("invalid csv file name: " + err.Error())
	}
	chs, err := csvCvt.CsvHeader()
	if err != nil {
		return errors.New("Error at building csv microdata header " + layout.Name + ": " + err.Error())
	}
	ch := strings.Join(chs, ",")

	f, err := os.Open(filepath.Join(csvDir, fn))
	if err != nil {
		return errors.New("csv file open error: " + err.Error())
	}
	defer f.Close()

	from, err := makeFromCsvReader(fn, f, ch, cvt)
	if err != nil {
		return errors.New("fail to create microdata csv reader: " + err.Error())
	}

	// write each csv row into microdata entity generation table
	err = db.WriteMicrodataFrom(dbConn, dbFacet, modelDef, runMeta, &layout, from)
	if err != nil {
		return err
	}

	return nil
}

// return closure to iterate over csv file rows
func makeFromCsvReader(
	fileName string, csvFile *os.File, csvHeader string, csvToCell func(row []string) (interface{}, error),
) (func() (interface{}, error), error) {

	// create csv reader from utf-8 line
	uRd, err := helper.Utf8Reader(csvFile, theCfg.encodingName)
	if err != nil {
		return nil, errors.New("fail to create utf-8 converter: " + err.Error())
	}

	csvRd := csv.NewReader(uRd)
	csvRd.TrimLeadingSpace = true
	csvRd.ReuseRecord = true

	// skip header line
	fhs, e := csvRd.Read()
	switch {
	case e == io.EOF:
		return nil, errors.New("invalid (empty) csv file: " + fileName)
	case err != nil:
		return nil, errors.New("csv file read error: " + fileName + ": " + err.Error())
	}
	fh := strings.Join(fhs, ",")
	if strings.HasPrefix(fh, string(helper.Utf8bom)) {
		fh = fh[len(helper.Utf8bom):]
	}
	if fh != csvHeader {
		return nil, errors.New("Invalid csv file header " + fileName + ": " + fh + " expected: " + csvHeader)
	}

	// convert each csv line into cell (id cell)
	// reading from .id.csv files not supported by converters
	from := func() (interface{}, error) {
		row, err := csvRd.Read()
		switch {
		case err == io.EOF:
			return nil, nil // eof
		case err != nil:
			return nil, errors.New("csv file read error: " + fileName + ": " + err.Error())
		}

		// convert csv line to cell and return from reader
		c, err := csvToCell(row)
		if err != nil {
			return nil, errors.New("csv file row convert error: " + fileName + ": " + err.Error())
		}
		return c, nil
	}
	return from, nil
}
